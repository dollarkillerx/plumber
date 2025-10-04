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
	startTime := time.Now()
	log.Printf("[Server] Starting task execution - TaskID: %s, Time: %s", taskID, startTime.Format("2006-01-02 15:04:05"))

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

	log.Printf("[Server] Task config parsed - TaskID: %s, Steps: %d", taskID, len(config.Steps))

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

	log.Printf("[Server] Execution created - ExecutionID: %s", execution.ID)

	// 更新任务状态
	task.Status = "running"
	if err := e.storage.UpdateTask(ctx, task); err != nil {
		log.Printf("Failed to update task status: %v", err)
	}

	// 执行步骤
	success := true
	for i, step := range config.Steps {
		log.Printf("[Server] Processing step %d/%d - Command: %s", i+1, len(config.Steps), step.CMD)

		agentID, err := uuid.Parse(step.ServerID)
		if err != nil {
			log.Printf("[Server] Invalid agent ID in step %d: %v", i, err)
			success = false
			break
		}

		// 检查Agent是否在线
		agent, err := e.storage.GetAgent(ctx, agentID)
		if err != nil {
			log.Printf("[Server] Agent %s not found: %v", agentID, err)
			success = false
			break
		}

		if agent.Status != "online" {
			log.Printf("[Server] Agent %s is offline", agentID)
			success = false
			break
		}

		log.Printf("[Server] Creating step for agent - AgentID: %s", agentID)

		// 创建步骤执行记录（状态为pending，等待agent拉取）
		stepExec := &models.StepExecution{
			ExecutionID: execution.ID,
			StepIndex:   i,
			AgentID:     agentID,
			Path:        step.Path,
			Command:     step.CMD,
			Status:      "pending",
			Assigned:    false,
		}

		if err := e.storage.CreateStepExecution(ctx, stepExec); err != nil {
			log.Printf("[Server] Failed to create step execution: %v", err)
			success = false
			break
		}

		log.Printf("[Server] Step created, waiting for agent to pull - StepID: %s", stepExec.ID)
		log.Printf("[Server] Waiting for step completion - StepID: %s", stepExec.ID)

		// 等待步骤完成(轮询检查状态)
		if err := e.waitForStepCompletion(ctx, stepExec.ID); err != nil {
			log.Printf("[Server] Step execution failed: %v", err)
			success = false
			break
		}

		// 检查退出码
		updatedStep, err := e.storage.GetStepExecution(ctx, stepExec.ID)
		if err != nil {
			log.Printf("[Server] Failed to get step execution: %v", err)
			success = false
			break
		}

		if updatedStep.ExitCode != nil && *updatedStep.ExitCode != 0 {
			log.Printf("[Server] Step %d failed with exit code %d", i, *updatedStep.ExitCode)
			success = false
			break
		}

		log.Printf("[Server] Step %d completed successfully", i+1)
	}

	// 更新执行状态
	endTime := time.Now()
	execution.EndTime = &endTime
	duration := endTime.Sub(startTime)

	if success {
		execution.Status = "success"
		task.Status = "success"
	} else {
		execution.Status = "failed"
		task.Status = "failed"
	}

	log.Printf("[Server] Task execution finished - TaskID: %s, ExecutionID: %s, Status: %s, StartTime: %s, EndTime: %s, Duration: %s",
		taskID, execution.ID, execution.Status, startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"), duration)

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
	// 构造参数
	params := map[string]string{
		"step_id": stepID.String(),
		"path":    path,
		"command": cmd,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	req := jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "plumber.agent.execute",
		Params:  json.RawMessage(paramsJSON),
		ID:      stepID.String(),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s:52182/rpc", agentIP)
	log.Printf("[Server] Sending HTTP request to agent - URL: %s, Body: %s", url, string(body))

	// 发送HTTP请求到Agent
	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Printf("[Server] Failed to send request to agent: %v", err)
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	var respBody bytes.Buffer
	if _, err := respBody.ReadFrom(resp.Body); err == nil {
		log.Printf("[Server] Agent response - Status: %d, Body: %s", resp.StatusCode, respBody.String())
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("agent returned non-OK status: %d", resp.StatusCode)
	}

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

// ListExecutionsMethod 获取任务的执行历史
type ListExecutionsMethod struct {
	storage storage.Storage
}

func NewListExecutionsMethod(storage storage.Storage) *ListExecutionsMethod {
	return &ListExecutionsMethod{storage: storage}
}

func (m *ListExecutionsMethod) Name() string {
	return "plumber.execution.list"
}

func (m *ListExecutionsMethod) RequireAuth() bool {
	return true
}

type ListExecutionsParams struct {
	TaskID string `json:"task_id"`
}

func (m *ListExecutionsMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p ListExecutionsParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	taskUUID, err := uuid.Parse(p.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task_id: %w", err)
	}

	executions, err := m.storage.ListExecutionsByTaskID(ctx, taskUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return map[string]interface{}{
		"executions": executions,
	}, nil
}
