package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	cfg.Database.Type = "sqlite"
	cfg.Database.Name = "data/opentether.db"
	db, err := gorm.Open(sqlite.Open(cfg.Database.Name), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	q := "林烽五月份出了多少单"

	// ====== STEP 1: FastPath 意图解析 ======
	fmt.Println("=============================================")
	fmt.Println("STEP 1: parseText2SQLTemplateIntent")
	fmt.Printf("  输入: %s\n", q)

	m := strings.ToLower(q)
	metric, tr, emp := parseTemplateIntent(q)
	fmt.Printf("  metric=%s, timeRange=%s, employee=%s\n", metric, tr, emp)

	// ====== STEP 2: 查是否有 approved template ======
	fmt.Println("\n=============================================")
	fmt.Println("STEP 2: 查已审批 Text2SQL 模板")
	key := fmt.Sprintf("employee_metric_by_time_range:%s", metric)
	fmt.Printf("  查询 key=%s, source=admin OR confidence>=0.9\n", key)
	type Mem struct {
		Key, Source, Content string
		Confidence           float64
	}
	var mems []Mem
	db.Raw("SELECT key, confidence, source, content FROM skill_runtime_memories WHERE type='text2sql_template' AND key=? AND (source='admin' OR confidence>=0.9) LIMIT 1", key).Scan(&mems)
	if len(mems) > 0 {
		fmt.Printf("  ✅ 找到已审批模板: source=%s conf=%.2f\n", mems[0].Source, mems[0].Confidence)
		fmt.Printf("  SQL模板: %.120s\n", mems[0].Content)
	} else {
		fmt.Println("  ❌ 未找到已审批模板 → 回退到完整 text2sql 流程")
	}

	// ====== STEP 3: data source 信息 ======
	fmt.Println("\n=============================================")
	fmt.Println("STEP 3: 数据源配置")
	type DS struct {
		ID, Name, SourceType, Host, Port, Database, SchemaInfo, TableRelations string
		Enabled                                                                bool
	}
	var dss []DS
	db.Raw("SELECT id, name, source_type, host, port, database, enabled, schema_info, table_relations FROM data_sources").Scan(&dss)
	var enabledDS *DS
	for _, ds := range dss {
		if ds.Enabled {
			enabledDS = &ds
			fmt.Printf("  name=%s type=%s host=%s port=%s db=%s\n", ds.Name, ds.SourceType, ds.Host, ds.Port, ds.Database)
			fmt.Printf("  schema_info len=%d bytes\n", len(ds.SchemaInfo))
			fmt.Printf("  table_relations: %s\n", ds.TableRelations)
		}
	}
	if enabledDS == nil {
		fmt.Println("  ❌ 无启用的数据源！")
		return
	}

	// ====== STEP 4: Skill MD 文档检查 ======
	fmt.Println("\n=============================================")
	fmt.Println("STEP 4: Text2SQL Skill 上下文（MD文档）")
	type Skill struct{ Name, Config, Category string }
	var skills []Skill
	db.Where("skill_type='text2sql' AND enabled=1").Find(&skills)

	mdContext := ""
	for _, sk := range skills {
		var cfg map[string]interface{}
		if json.Unmarshal([]byte(sk.Config), &cfg) == nil {
			if md, ok := cfg["context_md"].(string); ok && strings.TrimSpace(md) != "" {
				mdContext = md
				fmt.Printf("  Skill '%s': ✅ inline context_md (%d bytes)\n", sk.Name, len(md))
				break
			}
			if url, ok := cfg["context_md_url"].(string); ok && url != "" {
				fmt.Printf("  Skill '%s': ✅ context_md_url=%s\n", sk.Name, url)
				mdContext = "(external MD from " + url + ")"
				break
			}
			fmt.Printf("  Skill '%s': ❌ 没有 MD 文档\n", sk.Name)
		}
	}

	if mdContext == "" {
		fmt.Println("  → 没有 MD 文档，将使用实时 schema")
	} else {
		fmt.Println("  → 有 MD 文档，将替换实时 schema！")
	}

	// ====== STEP 5: Schema 选择 ======
	fmt.Println("\n=============================================")
	fmt.Println("STEP 5: Schema 选择 (selectRelevantSchema + selectFallbackTables)")

	schemaText := enabledDS.SchemaInfo
	tables := parseSchema(schemaText)
	fmt.Printf("  总表数: %d\n", len(tables))

	// 用 chineseToken 匹配
	matchByChinese := map[string]int{}
	for _, t := range tables {
		score := 0
		name := strings.ToLower(t)
		if strings.Contains(name, "order") {
			score += 10
		}
		if strings.Contains(name, "sale") {
			score += 10
		}
		if strings.Contains(name, "profile") {
			score += 10
		}
		if strings.Contains(name, "staff") {
			score += 5
		}
		if strings.Contains(name, "employee") {
			score += 5
		}
		if strings.Contains(name, "product") {
			score += 8
		}
		if strings.Contains(name, "customer") {
			score += 8
		}
		if strings.Contains(name, "goods") {
			score += 8
		}
		if strings.Contains(name, "pay") {
			score += 8
		}
		if strings.Contains(name, "user") {
			score += 3
		}
		if score > 0 {
			matchByChinese[t] = score
		}
	}

	// 排序
	type cand struct {
		Name  string
		Score int
	}
	var clist []cand
	for n, s := range matchByChinese {
		clist = append(clist, cand{n, s})
	}
	sort.Slice(clist, func(i, j int) bool { return clist[i].Score > clist[j].Score })

	limit := 10
	fmt.Printf("  selectFallbackTables 选出 %d 张表:\n", min(limit, len(clist)))
	hasProfile, hasOrder := false, false
	for i := 0; i < limit && i < len(clist); i++ {
		fmt.Printf("    %d. %s (score=%d)\n", i+1, clist[i].Name, clist[i].Score)
		if strings.Contains(strings.ToLower(clist[i].Name), "profile") {
			hasProfile = true
		}
		if strings.Contains(strings.ToLower(clist[i].Name), "order") {
			hasOrder = true
		}
	}

	// ====== STEP 6: 如果 MD 存在，检查 MD 内容 ======
	fmt.Println("\n=============================================")
	fmt.Println("STEP 6: 最终结论")

	if mdContext != "" && strings.TrimSpace(mdContext) != "" && !strings.HasPrefix(mdContext, "(external") {
		// MD 存在，检查是否包含关键表
		hasProfileMD := strings.Contains(strings.ToLower(mdContext), "t_profile") ||
			strings.Contains(strings.ToLower(mdContext), "profile") ||
			strings.Contains(strings.ToLower(mdContext), "员工")
		hasOrderMD := strings.Contains(strings.ToLower(mdContext), "t_order") ||
			strings.Contains(strings.ToLower(mdContext), "order") ||
			strings.Contains(strings.ToLower(mdContext), "订单")
		hasJoinMD := strings.Contains(strings.ToLower(mdContext), "user_id") ||
			strings.Contains(strings.ToLower(mdContext), "join") ||
			strings.Contains(strings.ToLower(mdContext), "关联") ||
			strings.Contains(strings.ToLower(mdContext), "外键")

		fmt.Printf("  MD文档包含员工表: %v\n", hasProfileMD)
		fmt.Printf("  MD文档包含订单表: %v\n", hasOrderMD)
		fmt.Printf("  MD文档包含join关系: %v\n", hasJoinMD)

		if !hasProfileMD || !hasOrderMD {
			fmt.Println("\n  ❌ MD 文档不包含必要的表结构！")
			fmt.Println("  → 建议：在 Skills 编辑页点击「生成 MD」或清空 MD 让系统读取实时 schema")
		} else if !hasJoinMD {
			fmt.Println("\n  ⚠️ MD 文档有表但缺少 JOIN 关系说明")
			fmt.Println("  → 建议：在 MD 中补充 t_profile.user_id = t_order.user_id")
		} else {
			fmt.Println("\n  ✅ MD 文档包含完整信息，LLM 应能生成查询")
		}
	} else if mdContext != "" && strings.HasPrefix(mdContext, "(external") {
		fmt.Println("  MD 文档在外部存储，无法内联检查")
		fmt.Println("  → 请确认外部 MD 包含: t_profile(员工表) + t_order(订单表) + user_id(关联字段)")
	} else {
		// 没有 MD，走实时 schema
		if hasProfile && hasOrder {
			fmt.Println("  ✅ 没有MD文档 → 使用实时schema")
			fmt.Printf("  ✅ 实时schema已选出员工表(t_profile)和订单表\n")
			fmt.Println("  ✅ LLM 应能生成: SELECT COUNT(*) FROM t_order WHERE user_id IN (SELECT id FROM t_profile WHERE name LIKE '%林烽%') AND create_time BETWEEN '2026-05-01' AND '2026-06-01'")
		} else {
			fmt.Printf("  ❌ 实时schema中 员工表=%v 订单表=%v\n", hasProfile, hasOrder)
			fmt.Println("  → 缺少关键表，请检查数据源配置的 table_relations")
		}
	}

	// Runtime memories 检查
	fmt.Println("\n=============================================")
	fmt.Println("STEP 7: 运行时记忆 (可能干扰)")
	var allMems []Mem
	db.Raw("SELECT key, confidence, source, SUBSTR(content,1,120) as content FROM skill_runtime_memories WHERE type='text2sql_template' ORDER BY confidence DESC LIMIT 5").Scan(&allMems)
	if len(allMems) > 0 {
		for _, m := range allMems {
			fmt.Printf("  %s conf=%.2f src=%s\n", m.Key, m.Confidence, m.Source)
		}
	} else {
		fmt.Println("  无运行时记忆（不会干扰）")
	}

	fmt.Println("\n=============================================")
	_ = os.Stdout
	_ = agent.NewAgentEngine(nil, nil, nil, nil)
}

