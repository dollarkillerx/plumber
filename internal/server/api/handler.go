package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/plumber/plumber/internal/server/storage"
	"github.com/plumber/plumber/pkg/auth"
	"github.com/plumber/plumber/pkg/jsonrpc"
)

// Handler HTTP处理器
type Handler struct {
	router     *jsonrpc.Router
	storage    storage.Storage
	jwtManager *auth.JWTManager
	agentToken string // Agent认证Token
}

// NewHandler 创建新的处理器
func NewHandler(router *jsonrpc.Router, storage storage.Storage, jwtManager *auth.JWTManager, agentToken string) *Handler {
	return &Handler{
		router:     router,
		storage:    storage,
		jwtManager: jwtManager,
		agentToken: agentToken,
	}
}

// ServeHTTP 处理HTTP请求
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Agent-Token")
	w.Header().Set("Content-Type", "application/json")

	// 处理预检请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		h.writeError(w, jsonrpc.InvalidRequest, "Only POST method is allowed")
		return
	}

	var req jsonrpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, jsonrpc.ParseError, "Failed to parse request")
		return
	}

	// 检查是否需要认证
	method, exists := h.router.GetMethod(req.Method)
	if !exists {
		h.writeError(w, jsonrpc.MethodNotFound, "Method not found")
		return
	}

	ctx := r.Context()

	// 检查是否为Agent方法（Agent主动上报的方法）
	isAgentMethod := req.Method == "plumber.agent.register" ||
		req.Method == "plumber.agent.heartbeat" ||
		strings.HasPrefix(req.Method, "plumber.step.")

	if isAgentMethod {
		// Agent方法使用Token认证
		agentToken := r.Header.Get("X-Agent-Token")
		if agentToken == "" || agentToken != h.agentToken {
			h.writeError(w, jsonrpc.InvalidRequest, "Invalid or missing agent token")
			return
		}
	} else if method.RequireAuth() {
		// 用户方法使用JWT认证
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.writeError(w, jsonrpc.InvalidRequest, "Authorization header required")
			return
		}

		// 验证Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			h.writeError(w, jsonrpc.InvalidRequest, "Invalid authorization header format")
			return
		}

		token := parts[1]
		claims, err := h.jwtManager.Verify(token)
		if err != nil {
			h.writeError(w, jsonrpc.InvalidRequest, "Invalid or expired token")
			return
		}

		// 将用户信息添加到context
		ctx = ContextWithUserID(ctx, claims.UserID)
		ctx = ContextWithUsername(ctx, claims.Username)
	}

	// 执行方法
	response := h.router.Handle(ctx, &req)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, code int, message string) {
	response := jsonrpc.NewErrorResponse(nil, code, message)
	json.NewEncoder(w).Encode(response)
}

// Context keys
type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
)

// ContextWithUserID 添加用户ID到context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// ContextWithUsername 添加用户名到context
func ContextWithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

// GetUserIDFromContext 从context获取用户ID
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

// GetUsernameFromContext 从context获取用户名
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey).(string)
	return username, ok
}
