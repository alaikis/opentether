package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

const (
	defaultExternalMaxPools     = 20
	defaultExternalMaxOpenConns = 10
	defaultExternalMaxIdleConns = 5
	defaultExternalConnMaxLife  = 30 * time.Minute
	defaultExternalConnMaxIdle  = 5 * time.Minute
	defaultExternalPoolIdleTTL  = 30 * time.Minute
)

// ExternalDBPoolOptions controls the lifecycle and size of cached external DB pools.
type ExternalDBPoolOptions struct {
	MaxPools        int
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	IdleTTL         time.Duration
}

// DefaultExternalDBPoolOptions returns conservative defaults to avoid unbounded pool growth.
func DefaultExternalDBPoolOptions() ExternalDBPoolOptions {
	return ExternalDBPoolOptions{
		MaxPools:        defaultExternalMaxPools,
		MaxOpenConns:    defaultExternalMaxOpenConns,
		MaxIdleConns:    defaultExternalMaxIdleConns,
		ConnMaxLifetime: defaultExternalConnMaxLife,
		ConnMaxIdleTime: defaultExternalConnMaxIdle,
		IdleTTL:         defaultExternalPoolIdleTTL,
	}
}

type externalDBOpener func(ExternalDBConfig) (*sql.DB, error)

type externalDBPoolEntry struct {
	db          *sql.DB
	fingerprint string
	lastUsed    time.Time
}

// ExternalDBPoolManager caches external datasource sql.DB handles by datasource ID.
// sql.DB is itself a connection pool; this manager prevents recreating that pool per request.
type ExternalDBPoolManager struct {
	mu      sync.Mutex
	gormDB  *gorm.DB
	options ExternalDBPoolOptions
	open    externalDBOpener
	pools   map[string]*externalDBPoolEntry
	now     func() time.Time
}

// NewExternalDBPoolManager creates a bounded manager for external datasource pools.
func NewExternalDBPoolManager(gormDB *gorm.DB, opts *ExternalDBPoolOptions) *ExternalDBPoolManager {
	options := DefaultExternalDBPoolOptions()
	if opts != nil {
		options = normalizeExternalDBPoolOptions(*opts)
	}

	return &ExternalDBPoolManager{
		gormDB:  gormDB,
		options: options,
		open: func(cfg ExternalDBConfig) (*sql.DB, error) {
			return ConnectWithPoolOptions(cfg, options)
		},
		pools: make(map[string]*externalDBPoolEntry),
		now:   time.Now,
	}
}

func normalizeExternalDBPoolOptions(opts ExternalDBPoolOptions) ExternalDBPoolOptions {
	defaults := DefaultExternalDBPoolOptions()
	if opts.MaxPools <= 0 {
		opts.MaxPools = defaults.MaxPools
	}
	if opts.MaxOpenConns <= 0 {
		opts.MaxOpenConns = defaults.MaxOpenConns
	}
	if opts.MaxIdleConns < 0 {
		opts.MaxIdleConns = defaults.MaxIdleConns
	}
	if opts.MaxIdleConns > opts.MaxOpenConns {
		opts.MaxIdleConns = opts.MaxOpenConns
	}
	if opts.ConnMaxLifetime <= 0 {
		opts.ConnMaxLifetime = defaults.ConnMaxLifetime
	}
	if opts.ConnMaxIdleTime <= 0 {
		opts.ConnMaxIdleTime = defaults.ConnMaxIdleTime
	}
	if opts.IdleTTL <= 0 {
		opts.IdleTTL = defaults.IdleTTL
	}
	return opts
}

