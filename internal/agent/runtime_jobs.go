package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"github.com/alaikis/opentether/internal/templating"
	"gorm.io/gorm"
)

const runtimeJobLeaseDuration = 2 * time.Minute

func runtimeLeaseOwner() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}

func (e *AgentEngine) startRuntimeJob(user *UserContext, conversationID, query string, maxSteps int) *models.RuntimeJob {
	if e == nil || e.db == nil || user == nil {
		return nil
	}
	input, _ := json.Marshal(map[string]interface{}{"query": query})
	if user.Context != nil {
		if runtimeJobID, ok := user.Context["runtime_job_id"].(string); ok && runtimeJobID != "" {
			var existing models.RuntimeJob
			if err := e.db.Where("id = ?", runtimeJobID).First(&existing).Error; err == nil {
				updates := map[string]interface{}{
					"status":           "running",
					"input":            string(input),
					"max_steps":        maxSteps,
					"lease_owner":      runtimeLeaseOwner(),
					"lease_expires_at": time.Now().Add(runtimeJobLeaseDuration),
					"started_at":       time.Now(),
					"finished_at":      nil,
					"error":            "",
				}
				_ = e.db.Model(&models.RuntimeJob{}).Where("id = ?", existing.ID).Updates(updates).Error
				existing.Status = "running"
				existing.Input = string(input)
				existing.MaxSteps = maxSteps
				return &existing
			}
		}
	}
	job := &models.RuntimeJob{
		UserID:         user.UserID,
		ConversationID: conversationID,
		JobType:        "agent_loop",
		Status:         "running",
		Input:          string(input),
		MaxSteps:       maxSteps,
		Recoverable:    true,
		LeaseOwner:     runtimeLeaseOwner(),
		LeaseExpiresAt: time.Now().Add(runtimeJobLeaseDuration),
		StartedAt:      time.Now(),
	}
	if user.Context != nil {
		if skillID, ok := user.Context["selected_skill_id"].(string); ok {
			job.SkillID = skillID
		}
	}
	if err := e.db.Create(job).Error; err != nil {
		log.Printf("[RuntimeJob] 创建任务失败: %v", err)
		return nil
	}
	return job
}

func (e *AgentEngine) heartbeatRuntimeJob(job *models.RuntimeJob, step int) {
	if e == nil || e.db == nil || job == nil {
		return
	}
	_ = e.db.Model(&models.RuntimeJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"current_step":     step,
		"lease_owner":      runtimeLeaseOwner(),
		"lease_expires_at": time.Now().Add(runtimeJobLeaseDuration),
	}).Error
}

func (e *AgentEngine) saveRuntimeCheckpoint(job *models.RuntimeJob, step int, typ string, state interface{}) {
	if e == nil || e.db == nil || job == nil {
		return
	}
	summary := templating.SafeRender(runtimeCheckpointSummaryJinja, map[string]interface{}{"step": step, "type": typ, "summary": fmt.Sprintf("%v", state)}, fmt.Sprintf("第 %d 步 %s", step, typ))
	payload, _ := json.Marshal(map[string]interface{}{"summary": summary, "state": state})
	ckpt := &models.RuntimeCheckpoint{
		JobID:          job.ID,
		Step:           step,
		Type:           typ,
		State:          string(payload),
		IdempotencyKey: fmt.Sprintf("%s:%d:%s", job.ID, step, typ),
	}
	_ = e.db.Create(ckpt).Error
}

func (e *AgentEngine) finishRuntimeJob(job *models.RuntimeJob, status, output, errMsg string) {
	if e == nil || e.db == nil || job == nil {
		return
	}
	finished := time.Now()
	_ = e.db.Model(&models.RuntimeJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
		"status":      status,
		"output":      output,
		"error":       errMsg,
		"finished_at": &finished,
	}).Error
}

// RecoverRuntimeJobs scans orphaned runtime jobs after process restart.
// It marks expired running jobs as recovering/paused and keeps checkpoints for manual or future automatic resume.
func RecoverRuntimeJobs(db *gorm.DB) {
	if db == nil {
		return
	}
	now := time.Now()
	var jobs []models.RuntimeJob
	if err := db.Where("status IN ? AND lease_expires_at < ?", []string{"running", "recovering"}, now).Find(&jobs).Error; err != nil {
		log.Printf("[RuntimeJob] 恢复扫描失败: %v", err)
		return
	}
	for _, job := range jobs {
		status := "failed"
		errMsg := "进程重启或租约过期，任务已中断"
		if job.Recoverable {
			status = "paused"
			errMsg = "进程重启或租约过期，任务已暂停，可基于 checkpoint 恢复或重试"
		}
		_ = db.Model(&models.RuntimeJob{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
			"status": status,
			"error":  errMsg,
		}).Error
		ckpt := &models.RuntimeCheckpoint{
			JobID:          job.ID,
			Step:           job.CurrentStep,
			Type:           "recovery_detected",
			State:          fmt.Sprintf(`{"message":%q}`, errMsg),
			IdempotencyKey: fmt.Sprintf("%s:recovery", job.ID),
		}
		_ = db.Create(ckpt).Error
	}
	if len(jobs) > 0 {
		log.Printf("[RuntimeJob] 已处理 %d 个重启遗留任务", len(jobs))
	}
}
