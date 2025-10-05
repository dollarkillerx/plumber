package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/plumber/plumber/internal/server/storage"
	"github.com/plumber/plumber/pkg/auth"
	"github.com/plumber/plumber/pkg/jsonrpc"
	"github.com/plumber/plumber/pkg/models"
)

// AgentRegisterMethod Agent注册方法
type AgentRegisterMethod struct {
	storage storage.Storage
}

func NewAgentRegisterMethod(storage storage.Storage) *AgentRegisterMethod {
	return &AgentRegisterMethod{storage: storage}
}

func (m *AgentRegisterMethod) Name() string {
	return "plumber.agent.register"
}

func (m *AgentRegisterMethod) RequireAuth() bool {
	return false
}

type AgentRegisterParams struct {
	AgentID  string `json:"agent_id"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

func (m *AgentRegisterMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p AgentRegisterParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	// 尝试获取现有agent
	existing, err := m.storage.GetAgent(ctx, agentUUID)
	if err == nil {
		// Agent已存在,更新状态和实际信息
		existing.Hostname = p.Hostname
		existing.IP = p.IP
		existing.Status = "online"
		now := time.Now()
		existing.LastHeartbeat = &now

		if err := m.storage.UpdateAgent(ctx, existing); err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"status":  "updated",
			"message": "Agent reconnected",
		}, nil
	}

	// Agent不存在，返回错误（需要先在后台创建）
	return nil, fmt.Errorf("agent not found, please create agent in admin panel first")
}

// AgentHeartbeatMethod Agent心跳方法
type AgentHeartbeatMethod struct {
	storage storage.Storage
}

func NewAgentHeartbeatMethod(storage storage.Storage) *AgentHeartbeatMethod {
	return &AgentHeartbeatMethod{storage: storage}
}

func (m *AgentHeartbeatMethod) Name() string {
	return "plumber.agent.heartbeat"
}

func (m *AgentHeartbeatMethod) RequireAuth() bool {
	return false
}

type AgentHeartbeatParams struct {
	AgentID string `json:"agent_id"`
}

func (m *AgentHeartbeatMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p AgentHeartbeatParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	if err := m.storage.UpdateAgentHeartbeat(ctx, agentUUID); err != nil {
		return nil, fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return map[string]interface{}{
		"status": "ok",
	}, nil
}

// UserLoginMethod 用户登录方法
type UserLoginMethod struct {
	jwtManager    *auth.JWTManager
	adminUsername string
	adminPassword string
}

func NewUserLoginMethod(jwtManager *auth.JWTManager, adminUsername, adminPassword string) *UserLoginMethod {
	return &UserLoginMethod{
		jwtManager:    jwtManager,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}
}

func (m *UserLoginMethod) Name() string {
	return "plumber.user.login"
}

func (m *UserLoginMethod) RequireAuth() bool {
	return false
}

type UserLoginParams struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (m *UserLoginMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p UserLoginParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// 直接从配置验证用户名和密码
	if p.Username != m.adminUsername || p.Password != m.adminPassword {
		return nil, fmt.Errorf("invalid username or password")
	}

	// 生成用户ID (使用固定UUID或用户名hash)
	userID := "admin"

	token, err := m.jwtManager.Generate(userID, p.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return map[string]interface{}{
		"token":    token,
		"username": p.Username,
		"user_id":  userID,
	}, nil
}

// ListAgentsMethod 列出所有Agent
type ListAgentsMethod struct {
	storage storage.Storage
}

func NewListAgentsMethod(storage storage.Storage) *ListAgentsMethod {
	return &ListAgentsMethod{storage: storage}
}

func (m *ListAgentsMethod) Name() string {
	return "plumber.agent.list"
}

func (m *ListAgentsMethod) RequireAuth() bool {
	return true
}

func (m *ListAgentsMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	agents, err := m.storage.ListAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return map[string]interface{}{
		"agents": agents,
	}, nil
}

// CreateTaskMethod 创建任务
type CreateTaskMethod struct {
	storage storage.Storage
}

func NewCreateTaskMethod(storage storage.Storage) *CreateTaskMethod {
	return &CreateTaskMethod{storage: storage}
}

func (m *CreateTaskMethod) Name() string {
	return "plumber.task.create"
}

func (m *CreateTaskMethod) RequireAuth() bool {
	return true
}

type CreateTaskParams struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"` // TOML配置
}

