package text2sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/templating"
	"gorm.io/gorm"
)

// Text2SQL converts natural language to SQL queries
type Text2SQL struct {
	db             *gorm.DB
	llmClient      llm.Client
	externalDB     *sql.DB // 外部数据库连接
	ownsExternalDB bool    // true 表示由当前实例创建并负责关闭
	dataSourceID   string  // 当前使用的数据源ID
	auditSvc       AuditRecorder
}

// AuditRecorder SQL 审计记录器接口（避免循环依赖）
type AuditRecorder interface {
	RecordSQL(userID, skillID, question, sql, dataSourceID, status string) (string, error)
	MarkExecuted(auditID string, rowCount int, execTime string) error
}

// New creates a new Text2SQL instance
func New(db *gorm.DB, llmClient llm.Client) *Text2SQL {
	return &Text2SQL{
		db:        db,
		llmClient: llmClient,
	}
}

// NewWithDataSource creates a new Text2SQL instance with a data source.
// Prefer NewWithExternalDB for request paths that can reuse an application-level datasource pool.
func NewWithDataSource(db *gorm.DB, llmClient llm.Client, dataSourceID string) (*Text2SQL, error) {
	t2s := &Text2SQL{
		db:           db,
		llmClient:    llmClient,
		dataSourceID: dataSourceID,
	}

	// 如果提供了数据源ID，尝试连接
	if dataSourceID != "" {
		if err := t2s.connectToDataSource(dataSourceID); err != nil {
			return nil, err
		}
	}

	return t2s, nil
}

// NewWithExternalDB creates a Text2SQL instance using a shared external DB pool.
func NewWithExternalDB(db *gorm.DB, llmClient llm.Client, dataSourceID string, externalDB *sql.DB) *Text2SQL {
	return &Text2SQL{
		db:             db,
		llmClient:      llmClient,
		externalDB:     externalDB,
		dataSourceID:   dataSourceID,
		ownsExternalDB: false,
	}
}

// connectToDataSource 连接到指定的数据源
func (t *Text2SQL) connectToDataSource(dataSourceID string) error {
	var ds models.DataSource
	if err := t.db.Where("id = ?", dataSourceID).First(&ds).Error; err != nil {
		return fmt.Errorf("数据源不存在: %w", err)
	}

	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	db, err := database.Connect(cfg)
	if err != nil {
		return fmt.Errorf("连接数据源失败: %w", err)
	}

	t.externalDB = db
	t.ownsExternalDB = true
	t.dataSourceID = dataSourceID
	return nil
}

// Close 关闭由当前实例临时创建的外部数据库连接；共享连接池由 PoolManager 统一管理。
func (t *Text2SQL) Close() {
	if t.externalDB != nil && t.ownsExternalDB {
		t.externalDB.Close()
	}
}

// SetAuditService 设置 SQL 审计服务
func (t *Text2SQL) SetAuditService(svc AuditRecorder) {
	t.auditSvc = svc
}

// QueryRequest represents a text2sql query request
type QueryRequest struct {
	Question              string
	RawSQL                string
	DataSourceID          string
	DataSourceIDs         []string             // 多数据源联邦查询
	SchemaContext         string                 // Optional schema context
	UserID                string                 // 用户 ID（审计用）
	SkillID               string                 // Skill ID（审计用）
	IsAdmin               bool                   // 是否为管理员（管理员自动通过审批）
	IsBossMode            bool                   // 是否为 boss mode 用户（跳过 SQL 审计）
	AllowedOps            []string               // 允许的 SQL 操作前缀（为空时默认只读）
	DataBoundaryRules     []DataBoundaryRule     // 数据边界规则（用户组/用户级别的字段映射）
	UserContext           map[string]interface{} // 用户上下文字段值（如 global_user_id, company_user_id 等）
	SecureSemanticContext *SecureSemanticContext // 安全语义上下文（query_plan 模式）
	Permissions           []DataPermission       // 解析后的数据权限（系统强制执行）
}

