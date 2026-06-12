package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newPoolTestGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite gorm db: %v", err)
	}
	if err := db.AutoMigrate(&models.DataSource{}); err != nil {
		t.Fatalf("migrate datasource: %v", err)
	}
	return db
}

func insertPoolTestDataSource(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	ds := models.DataSource{
		ID:         id,
		Name:       id,
		SourceType: "mysql",
		Host:       "127.0.0.1",
		Port:       3306,
		User:       "user",
		Password:   "password",
		Database:   "app",
		Enabled:    true,
	}
	if err := db.Create(&ds).Error; err != nil {
		t.Fatalf("create datasource %s: %v", id, err)
	}
}

func newFakeSQLDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open fake sql db: %v", err)
	}
	return db
}

func TestExternalDBPoolManagerReusesPoolForSameDataSource(t *testing.T) {
	gormDB := newPoolTestGormDB(t)
	insertPoolTestDataSource(t, gormDB, "ds-1")

	openCount := 0
	manager := NewExternalDBPoolManager(gormDB, &ExternalDBPoolOptions{MaxPools: 2, IdleTTL: time.Hour})
	manager.open = func(ExternalDBConfig) (*sql.DB, error) {
		openCount++
		return newFakeSQLDB(t), nil
	}

	first, err := manager.Get(context.Background(), "ds-1")
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	second, err := manager.Get(context.Background(), "ds-1")
	if err != nil {
		t.Fatalf("second get: %v", err)
	}

	if first != second {
		t.Fatalf("expected same *sql.DB instance for repeated datasource get")
	}
	if openCount != 1 {
		t.Fatalf("expected one open, got %d", openCount)
	}
	if manager.PoolCount() != 1 {
		t.Fatalf("expected one cached pool, got %d", manager.PoolCount())
	}
}

func TestExternalDBPoolManagerInvalidatesWhenDataSourceChanges(t *testing.T) {
	gormDB := newPoolTestGormDB(t)
	insertPoolTestDataSource(t, gormDB, "ds-1")

	openCount := 0
	manager := NewExternalDBPoolManager(gormDB, &ExternalDBPoolOptions{MaxPools: 2, IdleTTL: time.Hour})
	manager.open = func(ExternalDBConfig) (*sql.DB, error) {
		openCount++
		return newFakeSQLDB(t), nil
	}

	first, err := manager.Get(context.Background(), "ds-1")
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	if err := gormDB.Model(&models.DataSource{}).Where("id = ?", "ds-1").Update("password", "new-password").Error; err != nil {
		t.Fatalf("update datasource: %v", err)
	}
	second, err := manager.Get(context.Background(), "ds-1")
	if err != nil {
		t.Fatalf("second get after update: %v", err)
	}

	if first == second {
		t.Fatalf("expected new *sql.DB instance after datasource config change")
	}
	if openCount != 2 {
		t.Fatalf("expected two opens after config change, got %d", openCount)
	}
	if manager.PoolCount() != 1 {
		t.Fatalf("expected stale pool replaced, got %d cached pools", manager.PoolCount())
	}
}

func TestExternalDBPoolManagerEvictsIdleAndOldestPools(t *testing.T) {
	gormDB := newPoolTestGormDB(t)
	insertPoolTestDataSource(t, gormDB, "ds-1")
	insertPoolTestDataSource(t, gormDB, "ds-2")
	insertPoolTestDataSource(t, gormDB, "ds-3")

	now := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	manager := NewExternalDBPoolManager(gormDB, &ExternalDBPoolOptions{MaxPools: 2, IdleTTL: time.Minute})
	manager.now = func() time.Time { return now }
	manager.open = func(ExternalDBConfig) (*sql.DB, error) {
		return newFakeSQLDB(t), nil
	}

	if _, err := manager.Get(context.Background(), "ds-1"); err != nil {
		t.Fatalf("get ds-1: %v", err)
	}
	now = now.Add(10 * time.Second)
	if _, err := manager.Get(context.Background(), "ds-2"); err != nil {
		t.Fatalf("get ds-2: %v", err)
	}
	now = now.Add(10 * time.Second)
	if _, err := manager.Get(context.Background(), "ds-3"); err != nil {
		t.Fatalf("get ds-3: %v", err)
	}
	if manager.PoolCount() != 2 {
		t.Fatalf("expected capacity capped at 2 pools, got %d", manager.PoolCount())
	}

	now = now.Add(2 * time.Minute)
	if manager.PoolCount() != 0 {
		t.Fatalf("expected idle TTL to evict all pools, got %d", manager.PoolCount())
	}
}
