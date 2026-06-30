package main

import (
	"fmt"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: migrate_sqlite_to_mysql <mysql_dsn>")
		fmt.Println("示例: migrate_sqlite_to_mysql root:password@tcp(127.0.0.1:3306)/opentether?charset=utf8mb4&parseTime=True")
		os.Exit(1)
	}
	srcDB, err := gorm.Open(sqlite.Open("data/opentether.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("SQLite: ", err)
	}
	dstDB, err := gorm.Open(mysql.Open(os.Args[1]), &gorm.Config{})
	if err != nil {
		log.Fatal("MySQL: ", err)
	}
	tables := []string{"providers", "skills", "data_sources", "mcp_configs", "conversations", "messages", "users", "groups", "api_keys", "skill_runtime_memories", "skill_config_versions", "eval_cases", "eval_runs", "cloud_products", "cloud_releases", "cloud_artifacts", "cloud_download_logs", "cloud_site_contents", "access_policies", "precompute_jobs", "agent_task_graphs", "agent_task_nodes", "agent_task_outputs", "webhook_configs", "webhook_delivery_logs", "rag_documents", "rag_chunks"}
	for _, table := range tables {
		var rows []map[string]interface{}
		if err := srcDB.Table(table).Find(&rows).Error; err != nil {
			continue
		}
		for _, row := range rows {
			delete(row, "id")
			if err := dstDB.Table(table).Create(row).Error; err != nil {
				fmt.Printf("WARN: %s: %v\n", table, err)
			}
		}
		fmt.Printf("OK: %s (%d rows)\n", table, len(rows))
	}
	fmt.Println("迁移完成")
}
