package agent

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type TemplateWatcher struct {
	db       *gorm.DB
	filePath string
	lastMod  time.Time
	stopCh   chan struct{}
}

func NewTemplateWatcher(db *gorm.DB, filePath string) *TemplateWatcher {
	return &TemplateWatcher{db: db, filePath: filePath, stopCh: make(chan struct{})}
}

func (w *TemplateWatcher) Start() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.check()
			}
		}
	}()
}

func (w *TemplateWatcher) Stop() {
	close(w.stopCh)
}

func (w *TemplateWatcher) check() {
	info, err := os.Stat(w.filePath)
	if err != nil {
		return
	}
	if !info.ModTime().After(w.lastMod) {
		return
	}
	w.lastMod = info.ModTime()
	data, err := os.ReadFile(w.filePath)
	if err != nil {
		log.Printf("[TemplateWatcher] 读取文件失败: %v", err)
		return
	}
	var templates []struct {
		SkillID      string  `json:"skill_id"`
		DataSourceID string  `json:"data_source_id"`
		Type         string  `json:"type"`
		Key          string  `json:"key"`
		Content      string  `json:"content"`
		Confidence   float64 `json:"confidence"`
		Source       string  `json:"source"`
		Status       string  `json:"status"`
	}
	if err := json.Unmarshal(data, &templates); err != nil {
		log.Printf("[TemplateWatcher] JSON 解析失败: %v", err)
		return
	}
	imported := 0
	for _, t := range templates {
		if strings.TrimSpace(t.Content) == "" {
			continue
		}
		if strings.Contains(strings.ToLower(t.Content), "<environment_details>") {
			continue
		}
		t.Content = strings.TrimSpace(t.Content)
		var existing models.SkillRuntimeMemory
		if err := w.db.Where("skill_id = ? AND data_source_id = ? AND type = ? AND `key` = ?", t.SkillID, t.DataSourceID, t.Type, t.Key).First(&existing).Error; err == nil {
			if existing.Content == t.Content {
				continue
			}
			w.db.Model(&existing).Updates(map[string]interface{}{
				"content":      t.Content,
				"confidence":   t.Confidence,
				"source":       t.Source,
				"status":       t.Status,
				"updated_at":   time.Now(),
				"last_used_at": time.Now(),
			})
			imported++
			continue
		}
		w.db.Create(&models.SkillRuntimeMemory{
			SkillID:      t.SkillID,
			DataSourceID: t.DataSourceID,
			Type:         t.Type,
			Key:          t.Key,
			Content:      t.Content,
			Confidence:   t.Confidence,
			Source:       t.Source,
			Status:       t.Status,
			LastUsedAt:   time.Now(),
		})
		imported++
	}
	if imported > 0 {
		log.Printf("[TemplateWatcher] 自动导入 %d 条模板", imported)
	}
}
