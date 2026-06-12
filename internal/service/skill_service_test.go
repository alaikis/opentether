package service

import (
	"testing"

	"github.com/alaikis/opentether/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestSkillService(t *testing.T) (*SkillService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.Skill{}); err != nil {
		t.Fatalf("migrate skills: %v", err)
	}

	return NewSkillService(db, false), db
}

func TestSkillServiceProtectsBuiltinSkillFromUpdateAndDelete(t *testing.T) {
	svc, db := newTestSkillService(t)

	skill := models.Skill{
		Name:      "PDF 报表",
		SkillType: "pdf",
		Category:  "系统内置",
		Enabled:   true,
		Config:    `{"builtin":true,"tool":"generate_pdf"}`,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create builtin skill: %v", err)
	}

	if _, err := svc.Update(skill.ID, UpdateSkillInput{Name: "改名", Enabled: false}); err == nil {
		t.Fatal("expected builtin skill update to be rejected")
	}
	if err := svc.Delete(skill.ID); err == nil {
		t.Fatal("expected builtin skill delete to be rejected")
	}

	var got models.Skill
	if err := db.Where("id = ?", skill.ID).First(&got).Error; err != nil {
		t.Fatalf("reload builtin skill: %v", err)
	}
	if got.Name != "PDF 报表" || !got.Enabled {
		t.Fatalf("builtin skill was modified: name=%q enabled=%v", got.Name, got.Enabled)
	}
}

func TestSkillServiceProtectsBuiltinSkillByConfigFlag(t *testing.T) {
	svc, db := newTestSkillService(t)

	skill := models.Skill{
		Name:      "内置但非系统分类",
		SkillType: "chat",
		Category:  "其他",
		Enabled:   true,
		Config:    `{"builtin":true}`,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create config builtin skill: %v", err)
	}

	if _, err := svc.Update(skill.ID, UpdateSkillInput{Name: "改名", Enabled: false}); err == nil {
		t.Fatal("expected builtin config flag update to be rejected")
	}
}

func TestSkillServiceAllowsCustomSkillUpdateAndDelete(t *testing.T) {
	svc, db := newTestSkillService(t)

	skill := models.Skill{
		Name:      "自定义 Skill",
		SkillType: "chat",
		Category:  "自定义",
		Enabled:   true,
		Config:    `{"builtin":false}`,
	}
	if err := db.Create(&skill).Error; err != nil {
		t.Fatalf("create custom skill: %v", err)
	}

	updated, err := svc.Update(skill.ID, UpdateSkillInput{
		Name:        "已更新 Skill",
		Description: "updated",
		Enabled:     false,
		Config:      `{"builtin":false}`,
	})
	if err != nil {
		t.Fatalf("update custom skill: %v", err)
	}
	if updated.Name != "已更新 Skill" || updated.Enabled {
		t.Fatalf("custom skill update not applied: name=%q enabled=%v", updated.Name, updated.Enabled)
	}

	if err := svc.Delete(skill.ID); err != nil {
		t.Fatalf("delete custom skill: %v", err)
	}
}
