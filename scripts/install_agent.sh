#!/bin/bash

# Plumber Agent 自动安装脚本

set -e

PLUMBER_SERVER=${PLUMBER_SERVER:-"http://localhost:52181"}
AGENT_TOKEN=${AGENT_TOKEN:-""}
INSTALL_DIR=${INSTALL_DIR:-"/opt/plumber"}
AGENT_BINARY=${AGENT_BINARY:-"plumber-agent"}

echo "=== Plumber Agent Installation ==="
echo "Server URL: $PLUMBER_SERVER"
echo "Install directory: $INSTALL_DIR"

# 检查必需参数
if [ -z "$AGENT_TOKEN" ]; then
    echo "Error: AGENT_TOKEN is required"
    echo "Usage: AGENT_TOKEN=your-token PLUMBER_SERVER=http://server:52181 bash install_agent.sh"
    exit 1
fi

# 检查是否为root用户
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root or with sudo"
    exit 1
fi

# 创建安装目录
mkdir -p $INSTALL_DIR
cd $INSTALL_DIR

# 下载Agent二进制文件 (假设从服务器下载)
echo "Downloading agent binary..."
# TODO: 实现从服务器下载逻辑
# curl -o $AGENT_BINARY $PLUMBER_SERVER/download/agent
# 目前假设二进制文件已经存在
if [ ! -f "$AGENT_BINARY" ]; then
    echo "Error: Agent binary not found"
    echo "Please copy the plumber-agent binary to $INSTALL_DIR"
    exit 1
fi

chmod +x $AGENT_BINARY

# 创建systemd服务文件
cat > /etc/systemd/system/plumber-agent.service <<EOF
[Unit]
Description=Plumber Agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$AGENT_BINARY -server=$PLUMBER_SERVER -token=$AGENT_TOKEN
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

# 重新加载systemd配置
systemctl daemon-reload

# 启动服务
systemctl enable plumber-agent
systemctl start plumber-agent

echo "=== Installation Complete ==="
echo "Agent service status:"
systemctl status plumber-agent --no-pager

echo ""
echo "Useful commands:"
echo "  systemctl status plumber-agent   - Check status"
echo "  systemctl stop plumber-agent     - Stop service"
echo "  systemctl start plumber-agent    - Start service"
echo "  systemctl restart plumber-agent  - Restart service"
echo "  journalctl -u plumber-agent -f   - View logs"
