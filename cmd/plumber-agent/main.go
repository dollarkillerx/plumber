package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/plumber/plumber/internal/agent/client"
	"github.com/plumber/plumber/internal/agent/executor"
)

const (
	agentIDFile     = ".plumber_agent_id"
	agentConfigFile = "agent.json"
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

	// 启动心跳
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agentClient.StartHeartbeat(ctx, 1*time.Second)
	log.Printf("Heartbeat started")

	// 启动任务轮询
	go agentClient.StartTaskPolling(ctx, exec, 500*time.Millisecond)
	log.Printf("Task polling started")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down agent...")
	cancel() // 取消context，停止心跳和任务轮询
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

// getOutboundIP 获取公网IPv4地址
func getOutboundIP() (string, error) {
	// 尝试多个公共IP查询服务（仅IPv4）
	services := []string{
		"https://api.ipify.org?format=text",
		"https://ipv4.icanhazip.com",
		"https://v4.ident.me",
		"https://api.my-ip.io/ip",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err == nil && len(body) > 0 {
				// 去除可能的空白字符
				ip := string(bytes.TrimSpace(body))
				if ip != "" {
					return ip, nil
				}
			}
		}
	}

	return "", fmt.Errorf("failed to get public IP from all services")
}
