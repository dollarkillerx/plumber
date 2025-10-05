#!/bin/bash

# Plumber Agent 自动安装脚本

set -e

DOWNLOAD_URL="https://github.com/dollarkillerx/plumber/releases/download/v0.0.1/plumber-agent-linux-x86"
SYSTEMD_URL="https://raw.githubusercontent.com/dollarkillerx/plumber/refs/heads/main/plumber-agent.service"
INSTALL_DIR="/opt/plumber_agent"

# rm -rf $INSTALL_DIR || true
# 创建安装目录
# mkdir -p $INSTALL_DIR

rm -rf /etc/systemd/system/plumber-agent.service || true

systemctl stop plumber-agent || true

# 下载安装文件
curl -L -o $INSTALL_DIR/plumber-agent $DOWNLOAD_URL
curl -L -o /etc/systemd/system/plumber-agent.service $SYSTEMD_URL

# 赋予执行权限
chmod +x $INSTALL_DIR/plumber-agent

# 启动服务
systemctl daemon-reload

# 设置开机自启
systemctl enable plumber-agent

# 启动服务
systemctl start plumber-agent