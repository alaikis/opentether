package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/alaikis/opentether/internal/models"
	"github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type TableSummary struct {
	Name       string
	Comment    string
	Rows       int64
	Columns    []ColumnSummary
	SampleCols []string
}

type ColumnSummary struct {
	Name    string
	Type    string
	Key     string
	Comment string
}

func main() {
	local, err := gorm.Open(sqlite.Open("data/opentether.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	var ds models.DataSource
	if err := local.Where("name = ?", "nw-mysql").First(&ds).Error; err != nil {
		log.Fatal(err)
	}
	if ds.SourceType != "mysql" {
		log.Fatalf("unsupported source type: %s", ds.SourceType)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", ds.User, ds.Password, ds.Host, ds.Port, ds.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query(`
SELECT table_name, table_comment, COALESCE(table_rows, 0)
FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
ORDER BY table_name`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var tables []TableSummary
	for rows.Next() {
		var t TableSummary
		if err := rows.Scan(&t.Name, &t.Comment, &t.Rows); err != nil {
			log.Fatal(err)
		}
		tables = append(tables, t)
	}

	for i := range tables {
		cols, err := db.Query(`
SELECT column_name, column_type, column_key, column_comment
FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = ?
ORDER BY ordinal_position`, tables[i].Name)
		if err != nil {
			log.Fatal(err)
		}
		for cols.Next() {
			var c ColumnSummary
			if err := cols.Scan(&c.Name, &c.Type, &c.Key, &c.Comment); err != nil {
				log.Fatal(err)
			}
			tables[i].Columns = append(tables[i].Columns, c)
		}
		cols.Close()
	}

	fmt.Printf("DATABASE %s (%s) tables=%d\n\n", ds.Database, ds.Name, len(tables))
	for _, t := range tables {
		fmt.Printf("TABLE %s rows~%d comment=%s\n", t.Name, t.Rows, t.Comment)
		for _, c := range t.Columns {
			marker := ""
			lower := strings.ToLower(c.Name + " " + c.Comment)
			if strings.Contains(lower, "emp") || strings.Contains(lower, "staff") || strings.Contains(lower, "user") || strings.Contains(lower, "sale") || strings.Contains(lower, "order") || strings.Contains(lower, "cost") || strings.Contains(lower, "stock") || strings.Contains(lower, "inventory") || strings.Contains(lower, "price") || strings.Contains(lower, "amount") || strings.Contains(lower, "qty") || strings.Contains(lower, "quantity") || strings.Contains(lower, "部门") || strings.Contains(lower, "员工") || strings.Contains(lower, "销售") || strings.Contains(lower, "成本") || strings.Contains(lower, "库存") || strings.Contains(lower, "订单") || strings.Contains(lower, "金额") || strings.Contains(lower, "数量") {
				marker = " *"
			}
			fmt.Printf("  - %s %s key=%s comment=%s%s\n", c.Name, c.Type, c.Key, c.Comment, marker)
		}
		fmt.Println()
	}

	keywords := []string{"sale", "sales", "order", "cost", "stock", "inventory", "employee", "staff", "user", "product", "sku", "warehouse"}
	fmt.Println("LIKELY BUSINESS TABLES:")
	var likely []string
	for _, t := range tables {
		all := strings.ToLower(t.Name + " " + t.Comment)
		for _, c := range t.Columns {
			all += " " + strings.ToLower(c.Name+" "+c.Comment)
		}
		for _, k := range keywords {
			if strings.Contains(all, k) {
				likely = append(likely, t.Name)
				break
			}
		}
	}
	sort.Strings(likely)
	for _, name := range likely {
		fmt.Println(" -", name)
	}
}
