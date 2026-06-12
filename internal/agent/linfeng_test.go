package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/text2sql"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTraceLinFengQuery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("../data/opentether.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	q := "林烽五月份出了多少单"
	fmt.Println("=============================================")
	fmt.Printf("测试查询: %s\n", q)
	fmt.Println("=============================================")

	// Step 1: parseText2SQLTemplateIntent
	fmt.Println("\n--- STEP 1: FastPath 意图解析 ---")
	metric, tr, emp := parseTestIntent(q)
	fmt.Printf("metric=%s timeRange=%s employee=%s\n", metric, tr, emp)

	// Step 2: check data sources
	fmt.Println("\n--- STEP 2: 数据源 ---")
	type DS struct {
		ID, Name, Host, Port, Database, SchemaInfo, TableRelations string
		Enabled                                                    bool
	}
	var dss []DS
	db.Raw("SELECT id,name,host,port,database,enabled,schema_info,table_relations FROM data_sources").Scan(&dss)
	var enabled *DS
	for _, ds := range dss {
		if ds.Enabled {
			enabled = &ds
		}
	}
	if enabled == nil {
		fmt.Println("❌ 无启用的数据源")
		return
	}
	fmt.Printf("数据源: %s (schema=%d bytes)\n", enabled.Name, len(enabled.SchemaInfo))
	fmt.Printf("表关系: %s\n", enabled.TableRelations)

	// Step 3: check skills
	fmt.Println("\n--- STEP 3: Text2SQL Skills ---")
	var skills []models.Skill
	db.Where("skill_type = 'text2sql' AND enabled = 1").Find(&skills)
	for _, sk := range skills {
		fmt.Printf("Skill: %s | keywords=%s\n", sk.Name, sk.Keywords)
		// check for MD
		if strings.Contains(sk.Config, "context_md") {
			fmt.Printf("  → 有 MD 文档 (len=%d)\n", len(sk.Config))
		} else {
			fmt.Printf("  → 无 MD 文档（走实时 schema）\n")
		}
	}

	// Step 4: scoreSkills simulation
	fmt.Println("\n--- STEP 4: 技能打分 (scoreSkills) ---")
	lowerMsg := strings.ToLower(q)
	var allSkills []models.Skill
	db.Where("enabled = 1").Find(&allSkills)
	candidates := scoreSkills(q, lowerMsg, allSkills)
	for i, c := range candidates {
		fmt.Printf("  %d. %s (%s) score=%.2f\n", i+1, c.SkillName, c.SkillType, c.Score)
	}

	// Step 5: selectFallbackTables
	fmt.Println("\n--- STEP 5: Schema 表选择 ---")
	t2s := text2sql.New(db, nil)
	schema, err := t2s.GetDataSourceSchema(enabled.ID)
	if err != nil {
		fmt.Printf("获取 schema 失败: %v\n", err)
		return
	}
	tables := parseSchemaTables(schema)
	fmt.Printf("总表数: %d\n", len(tables))

	// find matching tables
	type tableScore struct {
		Name  string
		Score int
	}
	var scored []tableScore
	for _, t := range tables {
		s := 0
		n := strings.ToLower(t.Name)
		if strings.Contains(n, "order") {
			s += 10
		}
		if strings.Contains(n, "sale") {
			s += 10
		}
		if strings.Contains(n, "profile") {
			s += 10
		}
		if strings.Contains(n, "staff") {
			s += 5
		}
		if strings.Contains(n, "product") {
			s += 8
		}
		if strings.Contains(n, "customer") {
			s += 8
		}
		if strings.Contains(n, "user") {
			s += 3
		}
		if strings.Contains(n, "work_order") {
			s += 10
		}
		if s > 0 {
			scored = append(scored, tableScore{t.Name, s})
		}
	}
	// sort
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	limit := 10
	if limit > len(scored) {
		limit = len(scored)
	}
	fmt.Printf("selectFallbackTables 选出 %d 张表:\n", limit)
	for i := 0; i < limit; i++ {
		fmt.Printf("  %d. %s (score=%d)\n", i+1, scored[i].Name, scored[i].Score)
	}

	// Step 6: check runtime memories
	fmt.Println("\n--- STEP 6: 运行时模板 ---")
	type Mem struct {
		Key        string
		Confidence float64
		Source     string
		Content    string
	}
	var mems []Mem
	key := "employee_metric_by_time_range:" + metric
	db.Raw("SELECT key,confidence,source,SUBSTR(content,1,200) as content FROM skill_runtime_memories WHERE type='text2sql_template' AND key=? AND (source='admin' OR confidence>=0.9) LIMIT 1", key).Scan(&mems)
	if len(mems) > 0 {
		fmt.Printf("✅ 已审批模板: key=%s conf=%.2f src=%s\n", mems[0].Key, mems[0].Confidence, mems[0].Source)
		fmt.Printf("   content: %.200s\n", mems[0].Content)
	} else {
		fmt.Println("❌ 无已审批模板 → 走完整 text2sql 流程")
	}

	// Step 7: final verdict
	fmt.Println("\n=============================================")
	fmt.Println("结论")
	fmt.Println("=============================================")
	fmt.Printf("意图: metric=%s time=%s employee=%s\n", metric, tr, emp)
	fmt.Printf("路由: scoreSkills top=%s (%.2f)\n", candidates[0].SkillName, candidates[0].Score)
	fmt.Printf("选中表: %d 张业务表\n", limit)
	fmt.Printf("有MD文档: %v\n", len(skills) > 0 && strings.Contains(skills[0].Config, "context_md"))
	fmt.Printf("有已审批模板: %v\n", len(mems) > 0)
}

func parseTestIntent(q string) (metric, timeRange, employee string) {
	m := strings.TrimSpace(q)
	if strings.Contains(m, "销售额") || strings.Contains(m, "金额") {
		metric = "销售额"
	} else if strings.Contains(m, "多少单") || strings.Contains(m, "订单数") || strings.Contains(m, "订单数量") || strings.Contains(m, "订单量") || strings.Contains(m, "卖了") || strings.Contains(m, "出了") || strings.Contains(m, "出单") || strings.Contains(m, "销量") || strings.Contains(m, "下单") {
		metric = "订单数"
	}
	timeTokens := []string{"上个季度", "上季度", "本季度", "这个季度", "上个月", "上月", "本月", "当前", "今天", "今年", "去年"}
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

func parseSchemaTables(schema string) []struct{ Name string } {
	var tables []struct{ Name string }
	for _, line := range strings.Split(schema, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "表:") {
			tables = append(tables, struct{ Name string }{strings.TrimSpace(strings.TrimPrefix(t, "表:"))})
		}
	}
	return tables
}
