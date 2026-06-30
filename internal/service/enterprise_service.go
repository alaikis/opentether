package service

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type EnterpriseService struct { db *gorm.DB }
func NewEnterpriseService(db *gorm.DB) *EnterpriseService { return &EnterpriseService{db: db} }

func (s *EnterpriseService) GetSettings() ([]models.SystemSetting, error) { var rows []models.SystemSetting; return rows, s.db.Order("key ASC").Find(&rows).Error }
func (s *EnterpriseService) SaveSetting(row *models.SystemSetting) error { return s.db.Save(row).Error }

func (s *EnterpriseService) RequestSkillPublish(skillID, reason string) (*models.SkillPublishRequest, error) {
	req := &models.SkillPublishRequest{SkillID: skillID, Reason: reason, Status: "pending"}
	return req, s.db.Create(req).Error
}
func (s *EnterpriseService) ReviewSkillPublish(id, reviewer, status, comment string) (*models.SkillPublishRequest, error) {
	var req models.SkillPublishRequest
	if err := s.db.First(&req, "id = ?", id).Error; err != nil { return nil, err }
	req.Status = status; req.Reviewer = reviewer; req.Comment = comment
	if status == "approved" { s.db.Model(&models.Skill{}).Where("id = ?", req.SkillID).Update("enabled", true) }
	return &req, s.db.Save(&req).Error
}
func (s *EnterpriseService) ListSkillPublishRequests() ([]models.SkillPublishRequest, error) { var rows []models.SkillPublishRequest; return rows, s.db.Order("created_at DESC").Find(&rows).Error }

func (s *EnterpriseService) BackupData() (*models.BackupRecord, error) {
	os.MkdirAll("backups", 0755)
	path := filepath.Join("backups", "backup_"+time.Now().Format("20060102_150405")+".zip")
	rec := &models.BackupRecord{Path: path, Status: "running"}
	s.db.Create(rec)
	err := zipPaths(path, []string{"config.yaml", "data"})
	if err != nil { rec.Status="failed"; rec.Error=err.Error() } else { rec.Status="completed" }
	return rec, s.db.Save(rec).Error
}
func (s *EnterpriseService) ListBackups() ([]models.BackupRecord, error) { var rows []models.BackupRecord; return rows, s.db.Order("created_at DESC").Find(&rows).Error }

func zipPaths(dst string, roots []string) error {
	f, err := os.Create(dst); if err != nil { return err }; defer f.Close()
	zw := zip.NewWriter(f); defer zw.Close()
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil { continue }
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() { return nil }
			w, err := zw.Create(path); if err != nil { return err }
			r, err := os.Open(path); if err != nil { return err }; defer r.Close()
			_, _ = io.Copy(w, r)
			return nil
		})
	}
	return nil
}