type DataBoundaryRule struct {
	Groups        []string `json:"groups"`
	ExcludeGroups []string `json:"exclude_groups"`
	Users         []string `json:"users"`
	ExcludeUsers  []string `json:"exclude_users"`
	Table         string   `json:"table"`
	Field         string   `json:"field"`
	Operator      string   `json:"operator"`
	UserField     string   `json:"user_field"`
}

// QueryResult represents the result of a text2sql query
type QueryResult struct {
	Question      string          `json:"question"`
	SQL           string          `json:"sql"`
	Columns       []string        `json:"columns"`
	Rows          [][]interface{} `json:"rows"`
	RowCount      int             `json:"row_count"`
	Error         string          `json:"error,omitempty"`
	ExecutionTime string          `json:"execution_time"`
}

// GenerateSQL generates a SQL query from natural language
func (t *Text2SQL) GenerateSQL(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	result := &QueryResult{
		Question: req.Question,
	}

	// query_plan 模式：LLM 生成结构化查询计划，系统构建安全 SQL
	if req.SecureSemanticContext != nil && req.SecureSemanticContext.Enabled {
		_, err := t.generateQueryPlan(ctx, req, result)
		return result, err
	}

	// 获取数据源 schema - 支持多数据源联邦查询
	var schema string
	dsIDs := req.DataSourceIDs
	if len(dsIDs) == 0 && req.DataSourceID != "" {
		dsIDs = []string{req.DataSourceID}
	}

	if len(dsIDs) > 0 {
		var schemas []string
		for _, dsID := range dsIDs {
			// 联邦查询授权检查：验证数据源存在且启用
			var ds models.DataSource
			if err := t.db.Where("id = ?", dsID).First(&ds).Error; err != nil {
				result.Error = fmt.Sprintf("数据源 %s 不存在或无权访问", dsID)
				return result, fmt.Errorf("数据源 %s 不存在或无权访问: %w", dsID, err)
			}
			if !ds.Enabled {
				result.Error = fmt.Sprintf("数据源 %s 已禁用", dsID)
				return result, fmt.Errorf("数据源 %s 已禁用", dsID)
			}

			s, err := t.GetDataSourceSchema(dsID)
			if err != nil {
				result.Error = fmt.Sprintf("获取数据源 %s 失败: %v", dsID, err)
				return result, err
			}
			schemas = append(schemas, fmt.Sprintf("## 数据源: %s (%s)\n%s", ds.Name, ds.SourceType, s))
		}
		schema = strings.Join(schemas, "\n\n")
	} else {
		result.Error = "未指定数据源"
		return result, fmt.Errorf("未指定数据源")
	}

	if strings.TrimSpace(req.SchemaContext) != "" {
		// MD 文档作为业务口径补充，不替代实时表结构
		selectedSchema := t.selectRelevantSchema(req.Question, req.DataSourceID, schema)
		schema = req.SchemaContext + "\n\n## 数据库实时表结构\n" + selectedSchema
	} else {
		// 先基于缓存的 SchemaInfo/TableRelations 做候选表和字段预筛选，减少 LLM 上下文和分析成本
		schema = t.selectRelevantSchema(req.Question, req.DataSourceID, schema)
	}

	// Build prompt for LLM
	prompt := t.buildPrompt(req.Question, schema)

	// Call LLM to generate SQL (含 HTTP 级重试)
	resp, err := llm.ChatCompletionWithRetry(t.llmClient, ctx, llm.ChatRequest{
		Model: t.llmClient.GetModel(),
		Messages: []llm.Message{
			{Role: "system", Content: "You are a SQL expert. Generate SQL queries based on the schema provided. Only output the SQL query, no explanations."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1000,
		Temperature: 0.1,
	}, 3)

	if err != nil {
		result.Error = fmt.Sprintf("LLM 调用失败: %v", err)
		return result, err
	}

	// Parse SQL from response
	sqlQuery := t.parseSQL(resp.Content)
	result.SQL = sqlQuery

	return result, nil
}

// ExecuteSQL executes a SQL query and returns results
func (t *Text2SQL) ExecuteSQL(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	startTime := time.Now()

	result := &QueryResult{
		Question: req.Question,
	}

	// 如果没有外部连接，尝试连接
	if t.externalDB == nil && req.DataSourceID != "" {
		if err := t.connectToDataSource(req.DataSourceID); err != nil {
			result.Error = fmt.Sprintf("连接数据源失败: %v", err)
			return result, err
		}
	}

	// 检查是否有外部数据库连接
	if t.externalDB == nil {
		result.Error = "未配置数据源，请先配置数据源后再执行查询"
		return result, errors.New(result.Error)
	}

	if strings.TrimSpace(req.RawSQL) != "" {
		result.SQL = req.RawSQL
	} else {
		genResult, err := t.GenerateSQL(ctx, req)
		if err != nil {
			return genResult, err
		}
		if genResult.Error != "" {
			return genResult, nil
		}
		result.SQL = genResult.SQL
	}

	// Check if SQL is read-only and safe to execute
	if err := validateReadOnlySQL(result.SQL, req.AllowedOps); err != nil {
		result.Error = err.Error()
		return result, nil
	}

	// 数据边界规则：仅在旧模式（非意图构建/query_plan）时应用
	// 新模式下权限已在 BuildSQL 中嵌入
	if req.SecureSemanticContext == nil || !req.SecureSemanticContext.Enabled {
		if boundedSQL, err := applyDataBoundaryRules(result.SQL, req.DataBoundaryRules, req.UserContext); err != nil {
			result.Error = fmt.Sprintf("数据边界规则错误: %v", err)
			return result, nil
		} else {
			result.SQL = boundedSQL
		}
	}

	// Add LIMIT if missing
	result.SQL = ensureLimit(result.SQL, 1000)

	// ====== SQL 审计拦截 ======
	if t.auditSvc != nil && req.UserID != "" {
		// 管理员或 boss mode 自动通过
		if req.IsAdmin || req.IsBossMode {
			auditID, _ := t.auditSvc.RecordSQL(req.UserID, req.SkillID, req.Question, result.SQL, req.DataSourceID, "auto_approved")
			_ = t.auditSvc.MarkExecuted(auditID, 0, "")
		} else {
			// 非管理员且非 boss mode：记录 pending，返回等待审批
			auditID, auditErr := t.auditSvc.RecordSQL(req.UserID, req.SkillID, req.Question, result.SQL, req.DataSourceID, "pending")
			if auditErr == nil {
				result.Error = fmt.Sprintf("[审计 #%s] 您的 SQL 查询已提交审批，请等待管理员审批通过后再重试。", auditID)
				result.SQL = "" // 清空 SQL，防止泄露
				return result, nil
			}
		}
	} else if req.UserID != "" {
		// 审计服务未就绪：非管理员且非 boss mode 禁止执行
		if !req.IsAdmin && !req.IsBossMode {
			result.Error = "SQL 审计服务未就绪，请联系管理员。"
			result.SQL = ""
			return result, nil
		}
	}

	// Execute the SQL
	queryCtx, queryCancel := context.WithTimeout(ctx, 30*time.Second)
	defer queryCancel()
	rows, err := t.externalDB.QueryContext(queryCtx, result.SQL)
	if err != nil {
		result.Error = fmt.Sprintf("SQL 执行失败: %v", err)
		return result, nil
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		result.Error = fmt.Sprintf("获取列名失败: %v", err)
		return result, nil
	}
	result.Columns = columns

	const maxRows = 1000
	for rows.Next() {
		if len(result.Rows) >= maxRows {
			result.Error = fmt.Sprintf("结果超过 %d 行，已截断。请缩小查询范围。", maxRows)
			break
		}
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			result.Error = fmt.Sprintf("扫描行失败: %v", err)
			break
		}
		for i, value := range values {
			if b, ok := value.([]byte); ok {
				values[i] = string(b)
			}
		}
		result.Rows = append(result.Rows, values)
	}

	result.RowCount = len(result.Rows)
	result.ExecutionTime = time.Since(startTime).String()

	return result, nil
}

// validateReadOnlySQL 强制只读检查
func validateReadOnlySQL(sql string, allowedOps []string) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return fmt.Errorf("生成的 SQL 为空")
	}

	cleaned := removeComments(sql)
	cleaned = strings.TrimSpace(cleaned)

	// 移除注释后为空，说明原始 SQL 只有注释（如 "-- Unable to answer"）
	if cleaned == "" {
		return fmt.Errorf("无法根据当前数据源结构生成查询：数据表中没有匹配的字段，请检查数据源配置或联系管理员")
	}

	upper := strings.ToUpper(cleaned)

	// "ALL" 表示跳过所有只读检查
	for _, op := range allowedOps {
		if strings.EqualFold(op, "ALL") {
			return nil
		}
	}

	// 禁止多语句注入
	if strings.Count(cleaned, ";") > 0 {
		trimmed := strings.TrimRight(cleaned, "; ")
		if strings.Count(trimmed, ";") > 0 {
			return fmt.Errorf("不允许执行多条 SQL 语句")
		}
	}

	// 默认允许的只读操作
	if len(allowedOps) == 0 {
		allowedOps = []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"}
	}

	// 当允许的操作仅限于只读操作时，执行 DML/DDL/DCL 禁止关键字检查
	isReadOnly := true
	readonlySet := []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"}
	for _, op := range allowedOps {
		if !startsWithAny(strings.ToUpper(strings.TrimSpace(op)), readonlySet...) {
			isReadOnly = false
			break
		}
	}

	if isReadOnly {
		forbidden := []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "TRUNCATE", "GRANT", "REVOKE", "REPLACE", "RENAME", "CALL", "EXEC", "EXECUTE"}
		for _, kw := range forbidden {
			if containsSQLKeyword(upper, kw) {
				return fmt.Errorf("不允许执行 %s 操作，仅支持 SELECT 只读查询", kw)
			}
		}
	}

	// 必须以允许的关键字开头
	if !startsWithAny(upper, allowedOps...) {
		return fmt.Errorf("SQL 必须以 SELECT 开头，当前: %.80s", upper)
	}

	return nil
}

