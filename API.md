# Plumber API 文档

Plumber 使用 JSON-RPC 2.0 协议进行通信。所有请求发送到 `/rpc` 端点。

## 基础信息

- **协议**: JSON-RPC 2.0
- **默认端口**: 52181 (Server), 52182 (Agent)
- **内容类型**: application/json

## 认证

需要认证的方法必须在HTTP请求头中包含JWT令牌:

```
Authorization: Bearer <token>
```

## API 方法

### 1. 用户登录

登录并获取访问令牌。

**方法**: `plumber.user.login`

**需要认证**: 否

**请求参数**:
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "username": "admin",
    "user_id": "uuid"
  },
  "id": "1"
}
```

**示例**:
```bash
curl -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "plumber.user.login",
    "params": {
      "username": "admin",
      "password": "admin123"
    },
    "id": "1"
  }'
```

---

### 2. Agent 注册

Agent 向 Server 注册。

**方法**: `plumber.agent.register`

**需要认证**: 否

**请求参数**:
```json
{
  "agent_id": "uuid",
  "hostname": "server-01",
  "ip": "192.168.1.100"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "created",
    "message": "Agent registered successfully"
  },
  "id": "1"
}
```

---

### 3. Agent 心跳

Agent 发送心跳保持在线状态。

**方法**: `plumber.agent.heartbeat`

**需要认证**: 否

**请求参数**:
```json
{
  "agent_id": "uuid"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "ok"
  },
  "id": "1"
}
```

---

### 4. 列出所有 Agent

获取所有已注册的 Agent 列表。

**方法**: `plumber.agent.list`

**需要认证**: 是

**请求参数**: 无

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "agents": [
      {
        "id": "uuid",
        "hostname": "server-01",
        "ip": "192.168.1.100",
        "status": "online",
        "last_heartbeat": "2024-01-01T10:00:00Z"
      }
    ]
  },
  "id": "1"
}
```

**示例**:
```bash
curl -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "jsonrpc": "2.0",
    "method": "plumber.agent.list",
    "params": {},
    "id": "1"
  }'
```

---

### 5. 创建任务

创建新的任务定义。

**方法**: `plumber.task.create`

**需要认证**: 是

**请求参数**:
```json
{
  "name": "Deploy App",
  "description": "Deploy application to production",
  "config": "[[step]]\nServerID = \"uuid\"\nPath = \"/opt/app\"\nCMD = \"git pull\""
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "task_id": "uuid",
    "status": "created"
  },
  "id": "1"
}
```

---

### 6. 列出所有任务

获取所有任务列表。

**方法**: `plumber.task.list`

**需要认证**: 是

**请求参数**: 无

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "tasks": [
      {
        "id": "uuid",
        "name": "Deploy App",
        "description": "Deploy application",
        "status": "pending"
      }
    ]
  },
  "id": "1"
}
```

---

### 7. 执行任务

运行指定任务。

**方法**: `plumber.task.run`

**需要认证**: 是

**请求参数**:
```json
{
  "task_id": "uuid"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "started",
    "message": "Task execution started"
  },
  "id": "1"
}
```

---

### 8. 获取执行记录

获取任务执行的详细记录。

**方法**: `plumber.execution.get`

**需要认证**: 是

**请求参数**:
```json
{
  "execution_id": "uuid"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "execution": {
      "id": "uuid",
      "task_id": "uuid",
      "status": "success",
      "start_time": "2024-01-01T10:00:00Z",
      "end_time": "2024-01-01T10:05:00Z",
      "steps": [
        {
          "id": "uuid",
          "step_index": 0,
          "agent_id": "uuid",
          "command": "git pull",
          "status": "success",
          "exit_code": 0,
          "output": "Already up to date."
        }
      ]
    }
  },
  "id": "1"
}
```

---

### 9. Agent 执行命令 (Agent内部使用)

Server 向 Agent 发送命令执行请求。

**方法**: `plumber.agent.execute`

**需要认证**: 否

**请求参数**:
```json
{
  "step_id": "uuid",
  "path": "/opt/app",
  "command": "git pull"
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "accepted",
    "message": "Command execution started"
  },
  "id": "1"
}
```

---

### 10. 上报步骤结果 (Agent内部使用)

Agent 向 Server 上报执行结果。

**方法**: `plumber.step.report`

**需要认证**: 否

**请求参数**:
```json
{
  "step_id": "uuid",
  "status": "success",
  "exit_code": 0,
  "output": "command output..."
}
```

**响应**:
```json
{
  "jsonrpc": "2.0",
  "result": {
    "status": "updated"
  },
  "id": "1"
}
```

---

## 错误代码

JSON-RPC 2.0 标准错误代码:

- `-32700`: 解析错误 (Parse error)
- `-32600`: 无效请求 (Invalid Request)
- `-32601`: 方法不存在 (Method not found)
- `-32602`: 无效参数 (Invalid params)
- `-32603`: 内部错误 (Internal error)

**错误响应示例**:
```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32601,
    "message": "Method not found"
  },
  "id": "1"
}
```

---

## 完整工作流程示例

### 1. 管理员登录
```bash
# 登录获取token
TOKEN=$(curl -s -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "plumber.user.login",
    "params": {"username": "admin", "password": "admin123"},
    "id": "1"
  }' | jq -r '.result.token')
```

### 2. 查看Agent列表
```bash
curl -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "method": "plumber.agent.list",
    "params": {},
    "id": "1"
  }' | jq
```

### 3. 创建任务
```bash
TASK_ID=$(curl -s -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "jsonrpc": "2.0",
    "method": "plumber.task.create",
    "params": {
      "name": "Test Task",
      "description": "Test deployment",
      "config": "[[step]]\nServerID = \"agent-uuid\"\nPath = \"/tmp\"\nCMD = \"echo Hello\""
    },
    "id": "1"
  }' | jq -r '.result.task_id')
```

### 4. 执行任务
```bash
curl -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"jsonrpc\": \"2.0\",
    \"method\": \"plumber.task.run\",
    \"params\": {\"task_id\": \"$TASK_ID\"},
    \"id\": \"1\"
  }" | jq
```

---

## 使用 Plumber CLI

更简单的方式是使用 CLI 工具:

```bash
# 配置
plumber-cli set-config --url http://localhost:52181 --token $TOKEN

# 列出Agent
plumber-cli agent list

# 列出任务
plumber-cli task list

# 执行任务
plumber-cli task run <task_id>
```