func parseTemplateIntent(q string) (metric, timeRange, employee string) {
	m := strings.TrimSpace(q)
	if strings.Contains(m, "销售额") || strings.Contains(m, "金额") {
		metric = "销售额"
	} else if strings.Contains(m, "多少单") || strings.Contains(m, "订单数") || strings.Contains(m, "订单数量") || strings.Contains(m, "订单量") || strings.Contains(m, "卖了") || strings.Contains(m, "出了") || strings.Contains(m, "出单") || strings.Contains(m, "销量") || strings.Contains(m, "下单") {
		metric = "订单数"
	}
	timeTokens := []string{"上个季度", "上季度", "本季度", "这个季度", "上个月", "上月", "本月", "当前", "现在", "今天", "今年", "去年"}
	monthMap := map[string]bool{
		"一月": true, "二月": true, "三月": true, "四月": true, "五月": true, "六月": true,
		"七月": true, "八月": true, "九月": true, "十月": true, "十一月": true, "十二月": true,
		"1月": true, "2月": true, "3月": true, "4月": true, "5月": true, "6月": true,
		"7月": true, "8月": true, "9月": true, "10月": true, "11月": true, "12月": true,
	}
	for t := range monthMap {
		timeTokens = append(timeTokens, t, t+"份")
	}
	for _, t := range timeTokens {
		if strings.Contains(m, t) {
			timeRange = t
			break
		}
	}
	// 简单提取员工名（第一个2-4汉字的词）
	runes := []rune(m)
	for i := 0; i < len(runes); i++ {
		if runes[i] >= 0x4E00 && runes[i] <= 0x9FFF {
			end := i
			for end < len(runes) && runes[end] >= 0x4E00 && runes[end] <= 0x9FFF && end-i < 4 {
				end++
			}
			if end-i >= 2 {
				employee = string(runes[i:end])
			}
			break
		}
	}
	return
}

func parseSchema(s string) []string {
	var tables []string
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "表:") {
			tables = append(tables, strings.TrimSpace(strings.TrimPrefix(t, "表:")))
		}
	}
	return tables
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
