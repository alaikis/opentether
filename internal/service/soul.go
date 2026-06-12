package service

import (
	"github.com/alaikis/opentether/internal/agent"
	"github.com/alaikis/opentether/internal/storage"
	"gorm.io/gorm"
)

type SoulService struct {
	db    *gorm.DB
	store storage.Driver
}

func NewSoulService(db *gorm.DB, store storage.Driver) *SoulService {
	return &SoulService{db: db, store: store}
}

func (s *SoulService) letm() *agent.LettaMemory {
	return agent.NewLettaMemory(s.db)
}

// --- 用户 Soul ---

func (s *SoulService) GetUserSoul(userID string) *agent.UserProfile {
	return s.letm().GetUserSoul(userID)
}

func (s *SoulService) UpsertUserSoul(userID, persona, human, preferences string) error {
	return s.letm().UpsertUserSoul(userID, persona, human, preferences)
}

// --- 公司 Soul ---

func (s *SoulService) GetCompanySoul() *agent.CompanyProfile {
	return s.letm().GetCompanySoul()
}

func (s *SoulService) UpsertCompanySoul(name, persona, brandTone, industry string) error {
	return s.letm().UpsertCompanySoul(name, persona, brandTone, industry)
}

// --- 记忆 ---

func (s *SoulService) ListUserMemories(userID string) ([]agent.UserMemory, error) {
	return s.letm().GetRecentUserMemories(userID, 50)
}

func (s *SoulService) DeleteUserMemory(id string) error {
	return s.db.Delete(&agent.UserMemory{}, "id = ?", id).Error
}

func (s *SoulService) ListGroupMemories(groupID string) ([]agent.GroupMemory, error) {
	return s.letm().RecallGroupMemories([]string{groupID}, "", 50)
}
