package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/models"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler manages scheduled tasks
type Scheduler struct {
	db       *gorm.DB
	config   *config.Config
	cron     *cron.Cron
	running  bool
	jobIDs   map[string]cron.EntryID // task ID -> cron entry ID
	stopChan chan struct{}
}

// CronJob is kept for compatibility but not used internally
type CronJob struct {
	ID         string
	Name       string
	Expression string
	Executor   Executor
	NextRun    time.Time
	Enabled    bool
}

// Executor interface for running tasks
type Executor interface {
	Execute(ctx context.Context, params map[string]interface{}) (string, error)
}

// NewScheduler creates a new scheduler
func NewScheduler(db *gorm.DB, cfg *config.Config) *Scheduler {
	return &Scheduler{
		db:       db,
		config:   cfg,
		cron:     cron.New(cron.WithChain()),
		jobIDs:   make(map[string]cron.EntryID),
		stopChan: make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	if s.running {
		return nil
	}
	s.running = true

	log.Println("Scheduler starting...")

	// Load tasks from database
	if err := s.loadTasks(); err != nil {
		log.Printf("Failed to load tasks: %v", err)
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Println("Scheduler started")

	// Start the task loader in background
	go s.taskLoader()

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	if !s.running {
		return nil
	}

	s.running = false
	s.cron.Stop()
	close(s.stopChan)

	log.Println("Scheduler stopped")
	return nil
}

// loadTasks loads tasks from the database and schedules them
func (s *Scheduler) loadTasks() error {
	var tasks []models.ScheduledTask
	if err := s.db.Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		return err
	}

	for _, task := range tasks {
		if err := s.scheduleTask(task); err != nil {
			log.Printf("Failed to schedule task %s: %v", task.Name, err)
		}
	}

	return nil
}

// scheduleTask schedules a single task
func (s *Scheduler) scheduleTask(task models.ScheduledTask) error {
	// Remove existing job if any
	if entryID, exists := s.jobIDs[task.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobIDs, task.ID)
	}

	// Parse cron expression
	schedule, err := cron.ParseStandard(task.CronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Create executor
	executor := s.getExecutor(task.ExecutorType, task.ScriptContent, task.ScriptPath)

	// Add job to cron
	entryID, err := s.cron.AddJob(task.CronExpression, &taskRunner{
		taskID:    task.ID,
		taskName:  task.Name,
		db:        s.db,
		executor:  executor,
		parameters: task.Parameters,
	})
	if err != nil {
		return fmt.Errorf("failed to add job: %w", err)
	}

	s.jobIDs[task.ID] = entryID
	log.Printf("Scheduled task: %s (next run: %s)", task.Name, schedule.Next(time.Now()))

	return nil
}

// taskLoader periodically reloads tasks from database
func (s *Scheduler) taskLoader() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.loadTasks()
		}
	}
}

// getExecutor creates an executor based on type
func (s *Scheduler) getExecutor(execType, scriptContent, scriptPath string) Executor {
	switch execType {
	case "script":
		return &ScriptExecutor{
			content: scriptContent,
			path:    scriptPath,
		}
	case "python":
		return &PythonExecutor{
			content: scriptContent,
			path:    scriptPath,
		}
	case "api":
		return &APIExecutor{
			content: scriptContent,
		}
	default:
		return nil
	}
}

// calculateNextRun calculates the next run time for a cron expression
func calculateNextRun(cronExpr string) time.Time {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return time.Now().Add(1 * time.Hour)
	}
	return schedule.Next(time.Now())
}

// RunTaskNow runs a task immediately
func (s *Scheduler) RunTaskNow(taskID string) error {
	var task models.ScheduledTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return err
	}

	executor := s.getExecutor(task.ExecutorType, task.ScriptContent, task.ScriptPath)
	if executor == nil {
		return fmt.Errorf("unknown executor type: %s", task.ExecutorType)
	}

	// Run asynchronously
	go func() {
		runner := &taskRunner{
			taskID:    task.ID,
			taskName:  task.Name,
			db:        s.db,
			executor:  executor,
			parameters: task.Parameters,
		}
		runner.run()
	}()

	return nil
}

// taskRunner wraps a task for execution in cron
type taskRunner struct {
	taskID    string
	taskName  string
	db        *gorm.DB
	executor  Executor
	parameters string
}

// Run implements cron.Job interface
func (r *taskRunner) Run() {
	r.run()
}

func (r *taskRunner) run() {
	log.Printf("Executing task: %s", r.taskName)

	// Parse parameters
	var params map[string]interface{}
	if r.parameters != "" {
		// Parse JSON params
	}

	// Create execution record
	execution := models.TaskExecution{
		TaskID:    r.taskID,
		Status:    "running",
		StartedAt: time.Now(),
	}
	r.db.Create(&execution)

	// Execute
	ctx := context.Background()
	output, err := r.executor.Execute(ctx, params)

	// Update execution record
	now := time.Now()
	execution.CompletedAt = &now
	if err != nil {
		execution.Status = "failed"
		execution.ErrorMsg = err.Error()
	} else {
		execution.Status = "success"
		execution.Output = output
	}
	execution.DurationMs = int64(now.Sub(execution.StartedAt).Milliseconds())
	r.db.Save(&execution)

	// Update task last run time
	r.db.Model(&models.ScheduledTask{}).Where("id = ?", r.taskID).Updates(map[string]interface{}{
		"last_run_at": time.Now(),
	})

	log.Printf("Task %s completed: %s", r.taskName, execution.Status)
}

// ScriptExecutor runs shell scripts
type ScriptExecutor struct {
	content string
	path    string
}

func (e *ScriptExecutor) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	var cmd *exec.Cmd

	if e.path != "" {
		cmd = exec.Command("sh", e.path)
	} else {
		cmd = exec.Command("sh", "-c", e.content)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// PythonExecutor runs Python scripts
type PythonExecutor struct {
	content string
	path    string
}

func (e *PythonExecutor) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	var cmd *exec.Cmd

	if e.path != "" {
		cmd = exec.Command("python", e.path)
	} else {
		tmpFile, err := os.CreateTemp("", "wisehoof_*.py")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString(e.content)
		tmpFile.Close()

		cmd = exec.Command("python", tmpFile.Name())
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// APIExecutor calls HTTP APIs
type APIExecutor struct {
	content string
}

func (e *APIExecutor) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	// TODO: Implement HTTP API calling
	return "API executor not fully implemented", nil
}
