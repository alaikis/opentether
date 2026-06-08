package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

// SkillFromMarkdownService 从 Markdown 文件创建 Skill
type SkillFromMarkdownService struct {
	db *gorm.DB
}

func NewSkillFromMarkdownService(db *gorm.DB) *SkillFromMarkdownService {
	return &SkillFromMarkdownService{db: db}
}

// ParseMarkdownToSkill 解析 MD 文件内容为 Skill
func (s *SkillFromMarkdownService) ParseMarkdownToSkill(markdownContent,filename string) (*ParsedSkill, error) {
	result := &ParsedSkill{
		SourceFile: filename,
	}

	// 提取标题 (第一个 # 标题)
	titleMatch := regexp.MustCompile(`^#\s+(.+)$`).FindStringSubmatch(markdownContent)
	if len(titleMatch) > 1 {
		result.Name = strings.TrimSpace(titleMatch[1])
	}

	// 提取 Frontmatter (--- 包裹的 YAML)
	frontmatterMatch := regexp.MustCompile(`(?s)^---\n(.+?)\n---`).FindStringSubmatch(markdownContent)
	if len(frontmatterMatch) > 1 {
		result.Frontmatter = frontmatterMatch[1]
		s.parseFrontmatter(result, frontmatterMatch[1])
	}

	// 移除 frontmatter 和标题，获取正文
	body := regexp.MustCompile(`(?s)^---.+?---\n`).ReplaceAllString(markdownContent, "")
	body = regexp.MustCompile(`(?s)^# .+\n`).ReplaceAllString(body, "")

	// 提取 Description (第一段)
	paragraphs := regexp.MustCompile(`\n\n+`).Split(strings.TrimSpace(body), -1)
	if len(paragraphs) > 0 {
		result.Description = strings.TrimSpace(paragraphs[0])
	}

	// 提取 Prompt Template (代码块或 ```prompt ... ``` 块)
	result.PromptTemplate = s.extractPromptTemplate(body)

	// 提取 Keywords (从标题、内容或 frontmatter)
	result.Keywords = s.extractKeywords(body, result.Frontmatter, result.Name)

	// 提取 Skill Type (从 frontmatter 或内容推断)
	result.SkillType = s.inferSkillType(body, result.Frontmatter)

	// 提取 MCP 配置 (如果存在)
	result.MCPConfig = s.extractMCPConfig(body)

	// 提取 API 配置 (如果存在)
	result.APIConfig = s.extractAPIConfig(body)

	// 提取数据源配置 (如果存在)
	result.DataSourceConfig = s.extractDataSourceConfig(body)

	return result, nil
}

type ParsedSkill struct {
	Name             string
	Description      string
	Keywords         string
	SkillType        string
	PromptTemplate   string
	Frontmatter      string
	SourceFile       string
	MCPConfig        string
	APIConfig        string
	DataSourceConfig string
	Category         string
	Enabled          bool
}

