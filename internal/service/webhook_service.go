package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type WebhookService struct {
	db *gorm.DB
}

func NewWebhookService(db *gorm.DB) *WebhookService { return &WebhookService{db: db} }

func (s *WebhookService) ListConfigs() ([]models.WebhookConfig, error) {
	var rows []models.WebhookConfig
	return rows, s.db.Order("created_at DESC").Find(&rows).Error
}

func (s *WebhookService) SaveConfig(row *models.WebhookConfig) error {
	return s.db.Save(row).Error
}

func (s *WebhookService) DeleteConfig(id string) error {
	return s.db.Delete(&models.WebhookConfig{}, "id = ?", id).Error
}

func (s *WebhookService) Deliver(event string, payload map[string]interface{}) {
	var configs []models.WebhookConfig
	if err := s.db.Where("enabled = ?", true).Find(&configs).Error; err != nil || len(configs) == 0 {
		return
	}
	b, _ := json.Marshal(payload)
	for _, cfg := range configs {
		if strings.TrimSpace(cfg.URL) == "" {
			continue
		}
		if !s.shouldDeliver(cfg, event) {
			continue
		}
		log := models.WebhookDeliveryLog{ConfigID: cfg.ID, Event: event, PayloadJSON: string(b), Status: "pending"}
		s.db.Create(&log)
		statusCode, respBody, err := s.sendHTTP(cfg, string(b))
		log.Status = "delivered"
		log.StatusCode = statusCode
		log.Response = truncateWebhook(respBody, 2000)
		if err != nil {
			log.Status = "failed"
			log.Error = err.Error()
		}
		if statusCode >= 400 {
			log.Status = "failed"
			log.Error = fmt.Sprintf("HTTP %d: %s", statusCode, truncateWebhook(respBody, 200))
		}
		s.db.Model(&log).Updates(map[string]interface{}{"status": log.Status, "status_code": log.StatusCode, "response": log.Response, "error": log.Error})
	}
}

func (s *WebhookService) shouldDeliver(cfg models.WebhookConfig, event string) bool {
	if strings.TrimSpace(cfg.Events) == "" || cfg.Events == "*" {
		return true
	}
	for _, e := range strings.Split(cfg.Events, ",") {
		if strings.TrimSpace(e) == event {
			return true
		}
	}
	return false
}

func (s *WebhookService) sendHTTP(cfg models.WebhookConfig, body string) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader([]byte(body)))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "OpenTether-Webhook/1.0")
	if cfg.Secret != "" {
		ts := time.Now().Unix()
		sig := hmacSHA256(cfg.Secret, fmt.Sprintf("%d.%s", ts, body))
		req.Header.Set("X-Webhook-Signature", sig)
		req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", ts))
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String(), nil
}

func hmacSHA256(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func truncateWebhook(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
