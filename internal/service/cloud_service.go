package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type CloudService struct {
	db *gorm.DB
}

func NewCloudService(db *gorm.DB) *CloudService {
	return &CloudService{db: db}
}

func (s *CloudService) ListProducts(publicOnly bool) ([]models.CloudProduct, error) {
	var rows []models.CloudProduct
	q := s.db.Order("created_at DESC")
	if publicOnly {
		q = q.Where("status = ?", "active")
	}
	return rows, q.Find(&rows).Error
}

func (s *CloudService) SaveProduct(row *models.CloudProduct) error {
	if row.Status == "" {
		row.Status = "active"
	}
	return s.db.Save(row).Error
}

func (s *CloudService) DeleteProduct(id string) error {
	return s.db.Delete(&models.CloudProduct{}, "id = ?", id).Error
}

func (s *CloudService) ListReleases(publicOnly bool) ([]models.CloudRelease, error) {
	var rows []models.CloudRelease
	q := s.db.Preload("Product").Order("published_at DESC, created_at DESC")
	if publicOnly {
		q = q.Where("status = ?", "published")
	}
	return rows, q.Find(&rows).Error
}

func (s *CloudService) GetRelease(idOrVersion string, publicOnly bool) (*models.CloudRelease, error) {
	var row models.CloudRelease
	q := s.db.Preload("Product").Where("id = ? OR version = ?", idOrVersion, idOrVersion)
	if publicOnly {
		q = q.Where("status = ?", "published")
	}
	if err := q.First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *CloudService) SaveRelease(row *models.CloudRelease) error {
	if row.Status == "" {
		row.Status = "draft"
	}
	if row.Status == "published" && row.PublishedAt == nil {
		now := time.Now()
		row.PublishedAt = &now
	}
	return s.db.Save(row).Error
}

func (s *CloudService) PublishRelease(id string, published bool) (*models.CloudRelease, error) {
	var row models.CloudRelease
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	if published {
		now := time.Now()
		row.Status = "published"
		row.PublishedAt = &now
	} else {
		row.Status = "draft"
		row.PublishedAt = nil
	}
	return &row, s.db.Save(&row).Error
}

func (s *CloudService) DeleteRelease(id string) error {
	return s.db.Delete(&models.CloudRelease{}, "id = ?", id).Error
}

func (s *CloudService) ListArtifacts(releaseID string) ([]models.CloudArtifact, error) {
	var rows []models.CloudArtifact
	q := s.db.Preload("Release").Order("created_at DESC")
	if releaseID != "" {
		q = q.Where("release_id = ?", releaseID)
	}
	return rows, q.Find(&rows).Error
}

func (s *CloudService) SaveArtifact(row *models.CloudArtifact) error {
	return s.db.Save(row).Error
}

func (s *CloudService) DeleteArtifact(id string) error {
	return s.db.Delete(&models.CloudArtifact{}, "id = ?", id).Error
}

func (s *CloudService) DownloadArtifact(ctx context.Context, id, ip, ua string) (*models.CloudArtifact, error) {
	var row models.CloudArtifact
	if err := s.db.First(&row, "id = ?", id).Error; err != nil {
		return nil, err
	}
	logRow := models.CloudDownloadLog{ArtifactID: row.ID, IP: ip, UserAgent: ua}
	if err := s.db.WithContext(ctx).Create(&logRow).Error; err != nil {
		return nil, fmt.Errorf("记录下载日志失败: %w", err)
	}
	return &row, nil
}

func (s *CloudService) DownloadStats() (map[string]interface{}, error) {
	var total int64
	if err := s.db.Model(&models.CloudDownloadLog{}).Count(&total).Error; err != nil {
		return nil, err
	}
	var artifacts int64
	if err := s.db.Model(&models.CloudArtifact{}).Count(&artifacts).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"downloads": total, "artifacts": artifacts}, nil
}

func (s *CloudService) ListSiteContents(publicOnly bool) ([]models.CloudSiteContent, error) {
	var rows []models.CloudSiteContent
	q := s.db.Order("updated_at DESC")
	if publicOnly {
		q = q.Where("status = ?", "published")
	}
	return rows, q.Find(&rows).Error
}

func (s *CloudService) GetSiteContent(key string, publicOnly bool) (*models.CloudSiteContent, error) {
	var row models.CloudSiteContent
	q := s.db.Where("`key` = ?", key)
	if publicOnly {
		q = q.Where("status = ?", "published")
	}
	if err := q.First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *CloudService) SaveSiteContent(row *models.CloudSiteContent) error {
	if row.Status == "" {
		row.Status = "draft"
	}
	return s.db.Save(row).Error
}

func (s *CloudService) DeleteSiteContent(id string) error {
	return s.db.Delete(&models.CloudSiteContent{}, "id = ?", id).Error
}
