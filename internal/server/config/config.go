package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config 配置结构
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Database DatabaseConfig `toml:"database"`
	Auth     AuthConfig     `toml:"auth"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host           string `toml:"host"`
	Port           string `toml:"port"`
	PublicAddr     string `toml:"public_addr"`     // 公网访问地址（用于生成agent配置）
	ExportEndpoint string `toml:"export_endpoint"` // Agent配置文件中的server_addr
	Debug          bool   `toml:"debug"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
	SSLMode  string `toml:"sslmode"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret       string `toml:"jwt_secret"`
	TokenExpiration int    `toml:"token_expiration"` // 小时
	AgentToken      string `toml:"agent_token"`      // Agent认证Token
	AdminUsername   string `toml:"admin_username"`   // 管理员用户名
	AdminPassword   string `toml:"admin_password"`   // 管理员密码
	EncryptionKey   string `toml:"encryption_key"`   // 数据加密密钥（32字节）
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	var config Config

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
