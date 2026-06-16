package service

import (
	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/storage"
	"gorm.io/gorm"
)

type ConfigSuggestService struct {
	db    *gorm.DB
	cfg   *config.Config
	store storage.Driver
}

func NewConfigSuggestService(db *gorm.DB, cfg *config.Config, store storage.Driver) *ConfigSuggestService {
	return &ConfigSuggestService{db: db, cfg: cfg, store: store}
}
