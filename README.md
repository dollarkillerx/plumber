# Plumber

Plumber 是一个基于 C/S 架构的自动化任务编排与分发平台，用于统一管理多台服务器上的任务执行。

## 项目概述

Plumber 通过 Web 界面和命令行工具，简化批量运维、部署、批处理等场景下的自动化操作。

![](./image.png)

### 核心组件

- **Plumber Server** - 核心控制中心，负责任务管理、Agent 管理、任务分发和结果收集
- **Plumber Web** - Web 管理界面，提供可视化的任务编排、服务器管理和日志查看
- **Plumber Agent** - 轻量级代理，部署在目标服务器上执行任务
- **Plumber CLI** - 命令行工具，支持快速任务触发和状态查询

## 快速开始

### 方式一：使用 Docker（推荐）

1. 克隆仓库并进入目录：

```bash
git clone <repository-url>
cd plumber
```

2. 配置 Server：

```bash
# 复制配置文件模板
cp configs/server.toml.example configs/server.toml

# 编辑配置文件
vim configs/server.toml
```

修改数据库配置：

```toml
[database]
host = "your-db-host"
port = 5432
user = "plumber"
password = "your-password"
dbname = "plumber"
```

3. 构建 Docker 镜像：

```bash
make docker-build
# 或者
docker build -t plumber-server:latest .
```

4. 启动服务：

```bash
# 使用 docker-compose（推荐）
docker-compose up -d

# 或直接运行 Docker
docker run -d \
  --name plumber-server \
  -p 52281:52281 \
  -v $(pwd)/configs/server.toml:/app/configs/server.toml:ro \
  plumber-server:latest
```

5. 检查服务状态：

```bash
# 查看容器状态
docker ps

# 查看日志
docker logs -f plumber-server

# 健康检查
curl http://localhost:52281/health
```

6. 停止服务：

```bash
docker-compose down
# 或
docker stop plumber-server
docker rm plumber-server
```

### 方式二：手动运行

#### 启动 Server

```bash
# 配置数据库连接
export DATABASE_URL="postgres://user:password@localhost:5432/plumber?sslmode=disable"

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

## Docker 部署

### Dockerfile 说明

项目使用多阶段构建，优化镜像大小：

```dockerfile
# 第一阶段：构建
FROM golang:1.24-alpine AS builder
# ... 编译 Go 代码

# 第二阶段：运行
FROM alpine:latest
# ... 只包含二进制文件
```

最终镜像大小约 **20-30MB**。

### 配置文件

Server 使用 TOML 配置文件（**不是环境变量**）：

```bash
# 复制配置文件模板
cp configs/server.toml.example configs/server.toml

# 编辑配置
vim configs/server.toml
```

配置文件包含：
- Server 监听地址和端口
- 数据库连接信息
- JWT 和认证配置

### 常用命令

```bash
# 构建镜像
make docker-build

# 运行容器（挂载配置文件）
docker run -d \
  --name plumber-server \
  -p 52281:52281 \
  -v $(pwd)/configs/server.toml:/app/configs/server.toml:ro \
  plumber-server:latest

# 查看日志
docker logs -f plumber-server

# 重启容器
docker restart plumber-server

# 停止并删除容器
docker stop plumber-server
docker rm plumber-server

# 健康检查
curl http://localhost:52281/health
```

### 使用 Docker Compose

配置文件会自动挂载到容器：

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 生产环境部署

```bash
# 使用生产环境配置
docker-compose -f docker-compose.prod.yml up -d
```

生产环境特性：
- 配置文件只读挂载
- 日志轮转（最大 10MB，保留 3 个文件）
- 自动重启策略
- 健康检查监控

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

### Docker 网络配置

如果 Agent 需要连接到 Docker 容器中的 Server：

```json
{
  "id": "agent-uuid",
  "token": "agent-token",
  "server_addr": "http://host.docker.internal:52281"
}
```

或者使用宿主机 IP 地址。

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
