package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/plumber/plumber/internal/agent/executor"
	"github.com/plumber/plumber/pkg/jsonrpc"
)

// Client Agent客户端
type Client struct {
	serverURL  string
	agentID    uuid.UUID
	agentToken string
	httpClient *http.Client
}

// NewClient 创建新的Agent客户端
func NewClient(serverURL string, agentID uuid.UUID, agentToken string) *Client {
	return &Client{
		serverURL:  serverURL,
		agentID:    agentID,
		agentToken: agentToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Register 注册Agent
func (c *Client) Register(hostname, ip string) error {
	params := map[string]string{
		"agent_id": c.agentID.String(),
		"hostname": hostname,
		"ip":       ip,
	}

	_, err := c.callRPC("plumber.agent.register", params)
	return err
}

// Heartbeat 发送心跳
func (c *Client) Heartbeat() error {
	params := map[string]string{
		"agent_id": c.agentID.String(),
	}

	_, err := c.callRPC("plumber.agent.heartbeat", params)
	return err
}

// ReportStepResult 上报步骤执行结果
func (c *Client) ReportStepResult(stepID uuid.UUID, status string, exitCode int, output string) error {
	params := map[string]interface{}{
		"step_id":   stepID.String(),
		"status":    status,
		"exit_code": exitCode,
		"output":    output,
	}

	_, err := c.callRPC("plumber.step.report", params)
	return err
}

// callRPC 调用JSON-RPC方法
func (c *Client) callRPC(method string, params interface{}) (json.RawMessage, error) {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsBytes,
		ID:      uuid.New().String(),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.serverURL+"/api/rpc", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Agent-Token", c.agentToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	result, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// StartHeartbeat 启动心跳
func (c *Client) StartHeartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.Heartbeat(); err != nil {
				fmt.Printf("Heartbeat failed: %v\n", err)
			}
		}
	}
}

// PollTask 拉取待执行任务
func (c *Client) PollTask() (bool, *TaskInfo, error) {
	params := map[string]string{
		"agent_id": c.agentID.String(),
	}

	result, err := c.callRPC("plumber.agent.pollTask", params)
	if err != nil {
		return false, nil, err
	}

	var response struct {
		HasTask bool `json:"has_task"`
		Task    *struct {
			StepID  string `json:"step_id"`
			Path    string `json:"path"`
			Command string `json:"command"`
		} `json:"task,omitempty"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		return false, nil, err
	}

	if !response.HasTask || response.Task == nil {
		return false, nil, nil
	}

	taskInfo := &TaskInfo{
		StepID:  response.Task.StepID,
		Path:    response.Task.Path,
		Command: response.Task.Command,
	}

	return true, taskInfo, nil
}

// TaskInfo 任务信息
type TaskInfo struct {
	StepID  string
	Path    string
	Command string
}

// StartTaskPolling 启动任务轮询
func (c *Client) StartTaskPolling(ctx context.Context, exec *executor.Executor, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hasTask, taskInfo, err := c.PollTask()
			if err != nil {
				fmt.Printf("Failed to poll task: %v\n", err)
				continue
			}

			if !hasTask {
				continue
			}

			// 有任务，执行它
			log.Printf("[Task] Received task - StepID: %s, Path: %s, Command: %s",
				taskInfo.StepID, taskInfo.Path, taskInfo.Command)

			go func(info *TaskInfo) {
				startTime := time.Now()
				log.Printf("[Task] Starting execution - StepID: %s, Time: %s",
					info.StepID, startTime.Format("2006-01-02 15:04:05"))

				result := exec.ExecuteWithTimeout(info.Path, info.Command, 10*time.Minute)

				endTime := time.Now()
				duration := endTime.Sub(startTime)

				status := "success"
				if result.ExitCode != 0 {
					status = "failed"
				}

				log.Printf("[Task] Finished execution - StepID: %s, Status: %s, ExitCode: %d, Duration: %s",
					info.StepID, status, result.ExitCode, duration)

				stepID, _ := uuid.Parse(info.StepID)
				if err := c.ReportStepResult(stepID, status, result.ExitCode, result.Output); err != nil {
					log.Printf("[Task] Failed to report step result: %v", err)
				}
			}(taskInfo)
		}
	}
}
