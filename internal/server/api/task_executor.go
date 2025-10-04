package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/plumber/plumber/internal/server/storage"
	"github.com/plumber/plumber/pkg/jsonrpc"
	"github.com/plumber/plumber/pkg/models"
)

// TaskExecutor 任务执行器
type TaskExecutor struct {
	storage storage.Storage
}

// NewTaskExecutor 创建任务执行器
func NewTaskExecutor(storage storage.Storage) *TaskExecutor {
	return &TaskExecutor{storage: storage}
}

// ExecuteTask 执行任务
func (e *TaskExecutor) ExecuteTask(ctx context.Context, taskID uuid.UUID) error {
	// 获取任务
	task, err := e.storage.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 解析TOML配置
	var config models.TaskConfig
	if err := toml.Unmarshal([]byte(task.Config), &config); err != nil {
		return fmt.Errorf("failed to parse task config: %w", err)
	}

	// 创建执行记录
	execution := &models.TaskExecution{
		TaskID: taskID,
		Status: "running",
	}
	now := time.Now()
	execution.StartTime = &now

	if err := e.storage.CreateExecution(ctx, execution); err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	// 更新任务状态
	task.Status = "running"
	if err := e.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Failed to update task status: %v", err)
	}

	// 执行步骤
	success := true
	for i, step := range config.Steps {
		agentID, err := uuid.Parse(step.ServerID)
		if err != nil {
			log.Printf("Invalid agent ID in step %d: %v", i, err)
			success = false
			break
		}

		// 检查Agent是否在线
		agent, err := e.storage.GetAgent(ctx, agentID)
		if err != nil {
			log.Printf("Agent %s not found: %v", agentID, err)
			success = false
			break
		}

		if agent.Status != "online" {
			log.Printf("Agent %s is offline", agentID)
			success = false
			break
		}

		// 创建步骤执行记录
		stepExec := &models.StepExecution{
			ExecutionID: execution.ID,
			StepIndex:   i,
			AgentID:     agentID,
			Path:        step.Path,
			Command:     step.CMD,
			Status:      "running",
		}
		stepStartTime := time.Now()
		stepExec.StartTime = &stepStartTime

		if err := e.storage.CreateStepExecution(ctx, stepExec); err != nil {
			log.Printf("Failed to create step execution: %v", err)
			success = false
			break
		}

		// 发送执行命令到Agent
		if err := e.sendCommandToAgent(ctx, agent.IP, stepExec.ID, step.Path, step.CMD); err != nil {
			log.Printf("Failed to send command to agent: %v", err)
			stepExec.Status = "failed"
			e.storage.UpdateStepExecution(ctx, stepExec)
			success = false
			break
		}

		// 等待步骤完成(轮询检查状态)
		if err := e.waitForStepCompletion(ctx, stepExec.ID); err != nil {
			log.Printf("Step execution failed: %v", err)
			success = false
			break
		}

		// 检查退出码
		updatedStep, err := e.storage.GetStepExecution(ctx, stepExec.ID)
		if err != nil {
			log.Printf("Failed to get step execution: %v", err)
			success = false
			break
		}

		if updatedStep.ExitCode != nil && *updatedStep.ExitCode != 0 {
			log.Printf("Step %d failed with exit code %d", i, *updatedStep.ExitCode)
			success = false
			break
		}
	}

	// 更新执行状态
	endTime := time.Now()
	execution.EndTime = &endTime
	if success {
		execution.Status = "success"
		task.Status = "success"
	} else {
		execution.Status = "failed"
		task.Status = "failed"
	}

	if err := e.storage.UpdateExecution(ctx, execution); err != nil {
		log.Printf("Failed to update execution: %v", err)
	}

	if err := e.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Failed to update task: %v", err)
	}

	return nil
}

// sendCommandToAgent 发送命令到Agent
func (e *TaskExecutor) sendCommandToAgent(ctx context.Context, agentIP string, stepID uuid.UUID, path, cmd string) error {
	req := jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "plumber.agent.execute",
		Params: json.RawMessage(fmt.Sprintf(`{
			"step_id": "%s",
			"path": "%s",
			"command": "%s"
		}`, stepID, path, cmd)),
		ID: stepID.String(),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	// 发送HTTP请求到Agent
	resp, err := http.Post(
		fmt.Sprintf("http://%s:52182/rpc", agentIP),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// waitForStepCompletion 等待步骤完成
func (e *TaskExecutor) waitForStepCompletion(ctx context.Context, stepID uuid.UUID) error {
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("step execution timeout")
		case <-ticker.C:
			step, err := e.storage.GetStepExecution(ctx, stepID)
			if err != nil {
				return err
			}

			if step.Status == "success" || step.Status == "failed" {
				return nil
			}
		}
	}
}

// RunTaskMethod 运行任务方法
type RunTaskMethod struct {
	storage  storage.Storage
	executor *TaskExecutor
}

func NewRunTaskMethod(storage storage.Storage, executor *TaskExecutor) *RunTaskMethod {
	return &RunTaskMethod{
		storage:  storage,
		executor: executor,
	}
}

func (m *RunTaskMethod) Name() string {
	return "plumber.task.run"
}

func (m *RunTaskMethod) RequireAuth() bool {
	return true
}

type RunTaskParams struct {
	TaskID string `json:"task_id"`
}

func (m *RunTaskMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p RunTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	taskUUID, err := uuid.Parse(p.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task_id: %w", err)
	}

	// 异步执行任务
	go func() {
		if err := m.executor.ExecuteTask(context.Background(), taskUUID); err != nil {
			log.Printf("Task execution error: %v", err)
		}
	}()

	return map[string]interface{}{
		"status":  "started",
		"message": "Task execution started",
	}, nil
}

// GetExecutionMethod 获取执行记录
type GetExecutionMethod struct {
	storage storage.Storage
}

func NewGetExecutionMethod(storage storage.Storage) *GetExecutionMethod {
	return &GetExecutionMethod{storage: storage}
}

func (m *GetExecutionMethod) Name() string {
	return "plumber.execution.get"
}

func (m *GetExecutionMethod) RequireAuth() bool {
	return true
}

type GetExecutionParams struct {
	ExecutionID string `json:"execution_id"`
}

func (m *GetExecutionMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p GetExecutionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	executionUUID, err := uuid.Parse(p.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution_id: %w", err)
	}

	execution, err := m.storage.GetExecution(ctx, executionUUID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	return map[string]interface{}{
		"execution": execution,
	}, nil
}
