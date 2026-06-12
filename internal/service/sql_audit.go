package service

import (
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type SQLAuditService struct {
	db *gorm.DB
}

func NewSQLAuditService(db *gorm.DB) *SQLAuditService {
	return &SQLAuditService{db: db}
}

// RecordSQL 实现 text2sql.AuditRecorder 接口
func (s *SQLAuditService) RecordSQL(userID, skillID, question, sql, dataSourceID, status string) (string, error) {
	audit := &models.SQLAudit{
		UserID:       userID,
		SkillID:      skillID,
		Question:     question,
		GeneratedSQL: sql,
		DataSourceID: dataSourceID,
		Status:       status,
	}
	if err := s.db.Create(audit).Error; err != nil {
		return "", err
	}
	return audit.ID, nil
}

// IsApproved 检查 SQL 是否已被审批通过
func (s *SQLAuditService) IsApproved(auditID string) bool {
	var audit models.SQLAudit
	if err := s.db.Where("id = ? AND status = ?", auditID, "approved").First(&audit).Error; err != nil {
		return false
	}
	return true
}

// ListPending 列出待审批的 SQL
func (s *SQLAuditService) ListPending() ([]models.SQLAudit, error) {
	var audits []models.SQLAudit
	err := s.db.Where("status = ?", "pending").Order("created_at DESC").Find(&audits).Error
	return audits, err
}

// ListAll 列出所有 SQL 审计记录
func (s *SQLAuditService) ListAll(status string) ([]models.SQLAudit, error) {
	query := s.db.Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var audits []models.SQLAudit
	err := query.Find(&audits).Error
	return audits, err
}

// Approve 审批通过
func (s *SQLAuditService) Approve(id, approvedBy string) error {
	now := time.Now()
	return s.db.Model(&models.SQLAudit{}).Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{
			"status":      "approved",
			"approved_by": approvedBy,
			"approved_at": now,
		}).Error
}

// Reject 拒绝
func (s *SQLAuditService) Reject(id, approvedBy, reason string) error {
	now := time.Now()
	return s.db.Model(&models.SQLAudit{}).Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{
			"status":        "rejected",
			"approved_by":   approvedBy,
			"approved_at":   now,
			"reject_reason": reason,
		}).Error
}

// MarkExecuted 标记 SQL 已执行
func (s *SQLAuditService) MarkExecuted(id string, rowCount int, execTime string) error {
	return s.db.Model(&models.SQLAudit{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":    "executed",
			"row_count": rowCount,
			"exec_time": execTime,
		}).Error
}
