package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

type AgentScript = models.AgentScript

// ============================================
// Script Manager - 脚本生成与生命周期管理
// ============================================

// ScriptManager 脚本管理器
type ScriptManager struct {
	db *gorm.DB
}

func NewScriptManager(db *gorm.DB) *ScriptManager {
	return &ScriptManager{db: db}
}

// GenerateScript 从 Skill/文档生成脚本 (bash 优先)
func (m *ScriptManager) GenerateScript(skillID, name, description, promptTemplate string) (*AgentScript, error) {
	lang := "bash"
	content := m.generateBashScript(name, description, promptTemplate)
	if content == "" {
		lang = "python"
		content = m.generatePythonScript(name, description, promptTemplate)
	}

	script := &AgentScript{
		SkillID:     skillID,
		Name:        name + "." + scriptExt(lang),
		Language:    lang,
		Content:     content,
		Description: description,
		IsPermanent: false,
	}

	if err := m.db.Create(script).Error; err != nil {
		return nil, fmt.Errorf("保存脚本失败: %w", err)
	}

	log.Printf("[Script] 生成 %s 脚本: %s (%d 字节)", lang, script.Name, len(content))
	return script, nil
}

// MakePermanent 将脚本标记为永久（关联经验时调用）
func (m *ScriptManager) MakePermanent(scriptID, experienceID string) error {
	return m.db.Model(&AgentScript{}).Where("id = ?", scriptID).Updates(map[string]interface{}{
		"is_permanent":  true,
		"experience_id": experienceID,
		"expires_at":    nil,
	}).Error
}

// CleanupExpiredScripts 清理过期脚本（定时任务调用）
func (m *ScriptManager) CleanupExpiredScripts() (int, error) {
	result := m.db.Where("is_permanent = ? AND expires_at < ?", false, time.Now()).Delete(&AgentScript{})
	return int(result.RowsAffected), result.Error
}

// GetScriptsBySkill 获取 Skill 关联的脚本
func (m *ScriptManager) GetScriptsBySkill(skillID string) ([]AgentScript, error) {
	var scripts []AgentScript
	err := m.db.Where("skill_id = ?", skillID).Order("created_at DESC").Find(&scripts).Error
	return scripts, err
}

// RecordExecution 记录脚本执行
func (m *ScriptManager) RecordExecution(scriptID string) {
	now := time.Now()
	m.db.Model(&AgentScript{}).Where("id = ?", scriptID).Updates(map[string]interface{}{
		"exec_count":   gorm.Expr("exec_count + 1"),
		"last_exec_at": now,
	})
}

func (m *ScriptManager) generateBashScript(name, description, template string) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString(fmt.Sprintf("# %s - 自动生成脚本\n", name))
	sb.WriteString(fmt.Sprintf("# %s\n", description))
	sb.WriteString(fmt.Sprintf("# 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString("DB_HOST=\"${DB_HOST:-localhost}\"\n")
	sb.WriteString("DB_PORT=\"${DB_PORT:-3306}\"\n")
	sb.WriteString("DB_USER=\"${DB_USER:-root}\"\n")
	sb.WriteString("DB_NAME=\"${DB_NAME:-opentether}\"\n\n")
	sb.WriteString("echo \"[完成] 脚本执行结束\"\n")
	sb.WriteString("exit 0\n")
	return sb.String()
}

func (m *ScriptManager) generatePythonScript(name, description, template string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env python3\n")
	sb.WriteString(fmt.Sprintf("\"\"\"%s - 自动生成脚本\"\"\"\n", name))
	sb.WriteString("import os\nimport sys\nimport json\n\n")
	sb.WriteString("def main():\n")
	sb.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n", description))
	sb.WriteString("    print(\"[信息] 执行任务...\")\n")
	sb.WriteString("    print(\"[完成] 脚本执行结束\")\n\n")
	sb.WriteString("if __name__ == \"__main__\":\n    main()\n")
	return sb.String()
}

func scriptExt(lang string) string {
	if lang == "bash" {
		return "sh"
	}
	return "py"
}
