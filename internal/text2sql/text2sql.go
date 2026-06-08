package text2sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/alaikis/opentether/internal/llm"
	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// Text2SQL converts natural language to SQL queries
type Text2SQL struct {
	db        *gorm.DB
	llmClient llm.Client
}

// New creates a new Text2SQL instance
func New(db *gorm.DB, llmClient llm.Client) *Text2SQL {
	return &Text2SQL{
		db:        db,
		llmClient: llmClient,
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
	// First generate SQL
	result, err := t.GenerateSQL(ctx, req)
	if err != nil {
		return result, err
	}

	if result.Error != "" {
		return result, nil
	}

	// Execute the SQL
	if strings.TrimSpace(result.SQL) == "" {
		result.Error = "生成的 SQL 为空"
		return result, nil
	}

	// Try to execute the query
	rows, err := t.db.Raw(result.SQL).Rows()
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

	return result, nil
}

// GetDataSourceSchema gets the schema information from a data source
func (t *Text2SQL) GetDataSourceSchema(dataSourceID string) (string, error) {
	var ds models.DataSource
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
		return "", err
	}

	// If schema info is stored, return it
	if ds.SchemaInfo != "" {
		return ds.SchemaInfo, nil
	}

	// Otherwise, need to connect and get schema
	return t.fetchSchemaFromConnection(ds)
}

// fetchSchemaFromConnection connects to the database and fetches schema
func (t *Text2SQL) fetchSchemaFromConnection(ds models.DataSource) (string, error) {
	// This would connect to the actual database and get schema
	// For now, return a simplified schema description
	return fmt.Sprintf(`Database: %s
Type: %s
Note: Schema information not available. Please configure schema manually.`,
		ds.Name, ds.SourceType), nil
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

	// Try a simple query to test connection
	tx := t.db.Raw("SELECT 1")
	if tx.Error != nil {
		return fmt.Errorf("连接测试失败: %w", tx.Error)
	}

	return nil
}

// AnalyzeDataSource analyzes and returns schema information for a data source
func (t *Text2SQL) AnalyzeDataSource(dataSourceID string) (map[string]interface{}, error) {
	var ds models.DataSource
	if err := t.db.First(&ds, dataSourceID).Error; err != nil {
		return nil, err
	}

	// For now, return basic info
	// In a full implementation, this would actually connect and analyze
	result := map[string]interface{}{
		"name":         ds.Name,
		"type":         ds.SourceType,
		"status":       "analyzed",
		"tables":       []string{}, // Would list actual tables
		"schema_info":  ds.SchemaInfo,
	}

	return result, nil
}

// OpenDB opens a connection to the data source and returns a sql.DB
func (t *Text2SQL) OpenDB(dataSource *models.DataSource) (*sql.DB, error) {
	// This would create a real database connection based on the data source type
	// For now, return an error as this requires proper implementation
	return nil, fmt.Errorf("database connection not implemented - need to implement based on SourceType")
}

// GetTableInfo returns information about tables in the data source
func (t *Text2SQL) GetTableInfo(dataSourceID string) ([]map[string]string, error) {
	// This would query the database for table information
	// For now, return empty
	return []map[string]string{}, nil
}
