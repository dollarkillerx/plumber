package webssh

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/plumber/plumber/internal/server/storage"
	"github.com/plumber/plumber/pkg/models"
	"github.com/plumber/plumber/pkg/util"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域，生产环境需要限制
	},
}

type WebSSHHandler struct {
	storage       storage.Storage
	encryptionKey []byte
}

func NewWebSSHHandler(storage storage.Storage, encryptionKey []byte) *WebSSHHandler {
	return &WebSSHHandler{
		storage:       storage,
		encryptionKey: encryptionKey,
	}
}

type WebSocketMessage struct {
	Type string `json:"type"` // data, resize
	Data string `json:"data"`
	Rows int    `json:"rows,omitempty"`
	Cols int    `json:"cols,omitempty"`
}

func (h *WebSSHHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 从查询参数获取 agent_id
	agentIDStr := r.URL.Query().Get("agent_id")
	if agentIDStr == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		http.Error(w, "invalid agent_id", http.StatusBadRequest)
		return
	}

	// 获取 Agent 信息
	ctx := context.Background()
	agent, err := h.storage.GetAgent(ctx, agentID)
	if err != nil {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}

	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// 创建 SSH 连接
	sshConn, err := h.createSSHConnection(agent)
	if err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to connect: %v", err))
		return
	}
	defer sshConn.Close()

	// 创建 SSH 会话
	session, err := sshConn.NewSession()
	if err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to create session: %v", err))
		return
	}
	defer session.Close()

	// 设置终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to request pty: %v", err))
		return
	}

	// 获取标准输入输出
	stdin, err := session.StdinPipe()
	if err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to get stdin: %v", err))
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to get stdout: %v", err))
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to get stderr: %v", err))
		return
	}

	// 启动 shell
	if err := session.Shell(); err != nil {
		h.sendError(conn, fmt.Sprintf("Failed to start shell: %v", err))
		return
	}

	// WebSocket 读取 -> SSH 写入
	go func() {
		for {
			var msg WebSocketMessage
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("WebSocket read error: %v", err)
				return
			}

			switch msg.Type {
			case "data":
				if _, err := stdin.Write([]byte(msg.Data)); err != nil {
					log.Printf("SSH write error: %v", err)
					return
				}
			case "resize":
				if msg.Rows > 0 && msg.Cols > 0 {
					session.WindowChange(msg.Rows, msg.Cols)
				}
			}
		}
	}()

	// SSH 输出 -> WebSocket 发送
	go h.copyOutput(conn, stdout, "stdout")
	go h.copyOutput(conn, stderr, "stderr")

	// 等待会话结束
	if err := session.Wait(); err != nil {
		log.Printf("Session ended: %v", err)
	}
}

func (h *WebSSHHandler) createSSHConnection(agent *models.Agent) (*ssh.Client, error) {
	if agent.SSHUser == "" {
		return nil, fmt.Errorf("SSH user is required")
	}
	if agent.SSHHost == "" {
		return nil, fmt.Errorf("SSH host is required")
	}

	config := &ssh.ClientConfig{
		User:            agent.SSHUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
		Timeout:         10 * time.Second,
	}

	var authMethods []ssh.AuthMethod

	// 根据认证类型配置
	switch agent.SSHAuthType {
	case "password":
		if agent.SSHPassword == "" {
			return nil, fmt.Errorf("password is required for password authentication")
		}
		password := agent.SSHPassword
		// 如果密码是加密的，尝试解密
		if len(h.encryptionKey) == 32 {
			if decrypted, err := util.Decrypt(password, h.encryptionKey); err == nil {
				password = decrypted
			}
		}
		log.Printf("Using password authentication for user: %s", agent.SSHUser)
		authMethods = append(authMethods, ssh.Password(password))

	case "key":
		if agent.SSHPrivateKey == "" {
			return nil, fmt.Errorf("private key is required for key authentication")
		}
		privateKey := agent.SSHPrivateKey
		// 如果私钥是加密的，尝试解密
		if len(h.encryptionKey) == 32 {
			if decrypted, err := util.Decrypt(privateKey, h.encryptionKey); err == nil {
				privateKey = decrypted
			}
		}
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		log.Printf("Using key authentication for user: %s", agent.SSHUser)
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	default:
		return nil, fmt.Errorf("authentication type '%s' is not supported. Use 'password' or 'key'", agent.SSHAuthType)
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method configured")
	}

	config.Auth = authMethods

	addr := fmt.Sprintf("%s:%d", agent.SSHHost, agent.SSHPort)
	log.Printf("Connecting to SSH server: %s@%s", agent.SSHUser, addr)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed: %w", err)
	}

	log.Printf("SSH connection established successfully")
	return client, nil
}

func (h *WebSSHHandler) copyOutput(conn *websocket.Conn, reader io.Reader, source string) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error from %s: %v", source, err)
			}
			return
		}

		if n > 0 {
			msg := WebSocketMessage{
				Type: "data",
				Data: string(buf[:n]),
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

func (h *WebSSHHandler) sendError(conn *websocket.Conn, message string) {
	msg := WebSocketMessage{
		Type: "error",
		Data: message,
	}
	conn.WriteJSON(msg)
}
