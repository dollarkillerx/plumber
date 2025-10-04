package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Agent 代理服务器信息
type Agent struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name          string         `gorm:"size:255;not null" json:"name"`           // Agent名称
	SSHHost       string         `gorm:"size:255" json:"ssh_host,omitempty"`      // SSH主机地址
	SSHPort       int            `gorm:"default:22" json:"ssh_port,omitempty"`    // SSH端口
	SSHUser       string         `gorm:"size:100" json:"ssh_user,omitempty"`      // SSH用户名
	SSHAuthType   string         `gorm:"size:20;default:'none'" json:"ssh_auth_type,omitempty"` // 认证类型：password/key/none
	SSHPassword   string         `gorm:"size:255" json:"ssh_password,omitempty"`  // SSH密码（明文存储）
	SSHPrivateKey string         `gorm:"type:text" json:"ssh_private_key,omitempty"` // SSH私钥（明文存储）
	Hostname      string         `gorm:"size:255" json:"hostname,omitempty"`      // 实际主机名（Agent上报）
	IP            string         `gorm:"size:50" json:"ip,omitempty"`             // 实际IP（Agent上报）
	Status        string         `gorm:"size:20;not null;default:'offline'" json:"status"` // online/offline
	LastHeartbeat *time.Time     `json:"last_heartbeat,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Task 任务定义
type Task struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Config      string         `gorm:"type:text;not null" json:"config"` // TOML配置
	Status      string         `gorm:"size:20;not null;default:'pending'" json:"status"` // pending/running/success/failed
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Executions  []TaskExecution `gorm:"foreignKey:TaskID" json:"executions,omitempty"`
}

// TaskExecution 任务执行记录
type TaskExecution struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TaskID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"task_id"`
	Status      string         `gorm:"size:20;not null;default:'pending'" json:"status"` // pending/running/success/failed
	StartTime   *time.Time     `json:"start_time,omitempty"`
	EndTime     *time.Time     `json:"end_time,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Steps       []StepExecution `gorm:"foreignKey:ExecutionID" json:"steps,omitempty"`
}

// StepExecution 步骤执行记录
type StepExecution struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ExecutionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"execution_id"`
	StepIndex   int            `gorm:"not null" json:"step_index"`
	AgentID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"agent_id"`
	Path        string         `gorm:"size:500" json:"path"`
	Command     string         `gorm:"type:text;not null" json:"command"`
	Status      string         `gorm:"size:20;not null;default:'pending'" json:"status"` // pending/running/success/failed
	Assigned    bool           `gorm:"default:false;index" json:"assigned"` // 是否已分配给agent
	ExitCode    *int           `json:"exit_code,omitempty"`
	Output      string         `gorm:"type:text" json:"output,omitempty"`
	StartTime   *time.Time     `json:"start_time,omitempty"`
	EndTime     *time.Time     `json:"end_time,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// User 用户表（用于认证）
type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username  string         `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"size:255;not null" json:"-"`
	Token     string         `gorm:"size:500;index" json:"token,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TaskConfig TOML任务配置
type TaskConfig struct {
	Steps []TaskStep `toml:"step"`
}

// TaskStep 任务步骤
type TaskStep struct {
	ServerID string `toml:"ServerID" json:"server_id"`
	Path     string `toml:"Path" json:"path"`
	CMD      string `toml:"CMD" json:"cmd"`
}
