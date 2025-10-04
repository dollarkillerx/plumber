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
	"text/tabwriter"
	"time"

	"github.com/plumber/plumber/pkg/jsonrpc"
)

const (
	configFile = ".plumber_cli_config"
)

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
}

var (
	config Config
)

func main() {
	// 加载配置
	if err := loadConfig(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// 定义子命令
	setConfigCmd := flag.NewFlagSet("set-config", flag.ExitOnError)
	setConfigURL := setConfigCmd.String("url", "", "Plumber server URL")
	setConfigToken := setConfigCmd.String("token", "", "Access token")

	taskCmd := flag.NewFlagSet("task", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "set-config":
		setConfigCmd.Parse(os.Args[2:])
		handleSetConfig(*setConfigURL, *setConfigToken)

	case "task":
		if len(os.Args) < 3 {
			fmt.Println("Usage: plumber-cli task <list|run|create>")
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
		case "create":
			handleTaskCreate()
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
	fmt.Println("  plumber-cli set-config --url <server_url> --token <token>")
	fmt.Println("  plumber-cli task list")
	fmt.Println("  plumber-cli task run <task_id>")
	fmt.Println("  plumber-cli task create")
	fmt.Println("  plumber-cli agent list")
}

func handleSetConfig(url, token string) {
	if url != "" {
		config.ServerURL = url
	}
	if token != "" {
		config.Token = token
	}

	if err := saveConfig(); err != nil {
		fmt.Printf("Failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Configuration saved successfully")
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

	result, err := callRPC("plumber.task.run", params)
	if err != nil {
		fmt.Printf("Failed to run task: %v\n", err)
		os.Exit(1)
	}

	var response struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task execution %s: %s\n", response.Status, response.Message)
}

func handleTaskCreate() {
	checkConfig()

	fmt.Println("Enter task details:")

	var name, description, config string
	fmt.Print("Name: ")
	fmt.Scanln(&name)
	fmt.Print("Description: ")
	fmt.Scanln(&description)
	fmt.Println("Config (TOML, press Ctrl+D when done):")

	configBytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Printf("Failed to read config: %v\n", err)
		os.Exit(1)
	}
	config = string(configBytes)

	params := map[string]string{
		"name":        name,
		"description": description,
		"config":      config,
	}

	result, err := callRPC("plumber.task.create", params)
	if err != nil {
		fmt.Printf("Failed to create task: %v\n", err)
		os.Exit(1)
	}

	var response struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}

	if err := json.Unmarshal(result, &response); err != nil {
		fmt.Printf("Failed to parse response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Task created successfully. ID: %s\n", response.TaskID)
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
		ID:      "cli",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", config.ServerURL+"/rpc", bytes.NewBuffer(body))
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

	var rpcResp jsonrpc.Response
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, err
	}

	if rpcResp.Error != nil {
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
	if config.ServerURL == "" || config.Token == "" {
		fmt.Println("Error: Configuration not set. Please run 'plumber-cli set-config' first.")
		os.Exit(1)
	}
}
