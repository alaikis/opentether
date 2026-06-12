package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config sandbox configuration
type Config struct {
	Enabled     bool          // enable sandbox
	Image       string        // Docker image, default "ubuntu:22.04"
	MemoryLimit string        // memory limit, e.g., "256m"
	CPULimit    string        // CPU limit, e.g., "0.5"
	Timeout     time.Duration // execution timeout
	WorkDir     string        // host working directory to mount
}

// Executor executes commands in a Docker sandbox
type Executor struct {
	config Config
}

// New creates a new sandbox executor
func New(cfg Config) *Executor {
	if cfg.Image == "" {
		cfg.Image = "ubuntu:22.04"
	}
	if cfg.MemoryLimit == "" {
		cfg.MemoryLimit = "256m"
	}
	if cfg.CPULimit == "" {
		cfg.CPULimit = "0.5"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "/tmp/sandbox"
	}
	return &Executor{config: cfg}
}

// IsAvailable checks if Docker is available on the host
func (e *Executor) IsAvailable() error {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Docker 未安装或不在 PATH 中，无法使用沙箱模式: %w", err)
	}

	// Verify docker is functional
	cmd := exec.Command(dockerPath, "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Docker 守护进程不可用: %w", err)
	}

	return nil
}

// Result represents the result of sandbox execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Execute runs a command in a Docker sandbox
func (e *Executor) Execute(ctx context.Context, command string, workDir string, env map[string]string) (*Result, error) {
	if !e.config.Enabled {
		return nil, fmt.Errorf("沙箱模式未启用")
	}

	if err := e.IsAvailable(); err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Resolve host workDir to absolute path
	hostWorkDir := e.config.WorkDir
	if workDir != "" {
		hostWorkDir = workDir
	}
	absWorkDir, err := filepath.Abs(hostWorkDir)
	if err != nil {
		return nil, fmt.Errorf("解析工作目录失败: %w", err)
	}

	// Ensure workDir exists on host
	if err := os.MkdirAll(absWorkDir, 0755); err != nil {
		return nil, fmt.Errorf("创建工作目录失败: %w", err)
	}

	// Build docker run arguments
	args := []string{
		"run",
		"--rm",
		"--network", "none",
		"--memory", e.config.MemoryLimit,
		"--cpus", e.config.CPULimit,
		"--read-only",
		"--tmpfs", "/tmp:exec,size=64m",
		"-v", absWorkDir + ":/workspace:rw",
		"-w", "/workspace",
		"--user", "1000:1000",
	}

	// Add timeout
	if e.config.Timeout > 0 {
		args = append(args, "--stop-timeout", "10")
	}

	// Add environment variables
	for k, v := range env {
		args = append(args, "-e", k+"="+v)
	}

	// Add image
	args = append(args, e.config.Image)

	// Add command (via sh -c)
	args = append(args, "sh", "-c", command)

	// Apply context timeout if set
	execCtx := ctx
	var cancel context.CancelFunc
	if e.config.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	dockerPath, _ := exec.LookPath("docker")
	cmd := exec.CommandContext(execCtx, dockerPath, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	duration := time.Since(startTime)

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if execCtx.Err() == context.DeadlineExceeded {
			result.ExitCode = 124
			result.Stderr += "\n[沙箱] 命令执行超时"
		} else {
			result.ExitCode = -1
			result.Stderr += "\n[沙箱] 执行错误: " + err.Error()
		}
	}

	return result, nil
}

