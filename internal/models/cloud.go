package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CloudProduct struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`
	Slug        string    `json:"slug" gorm:"type:varchar(120);uniqueIndex;not null"`
	Description string    `json:"description" gorm:"type:text"`
	Status      string    `json:"status" gorm:"type:varchar(20);default:active;index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (m *CloudProduct) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type CloudRelease struct {
	ID          string        `json:"id" gorm:"type:varchar(36);primaryKey"`
	ProductID   string        `json:"product_id" gorm:"type:varchar(36);index;not null"`
	Product     *CloudProduct `json:"product,omitempty" gorm:"foreignKey:ProductID"`
	Version     string        `json:"version" gorm:"type:varchar(80);index;not null"`
	Channel     string        `json:"channel" gorm:"type:varchar(30);default:stable;index"`
	Changelog   string        `json:"changelog" gorm:"type:text"`
	Status      string        `json:"status" gorm:"type:varchar(20);default:draft;index"`
	PublishedAt *time.Time    `json:"published_at"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (m *CloudRelease) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type CloudArtifact struct {
	ID        string        `json:"id" gorm:"type:varchar(36);primaryKey"`
	ReleaseID string        `json:"release_id" gorm:"type:varchar(36);index;not null"`
	Release   *CloudRelease `json:"release,omitempty" gorm:"foreignKey:ReleaseID"`
	OS        string        `json:"os" gorm:"type:varchar(50);index"`
	Arch      string        `json:"arch" gorm:"type:varchar(50);index"`
	FileName  string        `json:"file_name" gorm:"type:varchar(255);not null"`
	FileURL   string        `json:"file_url" gorm:"type:varchar(1000);not null"`
	Checksum  string        `json:"checksum" gorm:"type:varchar(128)"`
	Size      int64         `json:"size"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (m *CloudArtifact) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

type CloudDownloadLog struct {
	ID           string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	ArtifactID   string         `json:"artifact_id" gorm:"type:varchar(36);index;not null"`
	Artifact     *CloudArtifact `json:"artifact,omitempty" gorm:"foreignKey:ArtifactID"`
	IP           string         `json:"ip" gorm:"type:varchar(80);index"`
	UserAgent    string         `json:"user_agent" gorm:"type:text"`
	DownloadedAt time.Time      `json:"downloaded_at" gorm:"index"`
}

func (m *CloudDownloadLog) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if m.DownloadedAt.IsZero() {
		m.DownloadedAt = time.Now()
	}
	return nil
}

type CloudSiteContent struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Key       string    `json:"key" gorm:"type:varchar(120);uniqueIndex;not null"`
	Title     string    `json:"title" gorm:"type:varchar(255)"`
	BodyMD    string    `json:"body_md" gorm:"type:text"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:draft;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *CloudSiteContent) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}
