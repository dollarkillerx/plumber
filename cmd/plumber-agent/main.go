package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/plumber/plumber/internal/agent/client"
	"github.com/plumber/plumber/internal/agent/executor"
	"github.com/plumber/plumber/pkg/jsonrpc"
)

const (
	agentIDFile     = ".plumber_agent_id"
	agentConfigFile = "agent.json"
	agentPort       = "52182"
)

var (
	configPath = flag.String("config", "agent.json", "Path to agent configuration file")
	workDir    = flag.String("workdir", "/tmp", "Default working directory")
)

// AgentConfig Agent配置文件结构
type AgentConfig struct {
	ID         string `json:"id"`
	Token      string `json:"token"`
	ServerAddr string `json:"server_addr"`
}

func main() {
	flag.Parse()

	// 加载配置文件
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 解析Agent ID
	agentID, err := uuid.Parse(config.ID)
	if err != nil {
		log.Fatalf("Invalid agent ID in config: %v", err)
	}

	log.Printf("Agent ID: %s", agentID)
	log.Printf("Server: %s", config.ServerAddr)

	// 获取主机名和IP
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v", err)
	}

	ip, err := getOutboundIP()
	if err != nil {
		log.Fatalf("Failed to get IP: %v", err)
	}

	// 创建客户端
	agentClient := client.NewClient(config.ServerAddr, agentID, config.Token)

	// 注册Agent
	if err := agentClient.Register(hostname, ip); err != nil {
		log.Fatalf("Failed to register agent: %v", err)
	}
	log.Printf("Agent registered successfully")

	// 创建执行器
	exec := executor.NewExecutor(*workDir)

	// 启动JSON-RPC服务器
	router := jsonrpc.NewRouter()
	router.Register(&ExecuteCommandMethod{
		executor: exec,
		client:   agentClient,
	})

	handler := &RPCHandler{router: router}

	srv := &http.Server{
		Addr:    ":" + agentPort,
		Handler: handler,
	}

	go func() {
		log.Printf("Agent RPC server starting on port %s", agentPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 启动心跳
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentClient.StartHeartbeat(ctx, 30*time.Second)
	log.Printf("Heartbeat started")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down agent...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Agent forced to shutdown: %v", err)
	}

	log.Println("Agent exited")
}

// loadConfig 加载配置文件
func loadConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 验证必需字段
	if config.ID == "" {
		return nil, fmt.Errorf("agent id is required in config")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("agent token is required in config")
	}
	if config.ServerAddr == "" {
		return nil, fmt.Errorf("server address is required in config")
	}

	return &config, nil
}

// getOutboundIP 获取本机IP
func getOutboundIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// RPCHandler JSON-RPC处理器
type RPCHandler struct {
	router *jsonrpc.Router
}

func (h *RPCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequest, "Only POST method is allowed")
		return
	}

	var req jsonrpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.ParseError, "Failed to parse request")
		json.NewEncoder(w).Encode(resp)
		return
	}

	response := h.router.Handle(r.Context(), &req)
	json.NewEncoder(w).Encode(response)
}

// ExecuteCommandMethod 执行命令方法
type ExecuteCommandMethod struct {
	executor *executor.Executor
	client   *client.Client
}

func (m *ExecuteCommandMethod) Name() string {
	return "plumber.agent.execute"
}

func (m *ExecuteCommandMethod) RequireAuth() bool {
	return false
}

type ExecuteCommandParams struct {
	StepID  string `json:"step_id"`
	Path    string `json:"path"`
	Command string `json:"command"`
}

func (m *ExecuteCommandMethod) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p ExecuteCommandParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	stepID, err := uuid.Parse(p.StepID)
	if err != nil {
		return nil, fmt.Errorf("invalid step_id: %w", err)
	}

	// 异步执行命令
	go func() {
		result := m.executor.ExecuteWithTimeout(p.Path, p.Command, 10*time.Minute)

		status := "success"
		if result.ExitCode != 0 {
			status = "failed"
		}

		// 上报结果
		if err := m.client.ReportStepResult(stepID, status, result.ExitCode, result.Output); err != nil {
			log.Printf("Failed to report step result: %v", err)
		}
	}()

	return map[string]interface{}{
		"status":  "accepted",
		"message": "Command execution started",
	}, nil
}
