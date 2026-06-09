package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ============================================
// Environment Manager - uv 安装 + Python 环境管理
// ============================================

const (
	uvInstallScript = "https://astral.sh/uv/install.sh"  // Linux/macOS
	uvWindowsScript = "https://astral.sh/uv/install.ps1" // Windows
	envsDir         = "data/envs"                        // 虚拟环境目录
	uvPath          = "data/uv"                          // uv 安装路径
)

// EnvManager 环境管理器
type EnvManager struct {
	uvInstalled bool
	uvBin       string // uv 可执行文件路径
}

func NewEnvManager() *EnvManager {
	m := &EnvManager{}
	m.detectUV()
	return m
}

// detectUV 检测系统中是否已安装 uv
func (m *EnvManager) detectUV() bool {
	// 尝试查找 uv
	candidates := []string{"uv", "uv.exe", filepath.Join(uvPath, "uv"), filepath.Join(uvPath, "uv.exe")}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			m.uvInstalled = true
			m.uvBin = c
			log.Printf("[Env] 检测到 uv: %s", c)
			return true
		}
		// 也检查绝对路径
		if _, err := os.Stat(c); err == nil {
			m.uvInstalled = true
			m.uvBin = c
			log.Printf("[Env] 检测到 uv: %s", c)
			return true
		}
	}
	return false
}

// EnsureUV 确保 uv 已安装，如果没有则自动安装
func (m *EnvManager) EnsureUV(ctx context.Context) error {
	if m.uvInstalled {
		return nil
	}

	log.Printf("[Env] uv 未安装，开始自动安装...")

	switch runtime.GOOS {
	case "linux", "darwin":
		return m.installUVUnix(ctx)
	case "windows":
		return m.installUVWindows(ctx)
	default:
		return fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// installUVUnix Linux/macOS 安装 uv
func (m *EnvManager) installUVUnix(ctx context.Context) error {
	// 确保安装目录存在
	if err := os.MkdirAll(uvPath, 0755); err != nil {
		return fmt.Errorf("创建 uv 目录失败: %w", err)
	}

	// 下载安装脚本
	scriptPath := filepath.Join(uvPath, "install.sh")
	cmd := exec.CommandContext(ctx, "curl", "-LsSf", uvInstallScript, "-o", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// 尝试 wget 作为 curl 的替代
		cmd2 := exec.CommandContext(ctx, "wget", "-q", uvInstallScript, "-O", scriptPath)
		if err2 := cmd2.Run(); err2 != nil {
			return fmt.Errorf("下载 uv 安装脚本失败 (curl/wget 均不可用): %w", err)
		}
	}

	// 安装 uv 到本地目录
	installCmd := exec.CommandContext(ctx, "sh", scriptPath)
	installCmd.Env = append(os.Environ(), fmt.Sprintf("UV_INSTALL_DIR=%s", uvPath))
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	ctx2, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	installCmd = exec.CommandContext(ctx2, "sh", scriptPath)
	installCmd.Env = append(os.Environ(), fmt.Sprintf("UV_INSTALL_DIR=%s", uvPath))

	output, err := installCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("安装 uv 失败: %w\n输出: %s", err, string(output))
	}

	log.Printf("[Env] uv 安装成功: %s", uvPath)
	m.uvInstalled = true
	m.uvBin = filepath.Join(uvPath, "uv")
	return nil
}

// installUVWindows Windows 安装 uv
func (m *EnvManager) installUVWindows(ctx context.Context) error {
	if err := os.MkdirAll(uvPath, 0755); err != nil {
		return fmt.Errorf("创建 uv 目录失败: %w", err)
	}

	// Windows 使用 PowerShell 下载
	psCmd := fmt.Sprintf(
		`$env:UV_INSTALL_DIR="%s"; powershell -c "irm %s | iex"`,
		uvPath, uvWindowsScript,
	)
	cmd := exec.CommandContext(ctx, "powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Windows 安装 uv 失败: %w\n输出: %s", err, string(output))
	}

	log.Printf("[Env] uv 安装成功: %s", uvPath)
	m.uvInstalled = true
	m.uvBin = filepath.Join(uvPath, "uv.exe")
	return nil
}

// CreateEnv 创建 Python 虚拟环境
func (m *EnvManager) CreateEnv(ctx context.Context, envName, pythonVersion string) (string, error) {
	if err := m.EnsureUV(ctx); err != nil {
		return "", err
	}

	envDir := filepath.Join(envsDir, envName)
	if err := os.MkdirAll(filepath.Dir(envDir), 0755); err != nil {
		return "", err
	}

	args := []string{"venv", envDir}
	if pythonVersion != "" {
		args = append(args, "--python", pythonVersion)
	}

	cmd := exec.CommandContext(ctx, m.uvBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("创建虚拟环境失败: %w\n输出: %s", err, string(output))
	}

	log.Printf("[Env] 虚拟环境已创建: %s (Python %s)", envDir, pythonVersion)
	return envDir, nil
}

// InstallDeps 安装 Python 依赖包
func (m *EnvManager) InstallDeps(ctx context.Context, envName string, packages []string) error {
	if err := m.EnsureUV(ctx); err != nil {
		return err
	}

	envDir := filepath.Join(envsDir, envName)
	args := []string{"pip", "install", "--python", pythonInEnv(envDir)}
	args = append(args, packages...)

	cmd := exec.CommandContext(ctx, m.uvBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("安装依赖失败: %w\n输出: %s", err, string(output))
	}

	log.Printf("[Env] 依赖已安装: %v (环境: %s)", packages, envName)
	return nil
}

