# 禁止硬编码业务语义 — AI 自举开发规范

## 原则

**平台代码只做管道，业务语义由 AI + Skill 配置驱动。**

> 任何根据"用户说的中文词"来判断"该查什么表/字段/指标"的代码，都必须替换为 LLM 解读 + Skill 语义模型。

## 禁止清单

### 1. 禁止中文关键词→表名/列名映射

❌ 不得出现：
```go
hints := map[string][]string{
    "订单": {"order", "sale", "sales"},
    "员工": {"profile", "staff", "employee"},
    "林烽": {"profile", "staff", "employee", "user_id"},
}
```

✅ 应改为：Skill 的 `selected_tables` + `entity_rules` + `table_relations` + LLM 选表。

### 2. 禁止实体/指标中文标签硬编码

❌ 不得出现：
```go
labelMap := map[string]string{
    "order": "订单", "employee": "员工", "product": "产品",
}
```

✅ 应改为：Skill 配置的 `entity_rules` 中 `entity/label` 字段。

### 3. 禁止硬编码指标槽位提取

❌ 不得出现：
```go
if strings.Contains(text, "多少单") { return "订单数" }
if strings.Contains(text, "多少钱") { return "销售额" }
```

✅ 应改为：LLM 根据 Skill 的 `metric_rules` 自行理解。

### 4. 禁止硬编码时间/主语/意图分类

❌ 不得出现：
```go
timeTokens := []string{"今年", "上月", "本月", "上季度", "昨天"}
intent := strings.Contains("订单/销售/库存/员工") ? "query" : "chat"
```

✅ 应改为：LLM 自然语言理解；仅保留**纯结构性**判断（如问号计数判断多问题）。

### 5. 禁止硬编码业务表名/列名模式

❌ 不得出现：
```go
commonPatterns := []string{"order", "sale", "profile", "staff", "product", "sku"}
```

✅ 应改为：Skill 的 `selected_tables` 配置 + LLM 决定。

### 6. 禁止硬编码权限组名

❌ 不得出现：
```go
if group.Code == "admin" || group.Code == "Administrators" { ... }
```

✅ 应改为：Skill 配置的 `full_access_groups` / `allowed_all_scope_groups`。

### 7. 禁止硬编码对话模板中的具体业务示例

❌ 不得出现：
```go
prompt := "...如「林烽上月出了多少单」..."
```

✅ 应改为：通用示例，不写具体人名/业务指标。

### 8. 禁止硬编码快捷回复

❌ 不得出现：
```go
case "你好": return "你好，我是 OpenTether AI 助手。"
case "帮助": return "你可以直接问我：查询订单数据..."
```

✅ 应改为：由 LLM 自然应答。

## 允许清单

| 类别 | 允许 | 示例 |
|------|------|------|
| 结构性判断 | 问号/感叹号计数、长度判断 | `strings.Count(msg, "？") >= 2` |
| 通用 NER | 时间格式解析（纯日期/月份格式） | `2026-03`, `January`, `Q1` |
| 管道控制 | 路由、超时、重试、熔断 | `context.WithTimeout` |
| 配置读取 | 从 Skill 配置/语义模型取值 | `cfg["metric_rules"]` |
| 安全审计 | 记录谁做了什么 | `audit(c, "update", "skill", id)` |
| 数据格式 | 字节→字符串转换、JSON 序列化 | `values[i] = string(b)` |

## 检查清单

代码审查时，若发现以下任一模式，**必须拒绝合并**：

- [ ] 代码中出现 `"订单"`、`"销售额"`、`"多少单"`、`"业绩"`、`"林烽"`、`"张三"` 等业务中文词
- [ ] 代码中出现 `"order"`、`"sale"`、`"profile"`、`"staff"`、`"product"`、`"sku"` 等表名猜测
- [ ] 代码中出现 `"admin"`、`"Administrators"`、`"sales_admin"` 等权限组名
- [ ] 代码中出现 `"员工"`、`"业绩"`、`"考勤"` 等 Skill 类型判断词
- [ ] 代码中出现 `"销售额"`、`"订单数"`、`"利润"` 等硬编码指标标签
- [ ] 代码中出现 `"查询"`、`"多少"`、`"卖"`、`"出"` 等硬编码意图分类词
- [ ] 代码中出现 `"order"`、`"sale"`、`"profile"`、`"price"`、`"amount"` 等列名模式
- [ ] 代码中出现 `"delivery_time"`、`"fee_code"`、`"sub_total"`、`"sale_staff_id"` 等具体字段名

## 迁移路径

已有硬编码的模块，按以下顺序迁移：

1. **Skill 配置补全** → 确保 `metric_rules`、`entity_rules`、`table_relations`、`full_access_groups` 等字段完整
2. **LLM 替换** → 移除代码中的 `strings.Contains` 关键词匹配，改为让 LLM 读取 Skill 上下文后自行判断
3. **测试验证** → 用真实问题验证 LLM 输出质量
4. **删除旧代码** → 确认 LLM 方案可行后删除硬编码逻辑