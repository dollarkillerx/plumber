# Plumber 部署指南

本文档介绍如何部署和运行 Plumber 系统。

## 系统要求

### Plumber Server
- Go 1.24+
- PostgreSQL 12+
- 至少 512MB 内存
- 至少 1GB 磁盘空间

### Plumber Agent
- Go 1.24+ (编译时)
- 至少 128MB 内存
- 支持的操作系统: Linux, macOS, Windows

## 快速开始

### 1. 准备数据库

安装并启动 PostgreSQL:

```bash
# macOS
brew install postgresql
brew services start postgresql

# Ubuntu/Debian
sudo apt install postgresql
sudo systemctl start postgresql

# 创建数据库和用户
sudo -u postgres psql
```

在 PostgreSQL 中执行:
```sql
CREATE USER plumber WITH PASSWORD 'plumber123';
CREATE DATABASE plumber OWNER plumber;
\q
```

### 2. 构建项目

```bash
# 克隆项目
cd /Users/github/Documents/workspace/plumber

# 安装依赖
make deps

# 构建所有组件
make build
```

构建产物在 `bin/` 目录:
- `plumber-server` - 服务器
- `plumber-agent` - Agent
- `plumber-cli` - 命令行工具

### 3. 配置 Server

编辑 `configs/server.toml`:

```toml
[server]
host = "0.0.0.0"
port = "52181"
debug = true

[database]
host = "localhost"
port = 5432
user = "plumber"
password = "plumber123"
dbname = "plumber"
sslmode = "disable"

[auth]
jwt_secret = "your-secret-key-change-in-production"
token_expiration = 8760  # 1年
```

**重要**: 在生产环境中，请修改 `jwt_secret` 为随机字符串！

### 4. 初始化数据库

```bash
# 执行初始化SQL脚本
psql -U plumber -d plumber -f scripts/init_db.sql
```

这将创建必要的表结构并添加默认管理员用户:
- 用户名: `admin`
- 密码: `admin123`

### 5. 启动 Server

```bash
# 方式1: 直接运行
./bin/plumber-server -config configs/server.toml

# 方式2: 使用 Make
make run-server
```

Server 启动后监听在 `0.0.0.0:52181`

### 6. 部署 Agent

#### 方式 A: 手动部署

在目标服务器上:

```bash
# 复制二进制文件
scp bin/plumber-agent user@remote-server:/opt/plumber/

# SSH登录目标服务器
ssh user@remote-server

# 运行Agent
cd /opt/plumber
./plumber-agent -server http://your-server-ip:52181
```

#### 方式 B: 使用安装脚本 (Linux)

```bash
# 复制文件到目标服务器
scp bin/plumber-agent user@remote-server:/opt/plumber/
scp scripts/install_agent.sh user@remote-server:/tmp/

# SSH登录并安装
ssh user@remote-server
sudo PLUMBER_SERVER=http://your-server-ip:52181 bash /tmp/install_agent.sh
```

这将:
- 安装 Agent 到 `/opt/plumber`
- 创建 systemd 服务
- 自动启动并设置开机自启

### 7. 使用 CLI

```bash
# 获取登录令牌
curl -X POST http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "plumber.user.login",
    "params": {"username": "admin", "password": "admin123"},
    "id": "1"
  }' | jq

# 配置CLI
./bin/plumber-cli set-config \
  --url http://localhost:52181 \
  --token <your-token>

# 查看Agent列表
./bin/plumber-cli agent list

# 查看任务列表
./bin/plumber-cli task list
```

---

## 生产环境部署

### 使用 systemd (Linux)

#### Plumber Server

创建 `/etc/systemd/system/plumber-server.service`:

```ini
[Unit]
Description=Plumber Server
After=network.target postgresql.service

[Service]
Type=simple
User=plumber
Group=plumber
WorkingDirectory=/opt/plumber
ExecStart=/opt/plumber/bin/plumber-server -config /opt/plumber/configs/server.toml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务:

```bash
sudo systemctl daemon-reload
sudo systemctl enable plumber-server
sudo systemctl start plumber-server
sudo systemctl status plumber-server
```

#### Plumber Agent

已包含在 `scripts/install_agent.sh` 中，会自动创建服务。

### 使用 Docker

#### 构建镜像

```bash
# Server镜像
docker build -t plumber-server:1.0.0 -f docker/Dockerfile.server .