// InstallDepsFromFile 从 requirements.txt 安装依赖
func (m *EnvManager) InstallDepsFromFile(ctx context.Context, envName, reqFile string) error {
	if err := m.EnsureUV(ctx); err != nil {
		return err
	}

	envDir := filepath.Join(envsDir, envName)
	args := []string{"pip", "install", "-r", reqFile, "--python", pythonInEnv(envDir)}

	cmd := exec.CommandContext(ctx, m.uvBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("从 %s 安装依赖失败: %w\n输出: %s", err, reqFile, string(output))
	}

	log.Printf("[Env] 从 %s 安装依赖完成 (环境: %s)", reqFile, envName)
	return nil
}

// RunScript 在指定环境中执行 Python 脚本
func (m *EnvManager) RunScript(ctx context.Context, envName, scriptPath string) (string, error) {
	if err := m.EnsureUV(ctx); err != nil {
		return "", err
	}

	envDir := filepath.Join(envsDir, envName)
	args := []string{"run", "--python", pythonInEnv(envDir), scriptPath}

	cmd := exec.CommandContext(ctx, m.uvBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("脚本执行失败: %w", err)
	}

	return string(output), nil
}

// DetectScriptDeps 分析脚本内容，检测需要的依赖包
func (m *EnvManager) DetectScriptDeps(scriptContent string) []string {
	deps := make([]string, 0)
	seen := make(map[string]bool)

	// 常见 import 映射
	importMap := map[string]string{
		"pandas":         "pandas",
		"numpy":          "numpy",
		"requests":       "requests",
		"flask":          "flask",
		"fastapi":        "fastapi",
		"sqlalchemy":     "sqlalchemy",
		"pymysql":        "pymysql",
		"psycopg2":       "psycopg2-binary",
		"matplotlib":     "matplotlib",
		"seaborn":        "seaborn",
		"scipy":          "scipy",
		"scikit-learn":   "scikit-learn",
		"sklearn":        "scikit-learn",
		"openpyxl":       "openpyxl",
		"xlrd":           "xlrd",
		"pdfplumber":     "pdfplumber",
		"pydantic":       "pydantic",
		"jinja2":         "jinja2",
		"click":          "click",
		"rich":           "rich",
		"tqdm":           "tqdm",
		"pillow":         "Pillow",
		"PIL":            "Pillow",
		"reportlab":      "reportlab",
		"fpdf":           "fpdf2",
		"cryptography":   "cryptography",
		"jwt":            "PyJWT",
		"yaml":           "PyYAML",
		"toml":           "toml",
		"httpx":          "httpx",
		"aiohttp":        "aiohttp",
		"celery":         "celery",
		"redis":          "redis",
		"kafka":          "kafka-python",
		"selenium":       "selenium",
		"playwright":     "playwright",
		"beautifulsoup4": "beautifulsoup4",
		"bs4":            "beautifulsoup4",
		"lxml":           "lxml",
	}

	for imp, pkg := range importMap {
		if !seen[pkg] && strings.Contains(scriptContent, imp) {
			deps = append(deps, pkg)
			seen[pkg] = true
		}
	}

	return deps
}

// SetupScriptEnv 为脚本自动设置环境
// 返回环境名和可能的错误
func (m *EnvManager) SetupScriptEnv(ctx context.Context, scriptName, scriptContent string, extraDeps []string) (string, error) {
	envName := sanitizeEnvName(scriptName)

	// 检测依赖
	deps := m.DetectScriptDeps(scriptContent)
	deps = append(deps, extraDeps...)

	if len(deps) == 0 {
		// 无外部依赖，使用通用环境
		return "default", nil
	}

	// 创建/复用环境
	envDir := filepath.Join(envsDir, envName)
	if _, err := os.Stat(envDir); os.IsNotExist(err) {
		if _, err := m.CreateEnv(ctx, envName, ""); err != nil {
			return "", err
		}
	}

	// 安装依赖
	if err := m.InstallDeps(ctx, envName, deps); err != nil {
		return "", err
	}

	return envName, nil
}

// IsUVInstalled 检查 uv 是否已安装
func (m *EnvManager) IsUVInstalled() bool {
	return m.uvInstalled
}

// GetUVPython 获取环境中的 Python 路径
func (m *EnvManager) GetUVPython(envName string) string {
	envDir := filepath.Join(envsDir, envName)
	return pythonInEnv(envDir)
}

// CleanupEnvs 清理未使用的虚拟环境
func (m *EnvManager) CleanupEnvs(maxAge time.Duration) error {
	entries, err := os.ReadDir(envsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		envPath := filepath.Join(envsDir, entry.Name())
		info, err := os.Stat(envPath)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			log.Printf("[Env] 清理过期环境: %s", envPath)
			os.RemoveAll(envPath)
		}
	}
	return nil
}

func pythonInEnv(envDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(envDir, "Scripts", "python.exe")
	}
	return filepath.Join(envDir, "bin", "python")
}

func sanitizeEnvName(name string) string {
	name = strings.ToLower(name)
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)
	if len(name) > 50 {
		name = name[:50]
	}
	return strings.Trim(name, "_-")
}