func removeComments(sql string) string {
	for {
		start := strings.Index(sql, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(sql[start:], "*/")
		if end == -1 {
			break
		}
		sql = sql[:start] + " " + sql[start+end+2:]
	}
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}

func containsSQLKeyword(upperSQL, keyword string) bool {
	idx := strings.Index(upperSQL, keyword)
	if idx == -1 {
		return false
	}
	before := idx == 0 || !isLetter(rune(upperSQL[idx-1]))
	after := idx+len(keyword) >= len(upperSQL) || !isLetter(rune(upperSQL[idx+len(keyword)]))
	return before && after
}

func isLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_'
}

func startsWithAny(upper string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.HasPrefix(upper, kw) {
			return true
		}
	}
	return false
}

func ensureLimit(sql string, maxLimit int) string {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	// 只对 SELECT 查询添加 LIMIT
	if !strings.HasPrefix(upper, "SELECT") {
		return sql
	}
	if strings.Contains(upper, "LIMIT") {
		return sql
	}
	sql = strings.TrimRight(strings.TrimSpace(sql), ";")
	return fmt.Sprintf("%s LIMIT %d", sql, maxLimit)
}

func applyDataBoundaryRules(sqlQuery string, rules []DataBoundaryRule, userCtx map[string]interface{}) (string, error) {
	if len(rules) == 0 || len(userCtx) == 0 {
		return sqlQuery, nil
	}
	var conditions []string
	for _, rule := range rules {
		if strings.TrimSpace(rule.Table) == "" || strings.TrimSpace(rule.Field) == "" || strings.TrimSpace(rule.UserField) == "" {
			continue
		}
		if !boundaryRuleMatches(rule, userCtx) {
			continue
		}
		value, ok := lookupUserContextValue(userCtx, rule.UserField)
		if !ok {
			return sqlQuery, fmt.Errorf("用户上下文字段不存在: %s", rule.UserField)
		}
		cond, err := renderBoundaryCondition(rule, value)
		if err != nil {
			return sqlQuery, err
		}
		if cond != "" {
			conditions = append(conditions, cond)
		}
	}
	if len(conditions) == 0 {
		return sqlQuery, nil
	}
	return injectSQLConditions(sqlQuery, conditions), nil
}