// parseFrontmatter 解析 YAML frontmatter
func (s *SkillFromMarkdownService) parseFrontmatter(result *ParsedSkill, frontmatter string) {
	// 解析 name
	if match := regexp.MustCompile(`(?m)^name:\s*(.+)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
		result.Name = strings.TrimSpace(match[1])
	}
	// 解析 skill_type
	if match := regexp.MustCompile(`(?m)^skill_type:\s*(.+)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
		result.SkillType = strings.TrimSpace(match[1])
	}
	// 解析 category
	if match := regexp.MustCompile(`(?m)^category:\s*(.+)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
		result.Category = strings.TrimSpace(match[1])
	}
	// 解析 keywords
	if match := regexp.MustCompile(`(?m)^keywords:\s*(.+)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
		result.Keywords = strings.TrimSpace(match[1])
	}
	// 解析 enabled
	if match := regexp.MustCompile(`(?m)^enabled:\s*(true|false)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
		result.Enabled = match[1] == "true"
	}
}

// extractPromptTemplate 提取 Prompt 模板
func (s *SkillFromMarkdownService) extractPromptTemplate(body string) string {
	// 尝试匹配 ```prompt ... ``` 或 ```system ... ``` 或 ```template ... ```
	patterns := []string{
		"(?s)```prompt\\s*\\n(.+?)```",
		"(?s)```system\\s*\\n(.+?)```",
		"(?s)```template\\s*\\n(.+?)```",
		"(?s)```instructions\\s*\\n(.+?)```",
	}

	for _, pattern := range patterns {
		if match := regexp.MustCompile(pattern).FindStringSubmatch(body); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}

	// 如果没有明确的 prompt 块，返回整个正文作为基础
	return ""
}

// extractKeywords 提取关键词
func (s *SkillFromMarkdownService) extractKeywords(body, frontmatter, title string) string {
	var keywords []string

	// 从 frontmatter 获取
	if frontmatter != "" {
		if match := regexp.MustCompile(`(?m)^keywords:\s*(.+)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
			keywords = append(keywords, strings.Split(match[1], ",")...)
		}
	}

	// 从标题提取词
	if title != "" {
		words := strings.FieldsFunc(title, func(r rune) bool {
			return r == ' ' || r == '-' || r == '_'
		})
		keywords = append(keywords, words...)
	}

	// 从内容中提取常用词
	commonWords := map[string]bool{
		"查询": true, "搜索": true, "分析": true, "生成": true,
		"处理": true, "转换": true, "导出": true,
		"员工": true, "业绩": true, "销售": true, "数据": true,
		"报表": true, "报告": true, "PDF": true, "文档": true,
	}

	words := regexp.MustCompile(`[\p{Lu}\p{Lt}][\p{Ll}]+`).FindAllString(body, -1)
	wordCount := make(map[string]int)
	for _, w := range words {
		lower := strings.ToLower(w)
		if commonWords[lower] || len(w) > 3 {
			wordCount[lower]++
		}
	}

	// 取出现频率最高的词
	for word, count := range wordCount {
		if count >= 2 && len(keywords) < 10 {
			keywords = append(keywords, word)
		}
	}

	return strings.Join(keywords, ",")
}

// inferSkillType 推断技能类型
func (s *SkillFromMarkdownService) inferSkillType(body, frontmatter string) string {
	// 从 frontmatter 获取
	if frontmatter != "" {
		if match := regexp.MustCompile(`(?m)^skill_type:\s*(.+)$`).FindStringSubmatch(frontmatter); len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}

	// 从内容推断
	lowerBody := strings.ToLower(body)

	typeIndicators := map[string][]string{
		"text2sql":   {"查询", "sql", "数据库", "select", "数据", "统计"},
		"api_caller": {"api", "接口", "http", "调用", "请求", "webhook"},
		"file_process": {"文件", "上传", "下载", "处理", "转换", "解析", "excel", "csv", "pdf"},
		"report":     {"报表", "报告", "生成", "导出", "pdf", "统计"},
		"employee":   {"员工", "业绩", "绩效", "考勤", "人员"},
		"chat":       {"对话", "聊天", "问答", "帮助"},
	}

	for skillType, indicators := range typeIndicators {
		matchCount := 0
		for _, indicator := range indicators {
			if strings.Contains(lowerBody, indicator) {
				matchCount++
			}
		}
		if matchCount >= 2 {
			return skillType
		}
	}

	return "chat"
}

// extractMCPConfig 提取 MCP 配置
func (s *SkillFromMarkdownService) extractMCPConfig(body string) string {
	// 匹配 ```mcp 配置块
	mcpPattern := "(?s)```mcp\\s*\\n(.+?)```"
	if match := regexp.MustCompile(mcpPattern).FindStringSubmatch(body); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	// 匹配 mcp: 配置项
	mcpInline := regexp.MustCompile(`(?m)^mcp:\s*(.+)$`).FindStringSubmatch(body)
	if len(mcpInline) > 1 {
		return strings.TrimSpace(mcpInline[1])
	}

	return ""
}

