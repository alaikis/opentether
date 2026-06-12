package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// ============================================
// Agent Experience System — 智能体经验积累与复用
//
// 将成功的多步操作路径存储为"经验"，后续相似查询直接复用，省去重复的 LLM 推理。
// 所有经验需管理员审核后才能激活。支持全局经验和个人经验。
// ============================================

const (
	ExpStatusPending  = "pending_review" // 待审核
	ExpStatusActive   = "active"         // 已激活
	ExpStatusRejected = "rejected"       // 已拒绝
	ExpStatusDisabled = "disabled"       // 已禁用

	ExpScopeGlobal = "global" // 全局经验
	ExpScopeUser   = "user:"  // 个人经验 (前缀)
)

// AgentExperience 经验记录 (type alias to models)
type AgentExperience = models.AgentExperience

// ExperienceManager 经验管理器
type ExperienceManager struct {
	db *gorm.DB
}

func NewExperienceManager(db *gorm.DB) *ExperienceManager {
	return &ExperienceManager{db: db}
}

// MatchExperience 匹配经验（关键词 + 语义）
// 返回最佳匹配的经验和匹配分数
func (m *ExperienceManager) MatchExperience(userID, query string) (*AgentExperience, float64, error) {
	var experiences []AgentExperience

	// 查询范围: 全局激活 + 该用户的个人激活经验
	err := m.db.Where(
		"status = ? AND (scope = ? OR scope = ?)",
		ExpStatusActive, ExpScopeGlobal, ExpScopeUser+userID,
	).Find(&experiences).Error
	if err != nil {
		return nil, 0, err
	}

	if len(experiences) == 0 {
		return nil, 0, nil
	}

	queryLower := strings.ToLower(query)
	var bestMatch *AgentExperience
	bestScore := 0.0

	for i := range experiences {
		exp := &experiences[i]

		// 解析触发模式
		var patterns []string
		if err := json.Unmarshal([]byte(exp.TriggerPattern), &patterns); err != nil {
			continue
		}

		// 关键词匹配
		score := 0.0
		for _, pattern := range patterns {
			patternLower := strings.ToLower(pattern)
			if strings.Contains(queryLower, patternLower) {
				score += 1.0
			}
			// 部分匹配加分
			if strings.Contains(patternLower, queryLower) || strings.Contains(queryLower, patternLower) {
				score += 0.5
			}
		}

		// 匹配度归一化
		if len(patterns) > 0 {
			score = score / float64(len(patterns))
		}

		// 使用次数加权（高频经验更可靠）
		if exp.UsageCount > 0 {
			score *= (1.0 + float64(exp.UsageCount)*0.01)
		}

		if score > bestScore && score >= 0.6 {
			bestScore = score
			bestMatch = exp
		}
	}

	return bestMatch, bestScore, nil
}

// TrySaveExperience 尝试将成功执行步骤保存为待审核经验
// 只在多步操作（>1 个工具调用）且未匹配已有经验时才保存
func (m *ExperienceManager) TrySaveExperience(userID, query string, steps []LoopStep, tokensUsed int) error {
	// 只有多步操作才值得保存
	toolCount := 0
	for _, s := range steps {
		if s.Action == "tool_call" || s.Action == "parallel_call" {
			toolCount++
		}
	}
	if toolCount <= 1 {
		return nil
	}

	// 检查是否已有类似经验
	existing, _, _ := m.MatchExperience(userID, query)
	if existing != nil {
		// 更新使用统计
		m.db.Model(existing).Updates(map[string]interface{}{
			"usage_count":      gorm.Expr("usage_count + 1"),
			"avg_tokens_saved": (existing.AvgTokensSaved + tokensUsed) / 2,
		})
		return nil
	}

	// 提取触发关键词（从 query 中提取有意义的词）
	patterns := extractTriggerPatterns(query, steps)

	// 序列化步骤
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return err
	}
	patternsJSON, _ := json.Marshal(patterns)

	// 生成经验名称（从 query 截取）
	name := query
	if len([]rune(name)) > 50 {
		name = string([]rune(name)[:50]) + "..."
	}

	exp := &AgentExperience{
		Name:           name,
		Description:    fmt.Sprintf("来自用户 %s 的查询: %s", userID, query),
		TriggerPattern: string(patternsJSON),
		TriggerVector:  "[]",
		Steps:          string(stepsJSON),
		Scope:          ExpScopeUser + userID, // 默认个人经验
		Status:         ExpStatusPending,
		CreatedBy:      userID,
	}

	if err := m.db.Create(exp).Error; err != nil {
		return fmt.Errorf("保存经验失败: %w", err)
	}

	log.Printf("[Experience] 新经验已保存（待审核）: %s (关键词: %v)", exp.ID, patterns)
	return nil
}

// 内置工具名列表（硬编码在 executeTool 中，不依赖 skill 数据库记录）
var builtinToolNames = map[string]bool{
	"chat":            true,
	"text2sql":        true,
	"generate_pdf":    true,
	"generate_report": true,
	"setup_env":       true,
	"execute_script":  true,
}

// isToolAvailable 检查工具是否可用（内置工具或对应 skil 已启用）
func isToolAvailable(db *gorm.DB, toolName string) bool {
	if builtinToolNames[toolName] {
		return true
	}
	// MCP 工具始终允许（在 executeTool 中会检查 MCP 提供器）
	if strings.HasPrefix(toolName, "mcp__") {
		return true
	}
	// 检查是否有对应的已启用 Skill
	var count int64
	db.Model(&models.Skill{}).Where("enabled = ? AND (config LIKE ? OR skill_type = ?)", true, `%"tool":"`+toolName+`"%`, toolName).Count(&count)
	return count > 0
}