// Get returns a reusable sql.DB pool for the datasource.
func (m *ExternalDBPoolManager) Get(ctx context.Context, dataSourceID string) (*sql.DB, error) {
	if m == nil {
		return nil, fmt.Errorf("外部数据源连接池未初始化")
	}
	if dataSourceID == "" {
		return nil, fmt.Errorf("数据源 ID 不能为空")
	}
	if m.gormDB == nil {
		return nil, fmt.Errorf("主数据库未初始化，无法加载数据源配置")
	}

	var ds models.DataSource
	if err := m.gormDB.WithContext(ctx).Where("id = ?", dataSourceID).First(&ds).Error; err != nil {
		return nil, fmt.Errorf("数据源不存在: %w", err)
	}
	if !ds.Enabled {
		return nil, fmt.Errorf("数据源已禁用: %s", ds.Name)
	}

	cfg := ExternalDBConfig{
		Host:     ds.Host,
		Port:     ds.Port,
		User:     ds.User,
		Password: ds.Password,
		Database: ds.Database,
		Type:     ds.SourceType,
	}
	fingerprint := dataSourceFingerprint(ds)
	now := m.now()

	m.mu.Lock()
	m.closeExpiredLocked(now)
	if entry, ok := m.pools[dataSourceID]; ok && entry.fingerprint == fingerprint {
		entry.lastUsed = now
		db := entry.db
		m.mu.Unlock()
		return db, nil
	}
	if entry, ok := m.pools[dataSourceID]; ok {
		_ = entry.db.Close()
		delete(m.pools, dataSourceID)
	}
	m.mu.Unlock()

	newDB, err := m.open(cfg)
	if err != nil {
		return nil, fmt.Errorf("数据源 %s 暂时不可用，请稍后重试或联系管理员检查数据源配置", ds.Name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Another goroutine may have created the same valid pool while we were connecting.
	if entry, ok := m.pools[dataSourceID]; ok && entry.fingerprint == fingerprint {
		entry.lastUsed = now
		_ = newDB.Close()
		return entry.db, nil
	}
	if entry, ok := m.pools[dataSourceID]; ok {
		_ = entry.db.Close()
		delete(m.pools, dataSourceID)
	}

	m.evictUntilCapacityLocked(now)
	m.pools[dataSourceID] = &externalDBPoolEntry{
		db:          newDB,
		fingerprint: fingerprint,
		lastUsed:    now,
	}
	return newDB, nil
}

// Invalidate closes and removes a datasource pool, useful after manual datasource updates.
func (m *ExternalDBPoolManager) Invalidate(dataSourceID string) {
	if m == nil || dataSourceID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry, ok := m.pools[dataSourceID]; ok {
		_ = entry.db.Close()
		delete(m.pools, dataSourceID)
	}
}

// CloseAll closes all cached pools.
func (m *ExternalDBPoolManager) CloseAll() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var firstErr error
	for id, entry := range m.pools {
		if err := entry.db.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		delete(m.pools, id)
	}
	return firstErr
}

// PoolCount returns the number of cached pools. It is primarily useful for diagnostics and tests.
func (m *ExternalDBPoolManager) PoolCount() int {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeExpiredLocked(m.now())
	return len(m.pools)
}

func (m *ExternalDBPoolManager) closeExpiredLocked(now time.Time) {
	if m.options.IdleTTL <= 0 {
		return
	}
	for id, entry := range m.pools {
		if now.Sub(entry.lastUsed) > m.options.IdleTTL {
			_ = entry.db.Close()
			delete(m.pools, id)
		}
	}
}

func (m *ExternalDBPoolManager) evictUntilCapacityLocked(now time.Time) {
	m.closeExpiredLocked(now)
	for len(m.pools) >= m.options.MaxPools {
		oldestID := ""
		var oldest time.Time
		for id, entry := range m.pools {
			if oldestID == "" || entry.lastUsed.Before(oldest) {
				oldestID = id
				oldest = entry.lastUsed
			}
		}
		if oldestID == "" {
			return
		}
		_ = m.pools[oldestID].db.Close()
		delete(m.pools, oldestID)
	}
}

func dataSourceFingerprint(ds models.DataSource) string {
	payload := fmt.Sprintf("%s\x00%s\x00%d\x00%s\x00%s\x00%s\x00%s\x00%t",
		ds.SourceType,
		ds.Host,
		ds.Port,
		ds.User,
		ds.Password,
		ds.Database,
		ds.Connection,
		ds.Enabled,
	)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
