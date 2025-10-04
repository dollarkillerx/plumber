# Plumber

Plumber 是一个基于 C/S 架构的自动化任务编排与分发平台，用于统一管理多台服务器上的任务执行。

## 项目概述

Plumber 通过 Web 界面和命令行工具，简化批量运维、部署、批处理等场景下的自动化操作。

### 核心组件

- **Plumber Server** - 核心控制中心，负责任务管理、Agent 管理、任务分发和结果收集
- **Plumber Web** - Web 管理界面，提供可视化的任务编排、服务器管理和日志查看
- **Plumber Agent** - 轻量级代理，部署在目标服务器上执行任务
- **Plumber CLI** - 命令行工具，支持快速任务触发和状态查询

## 快速开始

### 启动 Server

```bash
# 运行 Server
go run cmd/plumber-server/main.go
```

Server 默认监听端口：`52281`

### 启动 Web 界面

```bash
cd plumber-web
pnpm install
pnpm dev
```

Web 界面访问地址：`http://localhost:5173`

### 部署 Agent

1. 在 Web 界面创建配置文件 `agent.json`：

```json
{
  "id": "agent-uuid-here",
  "token": "agent-token-here",
  "server_addr": "http://server-ip:52281"
}
```

2. 运行 Agent：

```bash
go run cmd/plumber-agent/main.go --config agent.json
```

Agent 特性：
- ✅ 每 1 秒发送心跳
- ✅ 每 500ms 拉取待执行任务
- ✅ 无需监听端口（适合 NAT 环境）

## 核心特性

### 任务编排
- 可视化的步骤构建器
- 支持多服务器、多步骤任务
- 顺序执行保证
- 实时状态追踪

### 任务执行
- **Pull 模式**：Agent 主动拉取任务，适合 NAT 环境
- 数据库行锁防止任务重复分配
- 自动检查前序步骤完成状态
- 完整的命令输出捕获

### 执行历史
- 查看最近 20 次执行记录
- 每个步骤的详细信息：
  - 执行路径和命令
  - 退出码
  - 完整的标准输出/错误输出
  - 执行耗时

### 服务器管理
- Agent 在线状态监控
- 心跳检测
- 一键复制 Agent UUID

## 技术栈

### 后端
- Go 1.24+
- PostgreSQL
- GORM
- JSON-RPC 2.0

### 前端
- Vue 3
- TypeScript
- Pinia
- Tailwind CSS

## 架构设计

### 任务分发流程

```
用户触发任务
    ↓
Server 创建步骤记录 (status=pending)
    ↓
Agent 轮询拉取任务 (500ms 间隔)
    ↓
Server 返回可执行步骤（带行锁）
    ↓
Agent 执行命令
    ↓
Agent 上报结果 (status=success/failed)
    ↓
Server 更新状态，准备下一步骤
```

### 并发安全机制

- PostgreSQL `FOR UPDATE SKIP LOCKED` 行锁
- 事务内原子查询和标记
- 前序步骤完成状态检查

## 项目结构

```
plumber/
├── cmd/
│   ├── plumber-server/    # Server 入口
│   ├── plumber-agent/     # Agent 入口
│   └── plumber-cli/       # CLI 工具
├── internal/
│   ├── server/
│   │   ├── api/          # RPC 方法和任务执行器
│   │   └── storage/      # 数据库操作
│   └── agent/
│       ├── client/       # Agent RPC 客户端
│       └── executor/     # 命令执行器
├── pkg/
│   ├── models/           # 数据模型
│   └── jsonrpc/          # JSON-RPC 框架
├── plumber-web/          # Vue 前端
└── SRS.md               # 软件需求规格说明书
```

## 配置说明

### Server 配置

环境变量：
- `DATABASE_URL` - PostgreSQL 连接字符串
- `SERVER_PORT` - Server 监听端口（默认 52281）
- `DEBUG` - 是否开启调试日志

### Agent 配置

`agent.json` 文件：
```json
{
  "id": "agent-uuid",
  "token": "agent-token",
  "server_addr": "http://server:52281"
}
```

命令行参数：
- `--config` - 配置文件路径（默认 agent.json）
- `--workdir` - 默认工作目录（默认 /tmp）

## API 文档

详细的 JSON-RPC API 文档请参考 [SRS.md](./SRS.md)

## 开发指南

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/server/storage
```

### 构建

```bash
# 构建 Server
go build -o bin/plumber-server cmd/plumber-server/main.go

# 构建 Agent
go build -o bin/plumber-agent cmd/plumber-agent/main.go

# 构建 Web
cd plumber-web
pnpm build
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！
