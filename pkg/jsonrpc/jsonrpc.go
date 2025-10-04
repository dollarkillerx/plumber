package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
)

// Request JSON-RPC 2.0 请求结构
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// Response JSON-RPC 2.0 响应结构
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Error JSON-RPC 2.0 错误结构
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 标准错误代码
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Method JSON-RPC方法接口
type Method interface {
	Name() string
	Execute(ctx context.Context, params json.RawMessage) (interface{}, error)
	RequireAuth() bool
}

// Router JSON-RPC路由器
type Router struct {
	methods map[string]Method
}

// NewRouter 创建新的路由器
func NewRouter() *Router {
	return &Router{
		methods: make(map[string]Method),
	}
}

// Register 注册方法
func (r *Router) Register(method Method) {
	r.methods[method.Name()] = method
}

// Handle 处理JSON-RPC请求
func (r *Router) Handle(ctx context.Context, req *Request) *Response {
	method, exists := r.methods[req.Method]
	if !exists {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
			ID: req.ID,
		}
	}

	result, err := method.Execute(ctx, req.Params)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    InternalError,
				Message: err.Error(),
			},
			ID: req.ID,
		}
	}

	return &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

// GetMethod 获取方法
func (r *Router) GetMethod(name string) (Method, bool) {
	method, exists := r.methods[name]
	return method, exists
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id interface{}, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		Error: &Error{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(id interface{}, result interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}
