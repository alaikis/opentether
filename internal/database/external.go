package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// ExternalDBConfig 外部数据库配置
type ExternalDBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Type     string // mysql, postgres
}

// Connect 连接到外部数据库
func Connect(cfg ExternalDBConfig) (*sql.DB, error) {
	return ConnectWithPoolOptions(cfg, DefaultExternalDBPoolOptions())
}

// ConnectWithPoolOptions 连接到外部数据库并设置连接池限制。
func ConnectWithPoolOptions(cfg ExternalDBConfig, opts ExternalDBPoolOptions) (*sql.DB, error) {
	var dsn string

	switch cfg.Type {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
	case "postgres":
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
			cfg.Host, cfg.User, cfg.Password, cfg.Database, cfg.Port)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}

	db, err := sql.Open(cfg.Type, dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}

	opts = normalizeExternalDBPoolOptions(opts)
	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	db.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	// 测试连接
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("数据库连接测试失败 (type=%s)", cfg.Type)
	}

	return db, nil
}

// TestConnection 测试数据库连接
func TestConnection(cfg ExternalDBConfig) (map[string]interface{}, error) {
	db, err := Connect(cfg)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"message": err.Error(),
			"error":   "connection_failed",
		}, nil
	}
	defer db.Close()

	return map[string]interface{}{
		"success": true,
		"message": "数据库连接成功",
		"version": getDBVersion(db, cfg.Type),
	}, nil
}

// getDBVersion 获取数据库版本
func getDBVersion(db *sql.DB, dbType string) string {
	var version string
	var query string

	switch dbType {
	case "mysql":
		query = "SELECT VERSION()"
	case "postgres":
		query = "SELECT version()"
	default:
		return "unknown"
	}

	err := db.QueryRow(query).Scan(&version)
	if err != nil {
		return "unknown"
	}
	return version
}

// TableInfo 表结构信息
type TableInfo struct {
	Name    string       `json:"name"`
	Columns []ColumnInfo `json:"columns"`
}

// ColumnInfo 列信息
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	KeyType  string `json:"key_type"` // PRI, MUL, 空
	Extra    string `json:"extra"`
	Comment  string `json:"comment"`
}

// GetSchema 获取数据库表结构
func GetSchema(cfg ExternalDBConfig) ([]TableInfo, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var tables []TableInfo

	switch cfg.Type {
	case "mysql":
		tables, err = getMySQLSchema(db)
	case "postgres":
		tables, err = getPostgresSchema(db)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}

	return tables, err
}

// getMySQLSchema 获取 MySQL 表结构
func getMySQLSchema(db *sql.DB) ([]TableInfo, error) {
	// 获取所有表
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	var tableName string

	for rows.Next() {
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		// 获取表的列信息
		columns, err := getMySQLColumns(db, tableName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, TableInfo{
			Name:    tableName,
			Columns: columns,
		})
	}

	return tables, nil
}

// getMySQLColumns 获取 MySQL 表的列信息
func getMySQLColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT
			COLUMN_NAME,
			COLUMN_TYPE,
			IS_NULLABLE,
			COLUMN_KEY,
			EXTRA,
			COLUMN_COMMENT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		if err := rows.Scan(&col.Name, &col.Type, &isNullable, &col.KeyType, &col.Extra, &col.Comment); err != nil {
			return nil, err
		}
		col.Nullable = (isNullable == "YES")
		columns = append(columns, col)
	}

	return columns, nil
}

// getPostgresSchema 获取 PostgreSQL 表结构
func getPostgresSchema(db *sql.DB) ([]TableInfo, error) {
	// 获取所有表
	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	var tableName string

	for rows.Next() {
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		// 获取表的列信息
		columns, err := getPostgresColumns(db, tableName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, TableInfo{
			Name:    tableName,
			Columns: columns,
		})
	}

	return tables, nil
}

// getPostgresColumns 获取 PostgreSQL 表的列信息
func getPostgresColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			CASE WHEN pk.column_name IS NOT NULL THEN 'PRI' ELSE '' END,
			c.column_default,
			c.column_comment
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
			WHERE tc.table_name = $1 AND tc.constraint_type = 'PRIMARY KEY'
		) pk ON c.column_name = pk.column_name
		WHERE c.table_name = $1 AND c.table_schema = 'public'
		ORDER BY c.ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		if err := rows.Scan(&col.Name, &col.Type, &isNullable, &col.KeyType, &col.Extra, &col.Comment); err != nil {
			return nil, err
		}
		col.Nullable = (isNullable == "YES")
		columns = append(columns, col)
	}

	return columns, nil
}

// GenerateSchemaJSON 生成 Schema JSON 字符串
func GenerateSchemaJSON(tables []TableInfo) string {
	result := "数据库表结构：\n\n"

	for _, table := range tables {
		result += fmt.Sprintf("表: %s\n", table.Name)
		for _, col := range table.Columns {
			nullable := ""
			if col.Nullable {
				nullable = " (可空)"
			}
			primaryKey := ""
			if col.KeyType == "PRI" {
				primaryKey = " [主键]"
			}
			result += fmt.Sprintf("  - %s: %s%s%s\n", col.Name, col.Type, nullable, primaryKey)
		}
		result += "\n"
	}

	return result
}

// GetTableRelations 获取外键关系（从数据库）
func GetTableRelations(cfg ExternalDBConfig) ([]map[string]string, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var relations []map[string]string

	switch cfg.Type {
	case "mysql":
		relations, err = getMySQLRelations(db)
	case "postgres":
		relations, err = getPostgresRelations(db)
	}

	return relations, err
}

// getMySQLRelations 获取 MySQL 外键关系
func getMySQLRelations(db *sql.DB) ([]map[string]string, error) {
	query := `
		SELECT
			kcu.table_name AS from_table,
			kcu.column_name AS from_column,
			kcu.referenced_table_name AS to_table,
			kcu.referenced_column_name AS to_column
		FROM information_schema.key_column_usage kcu
		WHERE kcu.referenced_table_name IS NOT NULL
		AND kcu.table_schema = DATABASE()
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []map[string]string
	for rows.Next() {
		var fromTable, fromColumn, toTable, toColumn string
		if err := rows.Scan(&fromTable, &fromColumn, &toTable, &toColumn); err != nil {
			return nil, err
		}
		relations = append(relations, map[string]string{
			"from_table":  fromTable,
			"from_column": fromColumn,
			"to_table":    toTable,
			"to_column":   toColumn,
		})
	}

	return relations, nil
}

// getPostgresRelations 获取 PostgreSQL 外键关系
func getPostgresRelations(db *sql.DB) ([]map[string]string, error) {
	query := `
		SELECT
			tc.table_name AS from_table,
			kcu.column_name AS from_column,
			ccu.table_name AS to_table,
			ccu.column_name AS to_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_schema = 'public'
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relations []map[string]string
	for rows.Next() {
		var fromTable, fromColumn, toTable, toColumn string
		if err := rows.Scan(&fromTable, &fromColumn, &toTable, &toColumn); err != nil {
			return nil, err
		}
		relations = append(relations, map[string]string{
			"from_table":  fromTable,
			"from_column": fromColumn,
			"to_table":    toTable,
			"to_column":   toColumn,
		})
	}

	return relations, nil
}
