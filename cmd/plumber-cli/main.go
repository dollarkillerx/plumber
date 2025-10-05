package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/plumber/plumber/pkg/jsonrpc"
)

const (
	configFile = ".plumber_cli_config"
)

type Config struct {
	ServerURL string `json:"server_url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Token     string `json:"token"`
}

var (
	config Config
)

func main() {
	// 加载配置（首次使用时配置文件不存在是正常的）
	if err := loadConfig(); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: %v\n", err)
	}

	// 定义子命令
	setConfigCmd := flag.NewFlagSet("set-config", flag.ExitOnError)
	setConfigURL := setConfigCmd.String("url", "", "Plumber server URL")
	setConfigUser := setConfigCmd.String("user", "", "Username")
	setConfigPassword := setConfigCmd.String("password", "", "Password")

	taskCmd := flag.NewFlagSet("task", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "set-config":
		setConfigCmd.Parse(os.Args[2:])
		handleSetConfig(*setConfigURL, *setConfigUser, *setConfigPassword)

	case "task":
		if len(os.Args) < 3 {
			fmt.Println("Usage: plumber-cli task <list|run|info>")
			os.Exit(1)
		}

		taskCmd.Parse(os.Args[3:])
		switch os.Args[2] {
		case "list":
			handleTaskList()
		case "run":
			if len(os.Args) < 4 {
				fmt.Println("Usage: plumber-cli task run <task_id>")
				os.Exit(1)
			}
			handleTaskRun(os.Args[3])
		case "info":
			if len(os.Args) < 4 {
				fmt.Println("Usage: plumber-cli task info <task_id>")
				os.Exit(1)
			}
			handleTaskInfo(os.Args[3])
		default:
			fmt.Printf("Unknown task command: %s\n", os.Args[2])
			os.Exit(1)
		}

	case "agent":
		if len(os.Args) < 3 {
			fmt.Println("Usage: plumber-cli agent <list>")
			os.Exit(1)
		}

		switch os.Args[2] {
		case "list":
			handleAgentList()
		default:
			fmt.Printf("Unknown agent command: %s\n", os.Args[2])
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Plumber CLI - Task orchestration and distribution tool")
	fmt.Println("\nUsage:")
	fmt.Println("  plumber-cli set-config --url <server_url> --user <username> --password <password>")
	fmt.Println("  plumber-cli task list")
	fmt.Println("  plumber-cli task run <task_id>")
	fmt.Println("  plumber-cli task info <task_id>")
	fmt.Println("  plumber-cli agent list")
}

func handleSetConfig(url, username, password string) {
	// 设置 URL
	if url != "" {
		config.ServerURL = url
	}

	// 用户名和密码是必需的
	if username == "" || password == "" {
		fmt.Println("Error: Username and password are required")
		fmt.Println("Usage: plumber-cli set-config --url <server_url> --user <username> --password <password>")
		os.Exit(1)
	}

	// 保存用户名和密码（用于后续自动登录）
	config.Username = username
	config.Password = password

	// 验证配置并获取 token
	if config.ServerURL == "" {
		fmt.Println("Error: Server URL is required. Use --url to specify it.")
		os.Exit(1)
	}

	fmt.Printf("Logging in as %s...\n", username)
	obtainedToken, err := loginAndGetToken(config.ServerURL, username, password)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		os.Exit(1)
	}
	config.Token = obtainedToken
	fmt.Println("Login successful!")

	// 保存配置
	if err := saveConfig(); err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration saved successfully")
}

func loginAndGetToken(serverURL, username, password string) (string, error) {
	// 构建 URL，处理尾部斜杠
	if serverURL[len(serverURL)-1] == '/' {
		serverURL = serverURL[:len(serverURL)-1]
	}
	url := serverURL + "/api/rpc"

	// 构建登录请求
	loginParams := map[string]string{
		"username": username,
		"password": password,
	}
	paramsBytes, err := json.Marshal(loginParams)
	if err != nil {
		return "", err
	}

	req := jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "plumber.user.login",
		Params:  paramsBytes,
		ID:      1,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// 发送 HTTP 请求
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 解析响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp jsonrpc.Response
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("login error: %s", rpcResp.Error.Message)
	}

	// 提取 token
	resultBytes, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return "", err
	}

	var loginResult struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		UserID   string `json:"user_id"`
	}
	if err := json.Unmarshal(resultBytes, &loginResult); err != nil {
		return "", fmt.Errorf("failed to parse login result: %w", err)
	}

	if loginResult.Token == "" {
		return "", fmt.Errorf("no token returned from server")
	}

	return loginResult.Token, nil
}

func handleTaskList() {
	checkConfig()

	result, err := callRPC("plumber.task.list", nil)
	if err != nil {
		fmt.Printf("Failed to list tasks: %v\n", err)
		os.Exit(1)
	}

	var response struct {
		Tasks []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tDESCRIPTION")
	for _, task := range response.Tasks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.ID, task.Name, task.Status, task.Description)
	}
	w.Flush()
}

func handleTaskRun(taskID string) {
	checkConfig()

	params := map[string]string{
		"task_id": taskID,
	}

	fmt.Println("Starting task execution...")
	result, err := callRPC("plumber.task.run", params)
	if err != nil {
		fmt.Printf("Failed to run task: %v\n", err)
		os.Exit(1)
	}

	var response struct {
		Status      string `json:"status"`
		Message     string `json:"message"`
		ExecutionID string `json:"execution_id"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	var executionID string
	if response.ExecutionID == "" {
		fmt.Println("Task started, retrieving execution ID...")
		// 如果没有返回 execution_id，查询最新的 execution
		time.Sleep(2 * time.Second)
		execResult, err := callRPC("plumber.execution.list", map[string]string{
			"task_id": taskID,
		})
		if err != nil {
			fmt.Printf("Failed to get execution ID: %v\n", err)
			return
		}

		var execListResponse struct {
			Executions []struct {
				ID string `json:"id"`
			} `json:"executions"`
		}

		if err := json.Unmarshal(execResult, &execListResponse); err != nil || len(execListResponse.Executions) == 0 {
			fmt.Printf("Failed to retrieve execution ID\n")
			return
		}

		executionID = execListResponse.Executions[0].ID
	} else {
		executionID = response.ExecutionID
	}

	fmt.Printf("Task started, execution ID: %s\n", executionID)
	fmt.Println("Waiting for execution to complete...")
	fmt.Println(strings.Repeat("-", 80))

	// 轮询执行状态
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastStatus := ""
	shownSteps := make(map[string]bool)

	for {
		<-ticker.C

		execResult, err := callRPC("plumber.execution.get", map[string]string{
			"execution_id": executionID,
		})
		if err != nil {
			fmt.Printf("Failed to get execution status: %v\n", err)
			continue
		}

		var execResponse struct {
			Execution struct {
				ID        string    `json:"id"`
				TaskID    string    `json:"task_id"`
				Status    string    `json:"status"`
				StartTime *string   `json:"start_time"`
				EndTime   *string   `json:"end_time"`
				Steps     []struct {
					ID       string  `json:"id"`
					Command  string  `json:"command"`
					Path     string  `json:"path"`
					Status   string  `json:"status"`
					ExitCode *int    `json:"exit_code"`
					Output   string  `json:"output"`
				} `json:"steps"`
			} `json:"execution"`
		}

		if err := json.Unmarshal(execResult, &execResponse); err != nil {
			fmt.Printf("Failed to parse execution response: %v\n", err)
			continue
		}

		exec := execResponse.Execution

		// 显示状态变化
		if exec.Status != lastStatus {
			fmt.Printf("\nExecution status: %s\n", exec.Status)
			lastStatus = exec.Status
		}

		// 显示新完成的步骤
		for _, step := range exec.Steps {
			if !shownSteps[step.ID] && (step.Status == "success" || step.Status == "failed") {
				fmt.Printf("\n[Step] %s\n", step.Command)
				fmt.Printf("  Path: %s\n", step.Path)
				fmt.Printf("  Status: %s\n", step.Status)
				if step.ExitCode != nil {
					fmt.Printf("  Exit Code: %d\n", *step.ExitCode)
				}
				if step.Output != "" {
					fmt.Printf("  Output:\n%s\n", indentText(step.Output, "    "))
				}
				shownSteps[step.ID] = true
			}
		}

		// 检查是否完成
		if exec.Status == "success" || exec.Status == "failed" {
			fmt.Println(strings.Repeat("-", 80))
			fmt.Printf("\nTask execution completed with status: %s\n", exec.Status)
			if exec.StartTime != nil && exec.EndTime != nil {
				fmt.Printf("Duration: %s to %s\n", *exec.StartTime, *exec.EndTime)
			}

			// 退出码
			if exec.Status == "failed" {
				os.Exit(1)
			}
			return
		}
	}
}