# Agent镜像
docker build -t plumber-agent:1.0.0 -f docker/Dockerfile.agent .
```

#### 运行容器

```bash
# 启动PostgreSQL
docker run -d \
  --name plumber-postgres \
  -e POSTGRES_USER=plumber \
  -e POSTGRES_PASSWORD=plumber123 \
  -e POSTGRES_DB=plumber \
  -v plumber-db:/var/lib/postgresql/data \
  postgres:14

# 启动Server
docker run -d \
  --name plumber-server \
  -p 52181:52181 \
  --link plumber-postgres:postgres \
  -v $(pwd)/configs:/app/configs \
  plumber-server:1.0.0

# 启动Agent
docker run -d \
  --name plumber-agent \
  -e PLUMBER_SERVER=http://server-ip:52181 \
  plumber-agent:1.0.0
```

### 使用 Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_USER: plumber
      POSTGRES_PASSWORD: plumber123
      POSTGRES_DB: plumber
    volumes:
      - plumber-db:/var/lib/postgresql/data
    networks:
      - plumber-net

  server:
    build:
      context: .
      dockerfile: docker/Dockerfile.server
    ports:
      - "52181:52181"
    depends_on:
      - postgres
    volumes:
      - ./configs:/app/configs
    networks:
      - plumber-net

  agent:
    build:
      context: .
      dockerfile: docker/Dockerfile.agent
    environment:
      PLUMBER_SERVER: http://server:52181
    depends_on:
      - server
    networks:
      - plumber-net

volumes:
  plumber-db:

networks:
  plumber-net:
```

启动:

```bash
docker-compose up -d
```

---

## 安全建议

### 1. 更改默认密码

首次登录后立即更改管理员密码。

### 2. 使用 HTTPS

在生产环境中，建议在 Server 前部署 Nginx/Caddy 并配置 TLS:

```nginx
server {
    listen 443 ssl http2;
    server_name plumber.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:52181;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 3. 防火墙配置

```bash
# 只允许信任的IP访问Server
sudo ufw allow from trusted-ip to any port 52181

# Agent端口不应对外开放
sudo ufw deny 52182
```

### 4. 定期备份数据库

```bash
# 备份
pg_dump -U plumber plumber > backup-$(date +%Y%m%d).sql

# 恢复
psql -U plumber plumber < backup-20240101.sql
```

---

## 监控与日志

### 查看日志

```bash
# Server日志 (systemd)
sudo journalctl -u plumber-server -f

# Agent日志 (systemd)
sudo journalctl -u plumber-agent -f

# Docker日志
docker logs -f plumber-server
docker logs -f plumber-agent
```

### 健康检查

```bash
# 检查Server状态
curl http://localhost:52181/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"plumber.agent.list","params":{},"id":"1"}'

# 检查数据库连接
psql -U plumber -d plumber -c "SELECT COUNT(*) FROM agents;"
```

---

## 故障排查

### Server无法启动

1. 检查数据库连接:
```bash
psql -U plumber -h localhost -d plumber
```

2. 检查配置文件语法:
```bash
cat configs/server.toml
```

3. 查看详细日志:
```bash
./bin/plumber-server -config configs/server.toml
```

### Agent无法注册

1. 检查网络连通性:
```bash
curl http://server-ip:52181/rpc
```

2. 检查Agent配置:
```bash
./bin/plumber-agent -server http://server-ip:52181
```

3. 查看Server端日志确认注册请求

### 任务执行失败

1. 检查Agent状态:
```bash
./bin/plumber-cli agent list
```

2. 查看执行记录:
```bash
./bin/plumber-cli execution get <execution_id>
```

3. 检查命令权限和路径是否正确

---

## 性能优化

### 数据库优化

编辑 PostgreSQL 配置 `/etc/postgresql/*/main/postgresql.conf`:

```ini
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
```

### Server优化

- 增加数据库连接池大小
- 启用日志轮转
- 定期清理历史执行记录

---

## 升级指南

### 升级步骤

1. 备份数据库
```bash
pg_dump -U plumber plumber > backup-before-upgrade.sql
```

2. 停止服务
```bash
sudo systemctl stop plumber-server
sudo systemctl stop plumber-agent
```

3. 替换二进制文件
```bash
cp bin/plumber-server /opt/plumber/bin/
cp bin/plumber-agent /opt/plumber/bin/
```

4. 运行数据库迁移 (如有)

5. 启动服务
```bash
sudo systemctl start plumber-server
sudo systemctl start plumber-agent
```

6. 验证功能
```bash
./bin/plumber-cli agent list
```

---

如有问题，请参考 [API.md](API.md) 或提交 Issue。