func boundaryRuleMatches(rule DataBoundaryRule, userCtx map[string]interface{}) bool {
	uid, _ := lookupUserContextValue(userCtx, "user_id")
	gid, _ := lookupUserContextValue(userCtx, "global_user_id")
	if len(rule.ExcludeUsers) > 0 && (containsStringValue(rule.ExcludeUsers, uid) || containsStringValue(rule.ExcludeUsers, gid)) {
		return false
	}
	if len(rule.ExcludeGroups) > 0 {
		if groups, ok := lookupUserContextValue(userCtx, "group_ids"); ok && containsAnyValue(rule.ExcludeGroups, groups) {
			return false
		}
		if groups, ok := lookupUserContextValue(userCtx, "group_codes"); ok && containsAnyValue(rule.ExcludeGroups, groups) {
			return false
		}
		if groups, ok := lookupUserContextValue(userCtx, "group_names"); ok && containsAnyValue(rule.ExcludeGroups, groups) {
			return false
		}
	}
	if len(rule.Users) > 0 && !containsStringValue(rule.Users, uid) && !containsStringValue(rule.Users, gid) {
		return false
	}
	if len(rule.Groups) > 0 {
		if groups, ok := lookupUserContextValue(userCtx, "group_ids"); ok && containsAnyValue(rule.Groups, groups) {
			return true
		}
		if groups, ok := lookupUserContextValue(userCtx, "group_codes"); ok && containsAnyValue(rule.Groups, groups) {
			return true
		}
		if groups, ok := lookupUserContextValue(userCtx, "group_names"); ok && containsAnyValue(rule.Groups, groups) {
			return true
		}
		return false
	}
	return true
}