// ExecuteScript runs a script file in the sandbox
func (e *Executor) ExecuteScript(ctx context.Context, scriptPath string, language string, workDir string, env map[string]string) (*Result, error) {
	if !e.config.Enabled {
		return nil, fmt.Errorf("沙箱模式未启用")
	}

	// Resolve workDir
	hostWorkDir := e.config.WorkDir
	if workDir != "" {
		hostWorkDir = workDir
	}
	absWorkDir, err := filepath.Abs(hostWorkDir)
	if err != nil {
		return nil, fmt.Errorf("解析工作目录失败: %w", err)
	}

	var resolvedScriptPath string
	if filepath.IsAbs(scriptPath) {
		if _, err := os.Stat(scriptPath); err != nil {
			return nil, fmt.Errorf("脚本文件不存在: %s", scriptPath)
		}

		scriptName := filepath.Base(scriptPath)
		destPath := filepath.Join(absWorkDir, scriptName)
		if scriptPath != destPath {
			src, err := os.Open(scriptPath)
			if err != nil {
				return nil, fmt.Errorf("读取脚本文件失败: %w", err)
			}
			defer src.Close()

			dst, err := os.Create(destPath)
			if err != nil {
				return nil, fmt.Errorf("创建脚本文件失败: %w", err)
			}
			defer dst.Close()

			if _, err := io.Copy(dst, src); err != nil {
				return nil, fmt.Errorf("复制脚本文件失败: %w", err)
			}

			if err := os.Chmod(destPath, 0755); err != nil {
				return nil, fmt.Errorf("设置脚本权限失败: %w", err)
			}
		} else {
			if err := os.Chmod(scriptPath, 0755); err != nil {
				return nil, fmt.Errorf("设置脚本权限失败: %w", err)
			}
		}
		resolvedScriptPath = "/workspace/" + scriptName
	} else {
		resolvedScriptPath = "/workspace/" + filepath.Base(scriptPath)
		hostPath := filepath.Join(absWorkDir, filepath.Base(scriptPath))
		if _, err := os.Stat(hostPath); err == nil {
			if err := os.Chmod(hostPath, 0755); err != nil {
				return nil, fmt.Errorf("设置脚本权限失败: %w", err)
			}
		}
	}

	var cmd string
	switch strings.ToLower(language) {
	case "python", "py", "python3":
		cmd = fmt.Sprintf("python3 %s", resolvedScriptPath)
	case "bash", "sh", "shell":
		cmd = fmt.Sprintf("bash %s", resolvedScriptPath)
	default:
		cmd = fmt.Sprintf("bash %s", resolvedScriptPath)
	}

	return e.Execute(ctx, cmd, absWorkDir, env)
}

// SetupEnvironment installs packages in the sandbox
func (e *Executor) SetupEnvironment(ctx context.Context, packages []string, workDir string) (*Result, error) {
	if !e.config.Enabled {
		return nil, fmt.Errorf("沙箱模式未启用")
	}

	if len(packages) == 0 {
		return &Result{
			Stdout:   "[沙箱] 无需安装的包",
			ExitCode: 0,
		}, nil
	}

	var aptPackages []string
	var pipPackages []string
	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		if strings.HasPrefix(pkg, "python-") || strings.HasPrefix(pkg, "python3-") {
			aptPackages = append(aptPackages, pkg)
		} else if strings.Contains(pkg, ".") && !strings.Contains(pkg, "/") {
			pipPackages = append(pipPackages, pkg)
		} else {
			aptPackages = append(aptPackages, pkg)
		}
	}

	var allResults []string

	if len(aptPackages) > 0 {
		aptCmd := fmt.Sprintf("apt-get update -qq && apt-get install -y -qq %s", strings.Join(aptPackages, " "))
		result, err := e.Execute(ctx, aptCmd, workDir, nil)
		if err != nil {
			return nil, fmt.Errorf("apt 包安装失败: %w", err)
		}
		if result.ExitCode != 0 {
			return result, nil
		}
		allResults = append(allResults, fmt.Sprintf("[apt] 已安装: %s", strings.Join(aptPackages, ", ")))
	}

	if len(pipPackages) > 0 {
		pipCmd := fmt.Sprintf("pip install -q %s", strings.Join(pipPackages, " "))
		result, err := e.Execute(ctx, pipCmd, workDir, nil)
		if err != nil {
			return nil, fmt.Errorf("pip 包安装失败: %w", err)
		}
		if result.ExitCode != 0 {
			return result, nil
		}
		allResults = append(allResults, fmt.Sprintf("[pip] 已安装: %s", strings.Join(pipPackages, ", ")))
	}

	return &Result{
		Stdout:   strings.Join(allResults, "\n"),
		ExitCode: 0,
	}, nil
}
