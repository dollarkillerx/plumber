package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/plumber/plumber/internal/server/api"
	"github.com/plumber/plumber/internal/server/config"
	"github.com/plumber/plumber/internal/server/storage"
	"github.com/plumber/plumber/internal/server/webssh"
	"github.com/plumber/plumber/pkg/auth"
	"github.com/plumber/plumber/pkg/jsonrpc"
)

func main() {
	configPath := flag.String("config", "configs/server.toml", "Path to config file")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化存储
	store, err := storage.NewPostgresStorage(cfg.Database.GetDSN(), cfg.Server.Debug)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// 初始化JWT管理器
	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.TokenExpiration)*time.Hour,
	)

	// 设置服务器地址（用于 agent 配置文件中的 server_addr）
	exportEndpoint := cfg.Server.ExportEndpoint
	if exportEndpoint == "" {
		// 如果没有配置 export_endpoint，使用 public_addr
		exportEndpoint = cfg.Server.PublicAddr
		if exportEndpoint == "" {
			exportEndpoint = fmt.Sprintf("http://%s:%s", cfg.Server.Host, cfg.Server.Port)
		}
	}

	// 初始化JSON-RPC路由器
	router := jsonrpc.NewRouter()
	api.RegisterAllMethods(router, store, jwtManager, cfg.Auth.AdminUsername, cfg.Auth.AdminPassword, cfg.Auth.AgentToken, exportEndpoint)

	// 创建HTTP处理器
	apiHandler := api.NewHandler(router, store, jwtManager, cfg.Auth.AgentToken)

	// 创建 RESTful API 处理器
	restHandler := api.NewRestHandler(store, cfg.Auth.AgentToken, exportEndpoint)

	// 创建 WebSSH 处理器
	encryptionKey := []byte{}
	if cfg.Auth.EncryptionKey != "" {
		encryptionKey = []byte(cfg.Auth.EncryptionKey)
	}
	websshHandler := webssh.NewWebSSHHandler(store, encryptionKey)

	// 创建路由
	mux := http.NewServeMux()
	mux.Handle("/api/rpc", apiHandler)
	mux.Handle("/api/webssh", websshHandler)
	mux.HandleFunc("/api/agent/config/", restHandler.GetAgentConfig)

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		log.Printf("Plumber Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 启动Agent心跳检查
	go startHeartbeatChecker(store)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// startHeartbeatChecker 启动心跳检查器
func startHeartbeatChecker(store storage.Storage) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		agents, err := store.ListAgents(ctx)
		if err != nil {
			log.Printf("Failed to list agents: %v", err)
			continue
		}

		for _, agent := range agents {
			// 如果超过1分钟没有心跳,标记为离线
			if agent.LastHeartbeat != nil && time.Since(*agent.LastHeartbeat) > time.Minute {
				if agent.Status == "online" {
					if err := store.UpdateAgentStatus(ctx, agent.ID, "offline"); err != nil {
						log.Printf("Failed to update agent %s status: %v", agent.ID, err)
					} else {
						log.Printf("Agent %s marked as offline", agent.ID)
					}
				}
			}
		}
	}
}
