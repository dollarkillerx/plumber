package executor

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// Executor 命令执行器
type Executor struct {
	workDir string
}

// NewExecutor 创建新的执行器
func NewExecutor(workDir string) *Executor {
	return &Executor{
		workDir: workDir,
	}
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	ExitCode int
	Output   string
	Error    error
}

// Execute 执行命令
func (e *Executor) Execute(ctx context.Context, path, command string) *ExecuteResult {
	result := &ExecuteResult{}

	// 设置工作目录
	workDir := path
	if workDir == "" {
		workDir = e.workDir
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workDir

	// 捕获输出
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 执行命令
	err := cmd.Run()

	// 合并输出
	result.Output = stdout.String()
	if stderr.Len() > 0 {
		result.Output += "\n" + stderr.String()
	}

	// 获取退出码
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = err
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// ExecuteWithTimeout 带超时的命令执行
func (e *Executor) ExecuteWithTimeout(path, command string, timeout time.Duration) *ExecuteResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return e.Execute(ctx, path, command)
}
