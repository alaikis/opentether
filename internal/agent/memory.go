package agent

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// ============================================
// Long-term Memory System - 长期记忆
//
// 用户记忆: 偏好、历史上下文、常用操作
// 用户组记忆: 共享知识、团队规则
// 类型定义见 models.UserMemory, models.GroupMemory, models.AgentScript
// ============================================

// UserMemory 用户长期记忆 (alias to models)
type UserMemory = models.UserMemory

// GroupMemory 用户组共享记忆 (alias to models)
type GroupMemory = models.GroupMemory

// AgentScript 生成的脚本 (alias to models)
type AgentScript = models.AgentScript

// LongTermMemory 长期记忆管理器
type LongTermMemory struct {
	db *gorm.DB
}

func NewLongTermMemory(db *gorm.DB) *LongTermMemory {
	return &LongTermMemory{db: db}
}

// SaveUserMemory 保存用户记忆
func (m *LongTermMemory) SaveUserMemory(userID, memType, key, content string, priority int) error {
	var existing UserMemory
	err := m.db.Where("user_id = ? AND type = ? AND key = ?", userID, memType, key).First(&existing).Error
	if err == nil {
		existing.Content = content
		existing.Priority = priority
		return m.db.Save(&existing).Error
	}

	mem := &UserMemory{
		UserID:   userID,
		Type:     memType,
		Key:      key,
		Content:  content,
		Priority: priority,
	}
	return m.db.Create(mem).Error
}

// GetUserMemory 获取用户记忆
func (m *LongTermMemory) GetUserMemory(userID, memType string) ([]UserMemory, error) {
	var memories []UserMemory
	query := m.db.Where("user_id = ?", userID)
	if memType != "" {
		query = query.Where("type = ?", memType)
	}
	err := query.Order("priority DESC").Find(&memories).Error
	return memories, err
}

// GetUserMemoryByKey 按 key 获取
func (m *LongTermMemory) GetUserMemoryByKey(userID, key string) (*UserMemory, error) {
	var mem UserMemory
	err := m.db.Where("user_id = ? AND key = ?", userID, key).First(&mem).Error
	if err != nil {
		return nil, err
	}
	return &mem, nil
}

// SaveGroupMemory 保存用户组记忆
func (m *LongTermMemory) SaveGroupMemory(groupID, memType, key, content string) error {
	var existing GroupMemory
	err := m.db.Where("group_id = ? AND type = ? AND key = ?", groupID, memType, key).First(&existing).Error
	if err == nil {
		existing.Content = content
		return m.db.Save(&existing).Error
	}

	mem := &GroupMemory{
		GroupID: groupID,
		Type:    memType,
		Key:     key,
		Content: content,
	}
	return m.db.Create(mem).Error
}

// GetGroupMemories 获取用户组记忆
func (m *LongTermMemory) GetGroupMemories(groupIDs []string) ([]GroupMemory, error) {
	var memories []GroupMemory
	err := m.db.Where("group_id IN ?", groupIDs).Order("priority DESC").Find(&memories).Error
	return memories, err
}

// InjectMemoryIntoPrompt 将长期记忆注入 prompt
func (m *LongTermMemory) InjectMemoryIntoPrompt(userID string, groupIDs []string) string {
	var sb strings.Builder

	// 用户记忆
	userMems, _ := m.GetUserMemory(userID, "")
	if len(userMems) > 0 {
		sb.WriteString("## 用户记忆\n")
		for _, mem := range userMems {
			sb.WriteString(fmt.Sprintf("- [%s] %s: %s\n", mem.Type, mem.Key, truncateStr(mem.Content, 200)))
		}
	}

	// 组记忆
	groupMems, _ := m.GetGroupMemories(groupIDs)
	if len(groupMems) > 0 {
		sb.WriteString("\n## 团队认知\n")
		for _, mem := range groupMems {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", mem.Key, truncateStr(mem.Content, 200)))
		}
	}

	return sb.String()
}

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
	// 尝试生成 bash 脚本
	lang := "bash"
	content := m.generateBashScript(name, description, promptTemplate)
	if content == "" {
		// 回退到 Python
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

// generateBashScript 生成 bash 脚本骨架
func (m *ScriptManager) generateBashScript(name, description, template string) string {
	var sb strings.Builder
	sb.WriteString("#!/bin/bash\n")
	sb.WriteString(fmt.Sprintf("# %s - 自动生成脚本\n", name))
	sb.WriteString(fmt.Sprintf("# %s\n", description))
	sb.WriteString(fmt.Sprintf("# 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString("# ========================================\n")
	sb.WriteString("# 用户配置区域\n")
	sb.WriteString("# ========================================\n")
	sb.WriteString("DB_HOST=\"${DB_HOST:-localhost}\"\n")
	sb.WriteString("DB_PORT=\"${DB_PORT:-3306}\"\n")
	sb.WriteString("DB_USER=\"${DB_USER:-root}\"\n")
	sb.WriteString("DB_NAME=\"${DB_NAME:-opentether}\"\n\n")
	sb.WriteString("# ========================================\n")
	sb.WriteString("# 任务逻辑\n")
	sb.WriteString("# ========================================\n")

	// 从 prompt template 提取任务
	if strings.Contains(template, "sql") || strings.Contains(template, "查询") || strings.Contains(template, "数据") {
		sb.WriteString("# 数据查询任务\n")
		sb.WriteString("echo \"[信息] 执行数据查询...\"\n")
		sb.WriteString("mysql -h \"$DB_HOST\" -P \"$DB_PORT\" -u \"$DB_USER\" -D \"$DB_NAME\" -e \"SELECT '请在此处填入SQL查询'; \"\n")
	} else if strings.Contains(template, "文件") || strings.Contains(template, "上传") || strings.Contains(template, "下载") {
		sb.WriteString("# 文件处理任务\n")
		sb.WriteString("echo \"[信息] 执行文件处理...\"\n")
		sb.WriteString("INPUT_FILE=\"${1:-/tmp/input.csv}\"\n")
		sb.WriteString("OUTPUT_FILE=\"${2:-/tmp/output.csv}\"\n")
		sb.WriteString("echo \"输入: $INPUT_FILE\"\n")
		sb.WriteString("echo \"输出: $OUTPUT_FILE\"\n")
	} else {
		sb.WriteString("# 通用任务\n")
		sb.WriteString(fmt.Sprintf("echo \"[信息] 执行: %s\"\n\n", name))
		sb.WriteString("# 初始化\n")
		sb.WriteString("echo \"[OK] 环境准备完成\"\n")
	}

	sb.WriteString("\n# ========================================\n")
	sb.WriteString("# 完成\n")
	sb.WriteString("# ========================================\n")
	sb.WriteString("echo \"[完成] 脚本执行结束\"\n")
	sb.WriteString("exit 0\n")
	return sb.String()
}

// generatePythonScript 生成 python 脚本骨架
func (m *ScriptManager) generatePythonScript(name, description, template string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env python3\n")
	sb.WriteString(fmt.Sprintf("\"\"\"%s - 自动生成脚本\"\"\"\n", name))
	sb.WriteString("import os\nimport sys\nimport json\n\n")
	sb.WriteString("def main():\n")
	sb.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n", description))
	sb.WriteString("    print(\"[信息] 执行任务...\")\n")

	if strings.Contains(template, "sql") || strings.Contains(template, "查询") {
		sb.WriteString("    # TODO: 实现数据库查询逻辑\n")
	}
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

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
