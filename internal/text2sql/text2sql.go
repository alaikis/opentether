package text2sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/database"
	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// Text2SQL converts natural language to SQL queries
type Text2SQL struct {
	db           *gorm.DB
	llmClient    llm.Client
	externalDB   *sql.DB       // 外部数据库连接
	dataSourceID string        // 当前使用的数据源ID
}

// New creates a new Text2SQL instance
func New(db *gorm.DB, llmClient llm.Client) *Text2SQL {
	return &Text2SQL{
		db:        db,
		llmClient: llmClient,
	}
}

// NewWithDataSource creates a new Text2SQL instance with a data source
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

// connectToDataSource 连接到指定的数据源
func (t *Text2SQL) connectToDataSource(dataSourceID string) error {
	var ds models.DataSource
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
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
	t.dataSourceID = dataSourceID
	return nil
}

// Close 关闭外部数据库连接
func (t *Text2SQL) Close() {
	if t.externalDB != nil {
		t.externalDB.Close()
	}
}

// QueryRequest represents a text2sql query request
type QueryRequest struct {
	Question      string
	DataSourceID  string
	SchemaContext string // Optional schema context
}

// QueryResult represents the result of a text2sql query
type QueryResult struct {
	Question     string         `json:"question"`
	SQL          string         `json:"sql"`
	Columns      []string       `json:"columns"`
	Rows         [][]interface{} `json:"rows"`
	RowCount     int            `json:"row_count"`
	Error        string         `json:"error,omitempty"`
	ExecutionTime string        `json:"execution_time"`
}

// GenerateSQL generates a SQL query from natural language
func (t *Text2SQL) GenerateSQL(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	result := &QueryResult{
		Question: req.Question,
	}

	// Get datasource schema
	schema, err := t.GetDataSourceSchema(req.DataSourceID)
	if err != nil {
		result.Error = fmt.Sprintf("获取数据源失败: %v", err)
		return result, err
	}

	// Build prompt for LLM
	prompt := t.buildPrompt(req.Question, schema)

	// Call LLM to generate SQL
	resp, err := t.llmClient.ChatCompletion(ctx, llm.ChatRequest{
		Model: t.llmClient.GetModel(),
		Messages: []llm.Message{
			{Role: "system", Content: "You are a SQL expert. Generate SQL queries based on the schema provided. Only output the SQL query, no explanations."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1000,
		Temperature: 0.1,
	})

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
		return result, fmt.Errorf(result.Error)
	}

	// First generate SQL
	genResult, err := t.GenerateSQL(ctx, req)
	if err != nil {
		return genResult, err
	}

	if genResult.Error != "" {
		return genResult, nil
	}

	// Use the generated SQL
	result.SQL = genResult.SQL

	// Execute the SQL
	if strings.TrimSpace(result.SQL) == "" {
		result.Error = "生成的 SQL 为空"
		return result, nil
	}

	// 检查是否为危险SQL（DDL操作）
	upperSQL := strings.ToUpper(result.SQL)
	if strings.Contains(upperSQL, "DROP") || strings.Contains(upperSQL, "DELETE") || strings.Contains(upperSQL, "TRUNCATE") || strings.Contains(upperSQL, "ALTER") {
		result.Error = "不允许执行数据修改或表结构修改操作，仅支持 SELECT 查询"
		return result, nil
	}

	// Try to execute the query on external database
	rows, err := t.externalDB.QueryContext(ctx, result.SQL)
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

	// Scan results
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			result.Error = fmt.Sprintf("扫描行失败: %v", err)
			break
		}
		result.Rows = append(result.Rows, values)
	}

	result.RowCount = len(result.Rows)
	result.ExecutionTime = time.Since(startTime).String()

	return result, nil
}

// GetDataSourceSchema gets the schema information from a data source
func (t *Text2SQL) GetDataSourceSchema(dataSourceID string) (string, error) {
	var ds models.DataSource
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
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

// buildPrompt builds the prompt for LLM to generate SQL
func (t *Text2SQL) buildPrompt(question, schema string) string {
	return fmt.Sprintf(`Given the following database schema:
%s

Generate a SQL query to answer this question: %s

Requirements:
1. Use valid SQL syntax
2. Only return the SQL query, no explanations
3. If the question cannot be answered with the schema, return: -- Unable to answer
`, schema, question)
}

// parseSQL extracts SQL from LLM response
func (t *Text2SQL) parseSQL(response string) string {
	// Clean up the response - remove markdown code blocks if any
	response = strings.TrimSpace(response)
	response = strings.Trim(response, "```sql")
	response = strings.Trim(response, "```")
	response = strings.TrimSpace(response)

	return response
}

// TestConnection tests the connection to a data source
func (t *Text2SQL) TestConnection(dataSourceID string) error {
	var ds models.DataSource
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
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
		return fmt.Errorf(result["message"].(string))
	}

	return nil
}

// AnalyzeDataSource analyzes and returns schema information for a data source
func (t *Text2SQL) AnalyzeDataSource(dataSourceID string) (map[string]interface{}, error) {
	var ds models.DataSource
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
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
		relationsJSON = "[TODO: convert relations to JSON]"
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
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
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