func lookupUserContextValue(userCtx map[string]interface{}, key string) (interface{}, bool) {
	if v, ok := userCtx[key]; ok {
		return v, true
	}
	aliases := map[string]string{
		"company_user_id": "global_user_id",
		"current_user_id": "user_id",
		"login_user_id":   "user_id",
	}
	if alias, ok := aliases[key]; ok {
		v, found := userCtx[alias]
		return v, found
	}
	return nil, false
}

func renderBoundaryCondition(rule DataBoundaryRule, value interface{}) (string, error) {
	op := strings.ToUpper(strings.TrimSpace(rule.Operator))
	if op == "" {
		op = "="
	}
	field := fmt.Sprintf("%s.%s", safeSQLIdent(rule.Table), safeSQLIdent(rule.Field))
	values := valueToStringSlice(value)
	if len(values) == 0 {
		return "", nil
	}
	if op == "IN" || len(values) > 1 {
		return fmt.Sprintf("%s IN (%s)", field, quoteSQLList(values)), nil
	}
	if !isAllowedBoundaryOperator(op) {
		return "", fmt.Errorf("不支持的数据边界操作符: %s", op)
	}
	return fmt.Sprintf("%s %s %s", field, op, quoteSQLValue(values[0])), nil
}

func injectSQLConditions(sqlQuery string, conditions []string) string {
	sql := strings.TrimRight(strings.TrimSpace(sqlQuery), ";")
	condition := "(" + strings.Join(conditions, ") AND (") + ")"
	upper := strings.ToUpper(sql)
	insertAt := len(sql)
	for _, kw := range []string{" GROUP BY ", " ORDER BY ", " HAVING ", " LIMIT "} {
		if idx := strings.Index(upper, kw); idx >= 0 && idx < insertAt {
			insertAt = idx
		}
	}
	prefix := strings.TrimSpace(sql[:insertAt])
	suffix := sql[insertAt:]
	if strings.Contains(strings.ToUpper(prefix), " WHERE ") {
		return prefix + " AND " + condition + suffix
	}
	return prefix + " WHERE " + condition + suffix
}

