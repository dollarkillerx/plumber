# Plumber 软件需求规格说明书（SRS）

## 1. 系统概要

**Plumber** 是一个基于 **C/S 架构** 的自动化任务编排与分发平台，旨在通过 Web 界面和命令行工具统一管理多台服务器上的任务执行，简化批量运维、部署、批处理等场景下的自动化操作。

系统包含以下组件：

* **Plumber Server**：核心控制中心，负责管理任务、接收 Agent 注册、分发指令、收集执行结果等。
* **Plumber Web**：Web 管理面板，提供可视化界面，用于服务器管理、任务编排、任务监控和日志查看。
* **Plumber Agent Server**：部署在目标服务器上的轻量级代理，负责接收和执行任务指令，并上报执行结果与心跳。
* **Plumber CLI**：命令行工具，支持从终端触发任务、查看任务状态、配置连接等。

---

## 2. 系统功能结构

```
Plumber
├─ Plumber Server（控制中心）
│  ├─ 管理 Agent 注册 & 心跳
│  ├─ 管理任务及执行流程
│  └─ 任务执行状态和日志存储
│
├─ Plumber Web（管理面板）
│  ├─ 服务器管理（SSH 部署 Agent）
│  ├─ 任务编排与配置（TOML）
│  ├─ 手动触发任务
│  └─ 实时查看任务执行日志
│
├─ Plumber Agent（执行节点）
│  ├─ 注册并维持心跳
│  ├─ 接收任务并执行
│  └─ 上传执行结果和日志
│
└─ Plumber CLI（命令行工具）
   ├─ 任务触发
   ├─ 查看任务列表
   └─ 设置连接配置
```

---

## 3. 主要功能说明

### 3.1 Plumber Web 功能

#### 3.1.1 服务器管理

* **添加服务器**：输入 SSH 信息后，系统自动连接远程服务器并执行后台预设的部署脚本安装 Agent。
* **服务器列表**：显示所有已注册 Agent 的状态（在线/离线、UUID、IP、最近心跳时间等）。

#### 3.1.2 Agent 注册与心跳

* Agent 安装完成后自动生成本地 UUID 并向 Plumber Server 注册。
* Agent 定期发送心跳包以维持连接状态，若断连则状态标记为"离线"。

#### 3.1.3 任务编排与配置

* 新建任务时采用 TOML 格式配置步骤，示例如下：

```toml
[[step]]
ServerID = "uuid-xxx"
Path     = "/opt/project"
CMD      = "sh deploy.sh"

[[step]]
ServerID = "uuid-yyy"
Path     = "/data/scripts"
CMD      = "python3 main.py"
```

* 每个任务可包含多个步骤，按顺序依次执行。
* 每个步骤返回码为 `0` 时继续执行下一步，否则任务中断。

#### 3.1.4 任务执行与日志

* 可通过 Web 页面点击按钮手动执行任务。
* 实时展示每个步骤的执行输出和状态（运行中/成功/失败）。
* 支持历史任务日志查询。

---

### 3.2 Plumber Agent 功能

* **注册与身份标识**：首次运行自动生成 UUID，并向 Plumber Server 注册。
* **心跳机制**：定期向 Server 报告状态。
* **任务执行**：

  * 接收执行指令，进入指定目录运行命令。
  * 执行日志实时回传给 Server。
  * 返回退出码（`echo $?`）用于判断下一步骤是否继续。

---

### 3.3 Plumber CLI 功能

#### 3.3.1 基础配置

```bash
plumber-cli set-config --url https://plumber.example.com --token <ACCESS_TOKEN>
```

#### 3.3.2 任务列表

```bash
plumber-cli task list
```

* 显示所有任务的 `ID` 和 `名称`。

#### 3.3.3 任务触发

```bash
plumber-cli task run <task_id>
```

* 执行指定任务。
* CLI 仅显示各步骤的执行结果（退出码），不显示详细日志。

---

## 4. 系统流程

### 4.1 Agent 注册流程

```
[Agent] → 生成 UUID → 注册 → [Server] 存储 Agent 信息 → 心跳维持
```

### 4.2 部署流程

```
[Web] 输入 SSH 信息 → 后台执行部署脚本 → 安装 Agent → 自动注册 Server
```

### 4.3 任务执行流程

```
[Web/CLI] 触发任务
   ↓
[Server] 按步骤发送执行指令
   ↓
[Agent] 执行命令 → 返回日志与退出码
   ↓
[Server] 判断是否继续下一步 → 更新状态
   ↓
[Web] 展示实时日志和执行状态
```

---

## 5. 日志与监控要求

* 每个步骤执行的标准输出、错误输出均需完整记录。
* 任务执行历史需可查询（含执行时间、状态、日志、耗时等）。
* 心跳异常时，系统需能标记 Agent 离线并提示。

---

## 6. 权限与安全

* Plumber Server 提供基于 `token` 的认证机制。
* 所有 CLI 和 Web 请求均需携带有效 `token`。
* Agent 与 Server 之间通信应使用jsonrpc