// extractAPIConfig 提取 API 配置
func (s *SkillFromMarkdownService) extractAPIConfig(body string) string {
	// 匹配 ```api 配置块
	apiPattern := "(?s)```api\\s*\\n(.+?)```"
	if match := regexp.MustCompile(apiPattern).FindStringSubmatch(body); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	// 匹配 endpoint, url 等配置
	var configs []string

	if match := regexp.MustCompile(`(?m)^endpoint:\s*(.+)$`).FindStringSubmatch(body); len(match) > 1 {
		configs = append(configs, fmt.Sprintf("endpoint:%s", strings.TrimSpace(match[1])))
	}
	if match := regexp.MustCompile(`(?m)^url:\s*(.+)$`).FindStringSubmatch(body); len(match) > 1 {
		configs = append(configs, fmt.Sprintf("url:%s", strings.TrimSpace(match[1])))
	}
	if match := regexp.MustCompile(`(?m)^method:\s*(GET|POST|PUT|DELETE)$`).FindStringSubmatch(body); len(match) > 1 {
		configs = append(configs, fmt.Sprintf("method:%s", strings.TrimSpace(match[1])))
	}

	if len(configs) > 0 {
		return strings.Join(configs, ",")
	}

	return ""
}

// extractDataSourceConfig 提取数据源配置
func (s *SkillFromMarkdownService) extractDataSourceConfig(body string) string {
	// 匹配 ```datasource 配置块
	dsPattern := "(?s)```datasource\\s*\\n(.+?)```"
	if match := regexp.MustCompile(dsPattern).FindStringSubmatch(body); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	// 匹配数据库配置
	var configs []string

	if match := regexp.MustCompile(`(?m)^database:\s*(.+)$`).FindStringSubmatch(body); len(match) > 1 {
		configs = append(configs, fmt.Sprintf("database:%s", strings.TrimSpace(match[1])))
	}
	if match := regexp.MustCompile(`(?m)^table:\s*(.+)$`).FindStringSubmatch(body); len(match) > 1 {
		configs = append(configs, fmt.Sprintf("table:%s", strings.TrimSpace(match[1])))
	}

	if len(configs) > 0 {
		return strings.Join(configs, ",")
	}

	return ""
}

// CreateSkillFromParsed 从解析结果创建 Skill
func (s *SkillFromMarkdownService) CreateSkillFromParsed(parsed *ParsedSkill) (*models.Skill, error) {
	skill := &models.Skill{
		Name:            parsed.Name,
		SkillType:       parsed.SkillType,
		Description:     parsed.Description,
		Keywords:        parsed.Keywords,
		Category:        parsed.Category,
		Enabled:         parsed.Enabled,
		PromptTemplate:  parsed.PromptTemplate,
		VectorEnabled:   false,
	}

	// 构建配置 JSON
	config := map[string]interface{}{
		"source_file": parsed.SourceFile,
	}

	if parsed.MCPConfig != "" {
		config["mcp"] = parsed.MCPConfig
	}
	if parsed.APIConfig != "" {
		config["api"] = parsed.APIConfig
	}
	if parsed.DataSourceConfig != "" {
		config["datasource"] = parsed.DataSourceConfig
	}

	// 序列化配置
	configJSON, _ := json.Marshal(config)
	skill.Config = string(configJSON)

	// 保存到数据库
	if err := s.db.Create(skill).Error; err != nil {
		return nil, err
	}

	return skill, nil
}

// UpdateSkillFromParsed 更新已存在的 Skill
func (s *SkillFromMarkdownService) UpdateSkillFromParsed(skillID string, parsed *ParsedSkill) error {
	updates := map[string]interface{}{
		"name":            parsed.Name,
		"skill_type":      parsed.SkillType,
		"description":     parsed.Description,
		"keywords":        parsed.Keywords,
		"category":        parsed.Category,
		"enabled":         parsed.Enabled,
		"prompt_template": parsed.PromptTemplate,
	}

	// 更新配置
	var skill models.Skill
	if err := s.db.First(&skill, skillID).Error; err != nil {
		return err
	}

	config := make(map[string]interface{})
	json.Unmarshal([]byte(skill.Config), &config)

	config["source_file"] = parsed.SourceFile
	if parsed.MCPConfig != "" {
		config["mcp"] = parsed.MCPConfig
	}
	if parsed.APIConfig != "" {
		config["api"] = parsed.APIConfig
	}
	if parsed.DataSourceConfig != "" {
		config["datasource"] = parsed.DataSourceConfig
	}

	configJSON, _ := json.Marshal(config)
	updates["config"] = string(configJSON)

	return s.db.Model(&skill).Updates(updates).Error
}