func indentText(text, indent string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func handleTaskInfo(taskID string) {
	checkConfig()

	// 获取任务的最新执行记录
	result, err := callRPC("plumber.execution.list", map[string]string{
		"task_id": taskID,
	})
	if err != nil {
		fmt.Printf("Failed to get task executions: %v\n", err)
		os.Exit(1)
	}

	var response struct {
		Executions []struct {
			ID        string    `json:"id"`
			TaskID    string    `json:"task_id"`
			Status    string    `json:"status"`
			StartTime *string   `json:"start_time"`
			EndTime   *string   `json:"end_time"`
			Steps     []struct {
				ID       string  `json:"id"`
				Command  string  `json:"command"`
				Path     string  `json:"path"`
				Status   string  `json:"status"`
				ExitCode *int    `json:"exit_code"`
				Output   string  `json:"output"`
			} `json:"steps"`
		} `json:"executions"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	if len(response.Executions) == 0 {
		fmt.Println("No execution history found for this task")
		return
	}

	// 显示最新的执行记录
	exec := response.Executions[0]

	fmt.Println("=== Latest Execution ===")
	fmt.Printf("Execution ID: %s\n", exec.ID)
	fmt.Printf("Status: %s\n", exec.Status)
	if exec.StartTime != nil {
		fmt.Printf("Start Time: %s\n", *exec.StartTime)
	}
	if exec.EndTime != nil {
		fmt.Printf("End Time: %s\n", *exec.EndTime)
	}

	fmt.Println("\n=== Steps ===")
	for i, step := range exec.Steps {
		fmt.Printf("\n[Step %d] %s\n", i+1, step.Command)
		fmt.Printf("  Path: %s\n", step.Path)
		fmt.Printf("  Status: %s\n", step.Status)
		if step.ExitCode != nil {
			fmt.Printf("  Exit Code: %d\n", *step.ExitCode)
		}
		if step.Output != "" {
			fmt.Printf("  Output:\n%s\n", indentText(step.Output, "    "))
		}
	}
}

func handleAgentList() {
	checkConfig()

	result, err := callRPC("plumber.agent.list", nil)
	if err != nil {
		fmt.Printf("Failed to list agents: %v\n", err)
		os.Exit(1)
	}

	var response struct {
		Agents []struct {
			ID            string    `json:"id"`
			Hostname      string    `json:"hostname"`
			IP            string    `json:"ip"`
			Status        string    `json:"status"`
			LastHeartbeat time.Time `json:"last_heartbeat"`
		} `json:"agents"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tHOSTNAME\tIP\tSTATUS\tLAST HEARTBEAT")
	for _, agent := range response.Agents {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			agent.ID, agent.Hostname, agent.IP, agent.Status,
			agent.LastHeartbeat.Format("2006-01-02 15:04:05"))
	}
	w.Flush()
}

func callRPC(method string, params interface{}) (json.RawMessage, error) {
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsBytes,
		ID:      time.Now().UnixNano(),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// 构建 URL，处理尾部斜杠
	serverURL := config.ServerURL
	if serverURL[len(serverURL)-1] == '/' {
		serverURL = serverURL[:len(serverURL)-1]
	}
	url := serverURL + "/api/rpc"

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.Token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 先读取整个响应体用于调试
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp jsonrpc.Response
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	if rpcResp.Error != nil {
		// 如果是认证错误，尝试刷新 token 并重试
		if strings.Contains(rpcResp.Error.Message, "token") || strings.Contains(rpcResp.Error.Message, "expired") {
			fmt.Println("Token expired, refreshing...")
			refreshToken()
			// 重试请求
			return callRPC(method, params)
		}
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	result, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func loadConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &config)
}

func saveConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, configFile)
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func checkConfig() {
	if config.ServerURL == "" {
		fmt.Println("Error: Server URL not set. Please run 'plumber-cli set-config' first.")
		os.Exit(1)
	}

	if config.Username == "" || config.Password == "" {
		fmt.Println("Error: Username and password not set. Please run 'plumber-cli set-config' first.")
		os.Exit(1)
	}

	// 如果没有 token 或 token 可能已过期，重新登录
	if config.Token == "" {
		refreshToken()
	}
}

func refreshToken() {
	fmt.Printf("Logging in as %s...\n", config.Username)
	token, err := loginAndGetToken(config.ServerURL, config.Username, config.Password)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		fmt.Println("Please run 'plumber-cli set-config' to update your credentials.")
		os.Exit(1)
	}
	config.Token = token

	// 保存新的 token
	if err := saveConfig(); err != nil {
		fmt.Printf("Warning: Failed to save token: %v\n", err)
	}
}
