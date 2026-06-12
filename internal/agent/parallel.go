package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

// ============================================
// 并行工具执行器 - 支持并发执行多个互不依赖的工具调用
// ============================================

// ParallelCall 单个并行调用
type ParallelCall struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

// ParallelResult 并行执行结果
type ParallelResult struct {
	Index    int
	ToolName string
	Output   string
	Error    error
}

// executeParallelCalls 并行执行多个工具调用
// 用于报表等场景：多个独立查询可同时执行，然后聚合结果
func (e *AgentEngine) executeParallelCalls(ctx context.Context, user *UserContext, calls []ParallelCall, toolNames map[string]bool) []LoopStep {
	if len(calls) == 0 {
		return nil
	}

	// 边界检查 + 去重
	uniqueCalls := make([]ParallelCall, 0, len(calls))
	seen := make(map[string]bool)
	deferredPermChecks := make([]int, 0) // 记录被拒绝的调用索引，用于跳过
	for i, call := range calls {
		if !toolNames[call.ToolName] {
			log.Printf("[Parallel] 拒绝未授权工具: %s", call.ToolName)
			deferredPermChecks = append(deferredPermChecks, i)
			continue
		}
		key := call.ToolName + "_" + fmt.Sprintf("%v", call.ToolInput)
		if seen[key] {
			continue
		}
		seen[key] = true
		uniqueCalls = append(uniqueCalls, call)
	}
	_ = deferredPermChecks // 权限检查已执行，拒绝的调用已跳过

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make([]ParallelResult, len(uniqueCalls))
	)

	for i, call := range uniqueCalls {
		wg.Add(1)
		go func(idx int, c ParallelCall) {
			defer wg.Done()
			output, err := e.executeTool(ctx, user, c.ToolName, c.ToolInput)
			mu.Lock()
			results[idx] = ParallelResult{
				Index:    idx,
				ToolName: c.ToolName,
				Output:   output,
				Error:    err,
			}
			mu.Unlock()
		}(i, call)
	}

	wg.Wait()

	// 转换为 LoopStep
	steps := make([]LoopStep, len(uniqueCalls))
	for i, call := range uniqueCalls {
		r := results[i]
		step := LoopStep{
			StepID:    i,
			Action:    "parallel_call",
			ToolName:  r.ToolName,
			ToolInput: call.ToolInput,
		}
		if r.Error != nil {
			step.Error = r.Error.Error()
			step.ToolOutput = fmt.Sprintf("执行失败: %v", r.Error)
		} else {
			step.ToolOutput = r.Output
		}
		steps[i] = step
	}

	log.Printf("[Parallel] 并发执行 %d 个工具调用完成", len(uniqueCalls))
	return steps
}

// formatParallelResults 格式化并行执行结果为 observation 文本
func formatParallelResults(steps []LoopStep) string {
	var sb strings.Builder // sic: we know strings is imported in loop.go
	sb.WriteString(fmt.Sprintf("[并行执行结果] 共 %d 个查询:\n\n", len(steps)))
	for i, step := range steps {
		sb.WriteString(fmt.Sprintf("--- 查询 %d: %s ---\n", i+1, step.ToolName))
		if step.Error != "" {
			sb.WriteString(fmt.Sprintf("错误: %s\n", step.Error))
		} else {
			sb.WriteString(fmt.Sprintf("%s\n", step.ToolOutput))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