// ExecuteExperience 按经验步骤执行（跳过 LLM 推理）
func (m *ExperienceManager) ExecuteExperience(ctx context.Context, engine *AgentEngine, user *UserContext, exp *AgentExperience) (*ChatResponse, error) {
	var steps []LoopStep
	if err := json.Unmarshal([]byte(exp.Steps), &steps); err != nil {
		return nil, fmt.Errorf("解析经验步骤失败: %w", err)
	}

	log.Printf("[Experience] 复用经验 %s (%d 步), 节省 token", exp.Name, len(steps))

	// 按步骤执行（串行，因为步骤间可能有依赖）
	var observations []string
	for _, step := range steps {
		if step.Action == "tool_call" || step.Action == "parallel_call" {
			// 验证工具在当前环境中仍可用
			if !isToolAvailable(engine.db, step.ToolName) {
				observations = append(observations, fmt.Sprintf("[%s 不可用] 该工具对应的 Skill 已被删除或禁用，跳过此步骤", step.ToolName))
				continue
			}
			result, err := engine.executeTool(ctx, user, step.ToolName, step.ToolInput)
			if err != nil {
				observations = append(observations, fmt.Sprintf("[%s 失败] %v", step.ToolName, err))
			} else {
				observations = append(observations, fmt.Sprintf("[%s] %s", step.ToolName, result))
			}
		}
	}

	// 简单组装结果
	var sb strings.Builder
	for _, obs := range observations {
		sb.WriteString(obs + "\n\n")
	}

	// 更新使用统计
	m.db.Model(exp).Updates(map[string]interface{}{
		"usage_count":   gorm.Expr("usage_count + 1"),
		"success_count": gorm.Expr("success_count + 1"),
	})

	return &ChatResponse{
		Message:    sb.String(),
		SkillUsed:  "experience:" + exp.Name,
		TokensUsed: 0, // token 节省量
		Data: map[string]interface{}{
			"experience_id":   exp.ID,
			"experience_name": exp.Name,
			"token_saved":     true,
		},
	}, nil
}

// ListPendingExperiences 列出待审核经验
func (m *ExperienceManager) ListPendingExperiences() ([]AgentExperience, error) {
	var exps []AgentExperience
	err := m.db.Where("status = ?", ExpStatusPending).
		Order("created_at DESC").
		Find(&exps).Error
	return exps, err
}

// ListActiveExperiences 列出已激活经验
func (m *ExperienceManager) ListActiveExperiences(scope string) ([]AgentExperience, error) {
	query := m.db.Where("status = ?", ExpStatusActive)
	if scope != "" {
		if scope == "global" {
			query = query.Where("scope = ?", ExpScopeGlobal)
		} else if strings.HasPrefix(scope, "user:") {
			query = query.Where("scope = ? OR scope = ?", ExpScopeGlobal, scope)
		}
	}
	var exps []AgentExperience
	err := query.Order("usage_count DESC").Find(&exps).Error
	return exps, err
}

// ReviewExperience 审核经验
func (m *ExperienceManager) ReviewExperience(expID, reviewerID, status, note string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":      status,
		"reviewed_by": reviewerID,
		"reviewed_at": &now,
		"review_note": note,
	}

	if status == ExpStatusActive {
		// 激活时检查 scope 是否为 user:xxx，如果是则保持（PromoteToGlobal 负责升级为全局）
		var exp AgentExperience
		if err := m.db.Where("id = ?", expID).First(&exp).Error; err == nil {
			if strings.HasPrefix(exp.Scope, ExpScopeUser) {
				log.Printf("[ExperienceManager] 激活个人经验 %s (scope=%s)，保持用户级别", expID, exp.Scope)
			}
		}
	}

	return m.db.Model(&AgentExperience{}).Where("id = ?", expID).Updates(updates).Error
}

// PromoteToGlobal 将个人经验升级为全局经验（管理员操作）
func (m *ExperienceManager) PromoteToGlobal(expID string) error {
	return m.db.Model(&AgentExperience{}).Where("id = ?", expID).Update("scope", ExpScopeGlobal).Error
}

// DeleteExperience 删除经验
func (m *ExperienceManager) DeleteExperience(expID string) error {
	return m.db.Where("id = ?", expID).Delete(&AgentExperience{}).Error
}

// extractTriggerPatterns 从查询和步骤中提取触发关键词
func extractTriggerPatterns(query string, steps []LoopStep) []string {
	patterns := make([]string, 0)
	seen := make(map[string]bool)

	// 从查询中提取有意义的词
	words := strings.Fields(query)
	for _, w := range words {
		w = strings.Trim(w, "，,。.!！？?、")
		if len([]rune(w)) >= 2 && !seen[w] {
			patterns = append(patterns, w)
			seen[w] = true
		}
		if len(patterns) >= 8 {
			break
		}
	}

	// 从工具名提取
	for _, s := range steps {
		if s.ToolName != "" && !seen[s.ToolName] {
			patterns = append(patterns, s.ToolName)
			seen[s.ToolName] = true
		}
		if len(patterns) >= 10 {
			break
		}
	}

	return patterns
}
