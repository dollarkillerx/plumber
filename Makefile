.PHONY: all build build-server build-agent build-cli clean test deps fmt run-server run-agent install

# 构建变量
VERSION ?= 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 输出目录
BIN_DIR := bin
SERVER_BIN := $(BIN_DIR)/plumber-server
AGENT_BIN := $(BIN_DIR)/plumber-agent
CLI_BIN := $(BIN_DIR)/plumber-cli

all: deps build

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# 构建所有组件
build: build-server build-agent build-cli

# 构建Server
build-server:
	@echo "Building Plumber Server..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(SERVER_BIN) ./cmd/plumber-server

# 构建Agent
build-agent:
	@echo "Building Plumber Agent..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(AGENT_BIN) ./cmd/plumber-agent

# 构建CLI
build-cli:
	@echo "Building Plumber CLI..."
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(CLI_BIN) ./cmd/plumber-cli

# 交叉编译
build-linux-amd64:
	@echo "Building for Linux amd64..."
	@mkdir -p $(BIN_DIR)/linux-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/linux-amd64/plumber-server ./cmd/plumber-server
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/linux-amd64/plumber-agent ./cmd/plumber-agent
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/linux-amd64/plumber-cli ./cmd/plumber-cli

build-linux-arm64:
	@echo "Building for Linux arm64..."
	@mkdir -p $(BIN_DIR)/linux-arm64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/linux-arm64/plumber-server ./cmd/plumber-server
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/linux-arm64/plumber-agent ./cmd/plumber-agent
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/linux-arm64/plumber-cli ./cmd/plumber-cli

build-darwin-amd64:
	@echo "Building for macOS amd64..."
	@mkdir -p $(BIN_DIR)/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin-amd64/plumber-server ./cmd/plumber-server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin-amd64/plumber-agent ./cmd/plumber-agent
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin-amd64/plumber-cli ./cmd/plumber-cli

build-darwin-arm64:
	@echo "Building for macOS arm64..."
	@mkdir -p $(BIN_DIR)/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin-arm64/plumber-server ./cmd/plumber-server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin-arm64/plumber-agent ./cmd/plumber-agent
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BIN_DIR)/darwin-arm64/plumber-cli ./cmd/plumber-cli

# 构建所有平台
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

# 运行Server
run-server: build-server
	@echo "Starting Plumber Server..."
	$(SERVER_BIN) -config configs/server.toml

# 运行Agent
run-agent: build-agent
	@echo "Starting Plumber Agent..."
	$(AGENT_BIN) -server http://localhost:52181

# 测试
test:
	@echo "Running tests..."
	go test -v ./...

# 测试覆盖率
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 代码格式化
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# 代码检查
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# 清理
clean:
	@echo "Cleaning..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

# 安装到系统
install: build
	@echo "Installing binaries..."
	@mkdir -p ~/bin
	cp $(SERVER_BIN) ~/bin/
	cp $(AGENT_BIN) ~/bin/
	cp $(CLI_BIN) ~/bin/
	@echo "Installed to ~/bin/"
	@echo "Make sure ~/bin is in your PATH"

# Docker构建
docker-build:
	@echo "Building Docker image..."
	docker build -t plumber-server:$(VERSION) -t plumber-server:latest .

# 帮助
help:
	@echo "Plumber Makefile Commands:"
	@echo "  make deps              - Install dependencies"
	@echo "  make build             - Build all components"
	@echo "  make build-server      - Build server only"
	@echo "  make build-agent       - Build agent only"
	@echo "  make build-cli         - Build CLI only"
	@echo "  make build-all         - Build for all platforms"
	@echo "  make run-server        - Build and run server"
	@echo "  make run-agent         - Build and run agent"
	@echo "  make test              - Run tests"
	@echo "  make test-coverage     - Run tests with coverage"
	@echo "  make fmt               - Format code"
	@echo "  make lint              - Run linter"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make install           - Install to ~/bin"
	@echo "  make docker-build      - Build Docker images"
