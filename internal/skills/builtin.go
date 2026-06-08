package skills

// RegisterBuiltinSkills registers all built-in skills
func RegisterBuiltinSkills(reg *Registry) {
	// Register all built-in skills
	reg.Register(NewChatSkill())
	reg.Register(NewText2SQLSkill())
	reg.Register(NewFileProcessSkill())
	reg.Register(NewReportSkill())
	reg.Register(NewAPICallerSkill())
}

// RegisterGlobalBuiltinSkills registers built-in skills to the global registry
func RegisterGlobalBuiltinSkills() {
	RegisterBuiltinSkills(globalRegistry)
}

// ChatSkill is the basic chat skill
type ChatSkill struct {
	enabled bool
}

func NewChatSkill() *ChatSkill {
	return &ChatSkill{enabled: true}
}

func (s *ChatSkill) Name() string        { return "chat" }
func (s *ChatSkill) Description() string { return "General conversational skill for chatting with AI" }
func (s *ChatSkill) Type() string        { return "chat" }
func (s *ChatSkill) Enabled() bool       { return s.enabled }
func (s *ChatSkill) Schema() string      { return `{"type":"object","properties":{}}` }

func (s *ChatSkill) Execute(ctx *ExecutionContext) (*Result, error) {
	// For now, just echo the input as a placeholder
	// In the real implementation, this would call the LLM
	return &Result{
		Output:    "Chat skill: " + ctx.Input,
		SkillUsed: "chat",
	}, nil
}

// Text2SQLSkill converts natural language to SQL queries
type Text2SQLSkill struct {
	enabled bool
}

func NewText2SQLSkill() *Text2SQLSkill {
	return &Text2SQLSkill{enabled: true}
}

func (s *Text2SQLSkill) Name() string        { return "text2sql" }
func (s *Text2SQLSkill) Description() string { return "Converts natural language to SQL queries and executes them" }
func (s *Text2SQLSkill) Type() string        { return "text2sql" }
func (s *Text2SQLSkill) Enabled() bool       { return s.enabled }
func (s *Text2SQLSkill) Schema() string      { return `{"type":"object","properties":{"datasource_id":{"type":"string"},"schema":{"type":"string"}}}` }

func (s *Text2SQLSkill) Execute(ctx *ExecutionContext) (*Result, error) {
	// Placeholder implementation - in real implementation would:
	// 1. Analyze the input to extract intent
	// 2. Get schema information from the data source
	// 3. Construct SQL query using LLM
	// 4. Execute the query
	// 5. Format and return results
	return &Result{
		Output: "Text2SQL skill - 请配置数据源后使用",
		Data: map[string]interface{}{
			"type":        "text2sql",
			"data_source": ctx.DataSourceID,
		},
		SkillUsed: "text2sql",
	}, nil
}

// FileProcessSkill processes uploaded files
type FileProcessSkill struct {
	enabled bool
}

func NewFileProcessSkill() *FileProcessSkill {
	return &FileProcessSkill{enabled: true}
}

func (s *FileProcessSkill) Name() string        { return "file_process" }
func (s *FileProcessSkill) Description() string { return "Processes uploaded files (Excel, CSV, PDF, images, etc.)" }
func (s *FileProcessSkill) Type() string        { return "file_process" }
func (s *FileProcessSkill) Enabled() bool       { return s.enabled }
func (s *FileProcessSkill) Schema() string      { return `{"type":"object","properties":{"file_type":{"type":"string"},"operation":{"type":"string"}}}` }

func (s *FileProcessSkill) Execute(ctx *ExecutionContext) (*Result, error) {
	// Placeholder implementation
	return &Result{
		Output:    "File processing skill - 文件处理功能",
		SkillUsed: "file_process",
	}, nil
}

// ReportSkill generates reports from data
type ReportSkill struct {
	enabled bool
}

func NewReportSkill() *ReportSkill {
	return &ReportSkill{enabled: true}
}

func (s *ReportSkill) Name() string        { return "report" }
func (s *ReportSkill) Description() string { return "Generates reports and visualizations from data" }
func (s *ReportSkill) Type() string        { return "report" }
func (s *ReportSkill) Enabled() bool       { return s.enabled }
func (s *ReportSkill) Schema() string      { return `{"type":"object","properties":{"template":{"type":"string"},"format":{"type":"string"}}}` }

func (s *ReportSkill) Execute(ctx *ExecutionContext) (*Result, error) {
	// Placeholder implementation
	return &Result{
		Output:    "Report generation skill - 报表生成功能",
		Data: map[string]interface{}{
			"type": "report",
		},
		SkillUsed: "report",
	}, nil
}

// APICallerSkill calls external APIs
type APICallerSkill struct {
	enabled bool
}

func NewAPICallerSkill() *APICallerSkill {
	return &APICallerSkill{enabled: true}
}

func (s *APICallerSkill) Name() string        { return "api_caller" }
func (s *APICallerSkill) Description() string { return "Calls external HTTP APIs" }
func (s *APICallerSkill) Type() string        { return "api_caller" }
func (s *APICallerSkill) Enabled() bool       { return s.enabled }
func (s *APICallerSkill) Schema() string      { return `{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string"},"headers":{"type":"object"}}}` }

func (s *APICallerSkill) Execute(ctx *ExecutionContext) (*Result, error) {
	// Placeholder implementation
	return &Result{
		Output:    "API caller skill - API 调用功能",
		SkillUsed: "api_caller",
	}, nil
}