func (m *CreateTaskMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p CreateTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	task := &models.Task{
		Name:        p.Name,
		Description: p.Description,
		Config:      p.Config,
		Status:      "pending",
	}

	if err := m.storage.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return map[string]interface{}{
		"task_id": task.ID.String(),
		"status":  "created",
	}, nil
}

// UpdateTaskMethod 更新任务
type UpdateTaskMethod struct {
	storage storage.Storage
}

func NewUpdateTaskMethod(storage storage.Storage) *UpdateTaskMethod {
	return &UpdateTaskMethod{storage: storage}
}

func (m *UpdateTaskMethod) Name() string {
	return "plumber.task.update"
}

func (m *UpdateTaskMethod) RequireAuth() bool {
	return true
}

type UpdateTaskParams struct {
	TaskID      string `json:"task_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Config      string `json:"config"`
}

func (m *UpdateTaskMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p UpdateTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	taskUUID, err := uuid.Parse(p.TaskID)
	if err != nil {
		return nil, fmt.Errorf("invalid task_id: %w", err)
	}

	task, err := m.storage.GetTask(ctx, taskUUID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// 更新字段
	if p.Name != "" {
		task.Name = p.Name
	}
	task.Description = p.Description
	task.Config = p.Config

	if err := m.storage.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return map[string]interface{}{
		"status":  "updated",
		"message": "Task updated successfully",
	}, nil
}

// ListTasksMethod 列出所有任务
type ListTasksMethod struct {
	storage storage.Storage
}

func NewListTasksMethod(storage storage.Storage) *ListTasksMethod {
	return &ListTasksMethod{storage: storage}
}

func (m *ListTasksMethod) Name() string {
	return "plumber.task.list"
}

func (m *ListTasksMethod) RequireAuth() bool {
	return true
}

func (m *ListTasksMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	tasks, err := m.storage.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return map[string]interface{}{
		"tasks": tasks,
	}, nil
}

// PollTaskMethod Agent拉取待执行任务
type PollTaskMethod struct {
	storage storage.Storage
}

func NewPollTaskMethod(storage storage.Storage) *PollTaskMethod {
	return &PollTaskMethod{storage: storage}
}

func (m *PollTaskMethod) Name() string {
	return "plumber.agent.pollTask"
}

func (m *PollTaskMethod) RequireAuth() bool {
	return false
}

type PollTaskParams struct {
	AgentID string `json:"agent_id"`
}

func (m *PollTaskMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p PollTaskParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	// 获取待执行的步骤（限制1个，避免一次拉取太多）
	// GetPendingStepsForAgent 内部已经使用事务+行锁来防止重复分配
	steps, err := m.storage.GetPendingStepsForAgent(ctx, agentUUID, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending steps: %w", err)
	}

	if len(steps) == 0 {
		return map[string]interface{}{
			"has_task": false,
		}, nil
	}

	step := steps[0]

	// 更新步骤状态为running
	step.Status = "running"
	now := time.Now()
	step.StartTime = &now
	if err := m.storage.UpdateStepExecution(ctx, step); err != nil {
		log.Printf("[Server] Failed to update step status: %v", err)
	}

	log.Printf("[Server] Assigned task to agent - AgentID: %s, StepID: %s, Command: %s",
		agentUUID, step.ID, step.Command)

	return map[string]interface{}{
		"has_task": true,
		"task": map[string]interface{}{
			"step_id": step.ID.String(),
			"path":    step.Path,
			"command": step.Command,
		},
	}, nil
}

// StepReportMethod Agent上报步骤执行结果
type StepReportMethod struct {
	storage storage.Storage
}

func NewStepReportMethod(storage storage.Storage) *StepReportMethod {
	return &StepReportMethod{storage: storage}
}

func (m *StepReportMethod) Name() string {
	return "plumber.step.report"
}

func (m *StepReportMethod) RequireAuth() bool {
	return false
}

type StepReportParams struct {
	StepID   string `json:"step_id"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

func (m *StepReportMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p StepReportParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	stepUUID, err := uuid.Parse(p.StepID)
	if err != nil {
		return nil, fmt.Errorf("invalid step_id: %w", err)
	}

	step, err := m.storage.GetStepExecution(ctx, stepUUID)
	if err != nil {
		return nil, fmt.Errorf("step not found: %w", err)
	}

	now := time.Now()
	step.Status = p.Status
	step.ExitCode = &p.ExitCode
	step.Output = p.Output
	step.EndTime = &now

	if err := m.storage.UpdateStepExecution(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to update step: %w", err)
	}

	return map[string]interface{}{
		"status": "updated",
	}, nil
}

// CreateAgentMethod 创建Agent
type CreateAgentMethod struct {
	storage storage.Storage
}

func NewCreateAgentMethod(storage storage.Storage) *CreateAgentMethod {
	return &CreateAgentMethod{
		storage: storage,
	}
}

func (m *CreateAgentMethod) Name() string {
	return "plumber.agent.create"
}

func (m *CreateAgentMethod) RequireAuth() bool {
	return true
}

type CreateAgentParams struct {
	Name          string `json:"name"`
	SSHHost       string `json:"ssh_host,omitempty"`
	SSHPort       int    `json:"ssh_port,omitempty"`
	SSHUser       string `json:"ssh_user,omitempty"`
	SSHAuthType   string `json:"ssh_auth_type,omitempty"`   // password/key/none
	SSHPassword   string `json:"ssh_password,omitempty"`    // 密码认证
	SSHPrivateKey string `json:"ssh_private_key,omitempty"` // 密钥认证
}

func (m *CreateAgentMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p CreateAgentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if p.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	// 设置默认值
	if p.SSHPort == 0 {
		p.SSHPort = 22
	}
	if p.SSHAuthType == "" {
		p.SSHAuthType = "none"
	}

	agent := &models.Agent{
		ID:            uuid.New(),
		Name:          p.Name,
		SSHHost:       p.SSHHost,
		SSHPort:       p.SSHPort,
		SSHUser:       p.SSHUser,
		SSHAuthType:   p.SSHAuthType,
		SSHPassword:   p.SSHPassword,
		SSHPrivateKey: p.SSHPrivateKey,
		Status:        "offline",
	}

	if err := m.storage.CreateAgent(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return map[string]interface{}{
		"agent_id": agent.ID.String(),
		"name":     agent.Name,
		"status":   "created",
	}, nil
}

// UpdateAgentMethod 更新Agent
type UpdateAgentMethod struct {
	storage storage.Storage
}

func NewUpdateAgentMethod(storage storage.Storage) *UpdateAgentMethod {
	return &UpdateAgentMethod{storage: storage}
}

func (m *UpdateAgentMethod) Name() string {
	return "plumber.agent.update"
}

func (m *UpdateAgentMethod) RequireAuth() bool {
	return true
}

type UpdateAgentParams struct {
	AgentID       string `json:"agent_id"`
	Name          string `json:"name"`
	SSHHost       string `json:"ssh_host,omitempty"`
	SSHPort       int    `json:"ssh_port,omitempty"`
	SSHUser       string `json:"ssh_user,omitempty"`
	SSHAuthType   string `json:"ssh_auth_type,omitempty"`
	SSHPassword   string `json:"ssh_password,omitempty"`
	SSHPrivateKey string `json:"ssh_private_key,omitempty"`
}

func (m *UpdateAgentMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p UpdateAgentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	agent, err := m.storage.GetAgent(ctx, agentUUID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	// 更新字段
	if p.Name != "" {
		agent.Name = p.Name
	}
	agent.SSHHost = p.SSHHost
	agent.SSHPort = p.SSHPort
	agent.SSHUser = p.SSHUser
	agent.SSHAuthType = p.SSHAuthType
	agent.SSHPassword = p.SSHPassword
	agent.SSHPrivateKey = p.SSHPrivateKey

	if err := m.storage.UpdateAgent(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	return map[string]interface{}{
		"status":  "updated",
		"message": "Agent updated successfully",
	}, nil
}

// DeleteAgentMethod 删除Agent
type DeleteAgentMethod struct {
	storage storage.Storage
}

func NewDeleteAgentMethod(storage storage.Storage) *DeleteAgentMethod {
	return &DeleteAgentMethod{storage: storage}
}

func (m *DeleteAgentMethod) Name() string {
	return "plumber.agent.delete"
}

func (m *DeleteAgentMethod) RequireAuth() bool {
	return true
}

type DeleteAgentParams struct {
	AgentID string `json:"agent_id"`
}

func (m *DeleteAgentMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DeleteAgentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	if err := m.storage.DeleteAgent(ctx, agentUUID); err != nil {
		return nil, fmt.Errorf("failed to delete agent: %w", err)
	}

	return map[string]interface{}{
		"status":  "deleted",
		"message": "Agent deleted successfully",
	}, nil
}

// GetAgentConfigMethod 获取Agent配置文件
type GetAgentConfigMethod struct {
	storage    storage.Storage
	agentToken string
	serverAddr string
}

func NewGetAgentConfigMethod(storage storage.Storage, agentToken, serverAddr string) *GetAgentConfigMethod {
	return &GetAgentConfigMethod{
		storage:    storage,
		agentToken: agentToken,
		serverAddr: serverAddr,
	}
}

func (m *GetAgentConfigMethod) Name() string {
	return "plumber.agent.getConfig"
}

func (m *GetAgentConfigMethod) RequireAuth() bool {
	return true
}

type GetAgentConfigParams struct {
	AgentID string `json:"agent_id"`
}

func (m *GetAgentConfigMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p GetAgentConfigParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	// 验证Agent存在
	_, err = m.storage.GetAgent(ctx, agentUUID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	// 生成配置文件内容
	config := map[string]interface{}{
		"id":          p.AgentID,
		"token":       m.agentToken,
		"server_addr": m.serverAddr,
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	return map[string]interface{}{
		"config":   string(configJSON),
		"filename": "agent.json",
	}, nil
}

// DeployAgentMethod 部署Agent到服务器
type DeployAgentMethod struct {
	storage    storage.Storage
	agentToken string
	serverAddr string
}

func NewDeployAgentMethod(storage storage.Storage, agentToken, serverAddr string) *DeployAgentMethod {
	return &DeployAgentMethod{
		storage:    storage,
		agentToken: agentToken,
		serverAddr: serverAddr,
	}
}

func (m *DeployAgentMethod) Name() string {
	return "plumber.agent.deploy"
}

func (m *DeployAgentMethod) RequireAuth() bool {
	return true
}

type DeployAgentParams struct {
	AgentID    string `json:"agent_id"`
	ScriptURL  string `json:"script_url"`
	InstallDir string `json:"install_dir"`
}

func (m *DeployAgentMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p DeployAgentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	agentUUID, err := uuid.Parse(p.AgentID)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}

	agent, err := m.storage.GetAgent(ctx, agentUUID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	if agent.SSHHost == "" {
		return nil, fmt.Errorf("agent has no SSH configuration")
	}

	// 使用 SSH 执行部署
	output, err := m.deployViaSSH(agent, p)
	if err != nil {
		return nil, fmt.Errorf("deploy failed: %w", err)
	}

	return map[string]interface{}{
		"status": "success",
		"output": output,
	}, nil
}

func (m *DeployAgentMethod) deployViaSSH(agent *models.Agent, params DeployAgentParams) (string, error) {
	// 生成 agent 配置，使用服务器配置的地址
	config := map[string]interface{}{
		"id":          agent.ID.String(),
		"token":       m.agentToken,
		"server_addr": m.serverAddr,
	}
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to generate config: %w", err)
	}

	// 构建部署命令
	// 1. 删除安装目录
	// 2. 创建安装目录
	// 3. 写入配置文件
	// 4. 下载并执行安装脚本
	deployScript := fmt.Sprintf(`
set -e
echo "Step 1: Removing old installation directory..."
sudo rm -rf %s || true

echo "Step 2: Creating installation directory..."
sudo mkdir -p %s

echo "Step 3: Writing agent configuration..."
cat > /tmp/agent.json << 'EOF'
%s
EOF
sudo mv /tmp/agent.json %s/agent.json
sudo chmod 644 %s/agent.json

echo "Step 4: Downloading and executing installation script..."
curl -sSL "%s" | sudo bash

echo "Deployment completed successfully!"
`, params.InstallDir, params.InstallDir, string(configJSON), params.InstallDir, params.InstallDir, params.ScriptURL)

	// 根据认证类型构建 SSH 命令
	var sshCmd string
	if agent.SSHAuthType == "password" && agent.SSHPassword != "" {
		// 使用 sshpass
		sshCmd = fmt.Sprintf(`sshpass -p '%s' ssh -o StrictHostKeyChecking=no %s@%s -p %d '%s'`,
			agent.SSHPassword, agent.SSHUser, agent.SSHHost, agent.SSHPort, deployScript)
	} else if agent.SSHAuthType == "key" && agent.SSHPrivateKey != "" {
		// 暂时保存私钥到临时文件
		tmpKeyFile := fmt.Sprintf("/tmp/ssh_key_%s", agent.ID.String())
		if err := os.WriteFile(tmpKeyFile, []byte(agent.SSHPrivateKey), 0600); err != nil {
			return "", fmt.Errorf("failed to write SSH key: %w", err)
		}
		defer os.Remove(tmpKeyFile)

		sshCmd = fmt.Sprintf(`ssh -i %s -o StrictHostKeyChecking=no %s@%s -p %d '%s'`,
			tmpKeyFile, agent.SSHUser, agent.SSHHost, agent.SSHPort, deployScript)
	} else {
		// 无密码认证
		sshCmd = fmt.Sprintf(`ssh -o StrictHostKeyChecking=no %s@%s -p %d '%s'`,
			agent.SSHUser, agent.SSHHost, agent.SSHPort, deployScript)
	}

	// 执行 SSH 命令
	cmd := exec.Command("bash", "-c", sshCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("SSH command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// RegisterAllMethods 注册所有RPC方法
func RegisterAllMethods(router *jsonrpc.Router, storage storage.Storage, jwtManager *auth.JWTManager, adminUsername, adminPassword, agentToken, serverAddr string) {
	executor := NewTaskExecutor(storage)

	router.Register(NewAgentRegisterMethod(storage))
	router.Register(NewAgentHeartbeatMethod(storage))
	router.Register(NewUserLoginMethod(jwtManager, adminUsername, adminPassword))
	router.Register(NewListAgentsMethod(storage))
	router.Register(NewCreateAgentMethod(storage))
	router.Register(NewUpdateAgentMethod(storage))
	router.Register(NewDeleteAgentMethod(storage))
	router.Register(NewGetAgentConfigMethod(storage, agentToken, serverAddr))
	router.Register(NewDeployAgentMethod(storage, agentToken, serverAddr))
	router.Register(NewCreateTaskMethod(storage))
	router.Register(NewUpdateTaskMethod(storage))
	router.Register(NewListTasksMethod(storage))
	router.Register(NewPollTaskMethod(storage))
	router.Register(NewStepReportMethod(storage))
	router.Register(NewRunTaskMethod(storage, executor))
	router.Register(NewGetExecutionMethod(storage))
	router.Register(NewListExecutionsMethod(storage))
}
