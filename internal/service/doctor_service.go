package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/storage"
	"gorm.io/gorm"
)

type DoctorReport struct {
	DB       string `json:"db"`
	Storage  string `json:"storage"`
	Provider string `json:"provider"`
	Skills   string `json:"skills"`
}

func DoctorDiagnose(db *gorm.DB, cfg *config.Config, store storage.Driver) *DoctorReport {
	r := &DoctorReport{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sqlDB, err := db.DB()
	if err != nil {
		r.DB = fmt.Sprintf("获取DB失败: %v", err)
	} else if err := sqlDB.PingContext(ctx); err != nil {
		r.DB = fmt.Sprintf("DB连接失败: %v", err)
	} else {
		r.DB = "OK"
	}
	if store == nil {
		r.Storage = "未初始化"
	} else {
		path := fmt.Sprintf("health/doctor-%d.txt", time.Now().Unix())
		_, err := store.Save(ctx, path, []byte("doctor check"), "text/plain")
		if err != nil {
			r.Storage = fmt.Sprintf("写入失败: %v", err)
		} else {
			r.Storage = "OK"
		}
	}
	var count int64
	db.Model(&struct{ ID string }{}).Table("providers").Where("enabled = ?", true).Count(&count)
	if count > 0 {
		r.Provider = "OK"
	} else {
		r.Provider = "无可用Provider"
	}
	db.Model(&struct{ ID string }{}).Table("skills").Where("enabled = ?", true).Count(&count)
	if count > 0 {
		r.Skills = "OK"
	} else {
		r.Skills = "无可用Skill"
	}
	return r
}
