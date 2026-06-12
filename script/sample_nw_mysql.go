package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/alaikis/opentether/internal/models"
	"github.com/glebarez/sqlite"
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func main() {
	local, err := gorm.Open(sqlite.Open("data/opentether.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	var ds models.DataSource
	if err := local.Where("name = ?", "nw-mysql").First(&ds).Error; err != nil {
		log.Fatal(err)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", ds.User, ds.Password, ds.Host, ds.Port, ds.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tables := []string{"t_profile", "t_depart", "t_role", "t_order", "t_order_item", "t_order_amount", "t_order_sales_profit", "t_inventory", "t_inventory_cost_summary", "t_inventory_cost_log", "t_item", "t_variants", "t_purchase", "t_purchase_item", "t_member", "t_warehouse", "t_report_object_group"}
	for _, table := range tables {
		fmt.Println("\n==", table, "==")
		q := fmt.Sprintf("SELECT * FROM `%s` LIMIT 3", table)
		rows, err := db.Query(q)
		if err != nil {
			fmt.Println("ERR", err)
			continue
		}
		cols, _ := rows.Columns()
		fmt.Println(strings.Join(cols, " | "))
		for rows.Next() {
			vals := make([]interface{}, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				fmt.Println("scan", err)
				break
			}
			parts := make([]string, len(cols))
			for i, v := range vals {
				if b, ok := v.([]byte); ok {
					parts[i] = string(b)
				} else {
					parts[i] = fmt.Sprint(v)
				}
				if len(parts[i]) > 80 {
					parts[i] = parts[i][:80] + "..."
				}
			}
			fmt.Println(strings.Join(parts, " | "))
		}
		rows.Close()
	}
}