func safeSQLIdent(s string) string {
	return strings.Trim(strings.TrimSpace(s), "`\" ")
}

func quoteSQLValue(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func quoteSQLList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			quoted = append(quoted, quoteSQLValue(v))
		}
	}
	return strings.Join(quoted, ",")
}

func valueToStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return []string{fmt.Sprint(t)}
	}
}

func containsStringValue(list []string, value interface{}) bool {
	v := fmt.Sprint(value)
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func containsAnyValue(list []string, values interface{}) bool {
	for _, v := range valueToStringSlice(values) {
		for _, item := range list {
			if item == v {
				return true
			}
		}
	}
	return false
}

func isAllowedBoundaryOperator(op string) bool {
	switch op {
	case "=", "!=", "<>", ">", "<", ">=", "<=", "IN":
		return true
	default:
		return false
	}
}

// GetDataSourceSchema gets the schema information from a data source
func (t *Text2SQL) GetDataSourceSchema(dataSourceID string) (string, error) {
	var ds models.DataSource
	if err := t.db.Where("id = ?", dataSourceID).First(&ds).Error; err != nil {
		return "", err
	}

	// If schema info is stored and valid, return it (avoid loop)
	if ds.SchemaInfo != "" && !strings.Contains(ds.SchemaInfo, "表结构分析中") {
		return ds.SchemaInfo, nil
	}

	// Otherwise, connect and get schema
	return t.fetchSchemaFromConnection(ds)
}

// fetchSchemaFromConnection connects to the database and fetches schema
func (t *Text2SQL) fetchSchemaFromConnection(ds models.DataSource) (string, error) {
	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	// 获取表结构
	tables, err := database.GetSchema(cfg)
	if err != nil {
		return fmt.Sprintf(`数据库: %s\n类型: %s\n错误: %v`, ds.Name, ds.SourceType, err), nil
	}

	// 生成 schema 描述
	return database.GenerateSchemaJSON(tables), nil
}

const text2SQLPromptJinja = `Given the following database schema:
{{ schema }}
{{ relations }}
Generate a SQL query to answer this question: {{ question }}

Requirements:
1. Use valid SQL syntax
2. ONLY return a single SELECT SQL statement - do NOT generate UPDATE, DELETE, INSERT, DROP, ALTER, CREATE, or any DDL/DML
3. Enclose the SQL in a markdown code block with sql identifier
4. No explanations before or after the SQL
5. If the question cannot be answered with the schema, return: -- Unable to answer
`

// buildPrompt builds the prompt for LLM to generate SQL
func (t *Text2SQL) buildPrompt(question, schema string) string {
	// 获取表关系（外键关联）。如果 schema 已经是预筛选结果，也会再次补充显式关系，帮助 LLM 生成 join。
	relations := ""
	if t.dataSourceID != "" {
		var ds models.DataSource
		if err := t.db.Where("id = ?", t.dataSourceID).First(&ds).Error; err == nil && ds.TableRelations != "" && ds.TableRelations != "[]" {
			// 解析 relations JSON 为可读文本
			var rels []map[string]string
			if json.Unmarshal([]byte(ds.TableRelations), &rels) == nil && len(rels) > 0 {
				relations = "\n表关系（外键关联）：\n"
				for _, r := range rels {
					relations += fmt.Sprintf("  %s.%s → %s.%s\n",
						r["from_table"], r["from_column"], r["to_table"], r["to_column"])
				}
			}
		}
	}

	fallback := fmt.Sprintf("Given the following database schema:\n%s\n%s\nGenerate a SQL query to answer this question: %s\nOnly return SQL.", schema, relations, question)
	return templating.SafeRender(text2SQLPromptJinja, map[string]interface{}{"schema": schema, "relations": relations, "question": question}, fallback)
}

// parseSQL extracts SQL from LLM response
// 使用多策略提取：先尝试 markdown 代码块，再搜索 SELECT/WITH 语句
func (t *Text2SQL) parseSQL(response string) string {
	if strings.TrimSpace(response) == "" {
		return ""
	}

	// 策略 1: 提取 markdown code block (```sql ... ``` 或 ``` ... ```)
	if sql := extractSQLFromCodeBlock(response); sql != "" {
		return sql
	}

	// 策略 2: 搜索 SELECT/WITH/SHOW/DESCRIBE/EXPLAIN 开头的语句
	if sql := extractFirstValidSQL(response); sql != "" {
		return sql
	}

	// 策略 3: 直接返回清理后的原始内容
	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```sql")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}

// generateQueryPlan uses the LLM to generate a structured QueryPlan and builds safe SQL.
// Supports AI-driven correction retry (max 5 rounds) when the LLM returns invalid JSON.
func (t *Text2SQL) generateQueryPlan(ctx context.Context, req *QueryRequest, result *QueryResult) (*QueryResult, error) {
	secure := req.SecureSemanticContext
	basePrompt := secure.RenderPrompt() + "\n\n用户问题: " + req.Question
	messages := []llm.Message{
		{Role: "system", Content: "You are a SQL expert. You MUST output valid JSON only, no markdown, no explanations."},
		{Role: "user", Content: basePrompt},
	}

	for attempt := 0; attempt < 5; attempt++ {
		resp, err := llm.ChatCompletionWithRetry(t.llmClient, ctx, llm.ChatRequest{
			Model:       t.llmClient.GetModel(),
			Messages:    messages,
			MaxTokens:   1000,
			Temperature: 0.1,
		}, 3)
		if err != nil {
			result.Error = fmt.Sprintf("LLM 调用失败: %v", err)
			return result, err
		}

		if strings.TrimSpace(resp.Content) == "" {
			if attempt < 4 {
				messages = append(messages, llm.Message{Role: "user", Content: "你的响应为空，请返回有效的 JSON 查询计划。"})
				continue
			}
			result.Error = "LLM 返回空响应"
			return result, nil
		}

		plan, err := ParseQueryPlan(resp.Content)
		if err == nil {
			sql, sqlErr := BuildSQL(*plan, *secure, req.Permissions...)
			if sqlErr == nil {
				result.SQL = sql
				return result, nil
			}
			err = fmt.Errorf("SQL 构建失败: %w", sqlErr)
		}

		// AI correction: feed error back to LLM
		if attempt < 4 {
			lastResp := resp.Content
			if len([]rune(lastResp)) > 300 {
				lastResp = string([]rune(lastResp)[:300]) + "..."
			}
			correction := fmt.Sprintf("你上次的 JSON 查询计划无效: %v。\n上次响应: %s\n请修正并只返回有效 JSON 查询计划。", err, lastResp)
			messages = append(messages, llm.Message{Role: "user", Content: correction})
			continue
		}
		result.Error = fmt.Sprintf("查询计划解析失败（已重试%d次）: %v", attempt+1, err)
		return result, nil
	}
	result.Error = "已达最大纠错轮数"
	return result, nil
}

// extractSQLFromCodeBlock 从 markdown 代码块中提取 SQL
func extractSQLFromCodeBlock(response string) string {
	// 匹配 ```sql ... ``` 或 ``` ... ```
	idx := strings.Index(response, "```sql")
	if idx < 0 {
		idx = strings.Index(response, "```")
	}
	if idx < 0 {
		return ""
	}
	rest := response[idx+3:] // 跳过 ```
	// 如果是以 ```sql 开头，跳过 sql 标记
	if strings.HasPrefix(rest, "sql") {
		rest = rest[3:]
	}
	end := strings.Index(rest, "```")
	if end < 0 {
		// 没有闭合的 ```，取剩余全部
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

// extractFirstValidSQL 在文本中搜索第一个 SELECT/WITH/SHOW/DESCRIBE/EXPLAIN 开头的语句
func extractFirstValidSQL(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}
		// 忽略注释行
		if strings.HasPrefix(stripped, "--") || strings.HasPrefix(stripped, "#") {
			continue
		}
		upper := strings.ToUpper(stripped)
		for _, prefix := range []string{"SELECT", "WITH", "SHOW", "DESCRIBE", "DESC", "EXPLAIN"} {
			if strings.HasPrefix(upper, prefix) {
				return stripped
			}
		}
	}
	return ""
}

// TestConnection tests the connection to a data source
func (t *Text2SQL) TestConnection(dataSourceID string) error {
	var ds models.DataSource
	if err := t.db.Where("id = ?", dataSourceID).First(&ds).Error; err != nil {
		return fmt.Errorf("数据源不存在: %w", err)
	}

	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	// 使用 database 包测试连接
	result, err := database.TestConnection(cfg)
	if err != nil {
		return err
	}

	if !result["success"].(bool) {
		return fmt.Errorf("%s", result["message"].(string))
	}

	return nil
}

// AnalyzeDataSource analyzes and returns schema information for a data source
func (t *Text2SQL) AnalyzeDataSource(dataSourceID string) (map[string]interface{}, error) {
	var ds models.DataSource
	if err := t.db.Where("id = ?", dataSourceID).First(&ds).Error; err != nil {
		return nil, err
	}

	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	// 获取表结构
	tables, err := database.GetSchema(cfg)
	if err != nil {
		return map[string]interface{}{
			"name":   ds.Name,
			"type":   ds.SourceType,
			"status": "error",
			"error":  err.Error(),
		}, nil
	}

	// 获取表关系
	relations, _ := database.GetTableRelations(cfg)

	// 生成 schema JSON
	schemaJSON := database.GenerateSchemaJSON(tables)

	// 更新数据源的 SchemaInfo
	t.db.Model(&ds).Update("SchemaInfo", schemaJSON)

	// 保存表关系到数据源
	relationsJSON := "[]"
	if len(relations) > 0 {
		b, err := json.Marshal(relations)
		if err == nil {
			relationsJSON = string(b)
		}
	}
	t.db.Model(&ds).Update("TableRelations", relationsJSON)

	result := map[string]interface{}{
		"name":            ds.Name,
		"type":            ds.SourceType,
		"status":          "success",
		"tables":          tables,
		"table_count":     len(tables),
		"schema_info":     schemaJSON,
		"table_relations": relations,
	}

	return result, nil
}

// OpenDB opens a connection to the data source and returns a sql.DB
func (t *Text2SQL) OpenDB(dataSource *models.DataSource) (*sql.DB, error) {
	cfg := database.ExternalDBConfig{
		Host:     dataSource.Host,
		Port:     dataSource.Port,
		User:     dataSource.User,
		Password: dataSource.Password,
		Database: dataSource.Database,
		Type:     dataSource.SourceType,
	}

	return database.Connect(cfg)
}

// GetTableInfo returns information about tables in the data source
func (t *Text2SQL) GetTableInfo(dataSourceID string) ([]map[string]string, error) {
	var ds models.DataSource
	if err := t.db.Where("id = ?", dataSourceID).First(&ds).Error; err != nil {
		return nil, err
	}

	cfg := database.ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}

	// 获取表结构
	tables, err := database.GetSchema(cfg)
	if err != nil {
		return nil, err
	}

	// 转换为 []map[string]string 格式
	var result []map[string]string
	for _, table := range tables {
		tableInfo := map[string]string{
			"name":    table.Name,
			"columns": fmt.Sprintf("%d", len(table.Columns)),
		}
		result = append(result, tableInfo)
	}

	return result, nil
}
