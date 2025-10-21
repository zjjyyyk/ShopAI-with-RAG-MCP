package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
)

// MCPClient MCP 客户端 - 通过 stdio 与 Python MCP Server 通信
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	mu     sync.Mutex
	msgID  int
}

// MCPRequest MCP 请求格式
type MCPRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse MCP 响应格式
type MCPResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError MCP 错误格式
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPToolResult 工具调用结果
type MCPToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// NewMCPClient 创建并启动 MCP 客户端
func NewMCPClient(mcpServerPath string) (*MCPClient, error) {
	log.Printf("🔌 启动 MCP Server: python3 %s", mcpServerPath)

	// 启动 Python MCP Server
	cmd := exec.Command("python3", mcpServerPath)

	// 获取 stdin/stdout/stderr 管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdin 管道失败: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdout 管道失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stderr 管道失败: %w", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 MCP Server 失败: %w", err)
	}

	client := &MCPClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		msgID:  0,
	}

	// 启动 stderr 日志输出
	go client.logStderr()

	// 初始化会话
	if err := client.initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("初始化 MCP 会话失败: %w", err)
	}

	log.Println("✅ MCP Client 初始化成功")
	return client, nil
}

// logStderr 输出 MCP Server 的 stderr 日志
func (c *MCPClient) logStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		log.Printf("[MCP Server] %s", scanner.Text())
	}
}

// initialize 初始化 MCP 会话
func (c *MCPClient) initialize() error {
	req := MCPRequest{
		Jsonrpc: "2.0",
		ID:      c.nextID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]string{
				"name":    "go-ai-service",
				"version": "1.0.0",
			},
		},
	}

	var resp MCPResponse
	if err := c.sendRequest(req, &resp); err != nil {
		return err
	}

	if resp.Error != nil {
		return fmt.Errorf("MCP 初始化错误: %s", resp.Error.Message)
	}

	return nil
}

// ListTools 列出所有可用工具
func (c *MCPClient) ListTools() ([]string, error) {
	req := MCPRequest{
		Jsonrpc: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
	}

	var resp MCPResponse
	if err := c.sendRequest(req, &resp); err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("列出工具失败: %s", resp.Error.Message)
	}

	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	return toolNames, nil
}

// CallTool 调用 MCP 工具
func (c *MCPClient) CallTool(toolName string, arguments map[string]interface{}) (string, error) {
	req := MCPRequest{
		Jsonrpc: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	var resp MCPResponse
	if err := c.sendRequest(req, &resp); err != nil {
		return "", err
	}

	if resp.Error != nil {
		return "", fmt.Errorf("工具调用失败: %s", resp.Error.Message)
	}

	// 解析工具结果
	var toolResult MCPToolResult
	if err := json.Unmarshal(resp.Result, &toolResult); err != nil {
		return "", fmt.Errorf("解析工具结果失败: %w", err)
	}

	// 返回文本内容
	if len(toolResult.Content) > 0 {
		return toolResult.Content[0].Text, nil
	}

	return "", fmt.Errorf("工具返回空结果")
}

// sendRequest 发送请求并接收响应
func (c *MCPClient) sendRequest(req MCPRequest, resp *MCPResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 序列化请求
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求（以换行符结尾）
	if _, err := c.stdin.Write(append(reqJSON, '\n')); err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}

	// 读取响应
	reader := bufio.NewReader(c.stdout)
	respLine, err := reader.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析响应
	if err := json.Unmarshal(respLine, resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	return nil
}

// nextID 生成下一个消息 ID
func (c *MCPClient) nextID() int {
	c.msgID++
	return c.msgID
}

// Close 关闭 MCP 客户端
func (c *MCPClient) Close() error {
	log.Println("🔌 关闭 MCP Client...")

	// 关闭 stdin（通知 server 退出）
	if c.stdin != nil {
		c.stdin.Close()
	}

	// 等待进程结束
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Wait(); err != nil {
			log.Printf("⚠️  MCP Server 退出异常: %v", err)
		}
	}

	return nil
}

// 启动 MCP Client（全局单例）
var globalMCPClient *MCPClient

// InitMCPClient 初始化全局 MCP 客户端
func InitMCPClient() error {
	// 确定 MCP Server 路径
	mcpServerPath := os.Getenv("MCP_SERVER_PATH")
	if mcpServerPath == "" {
		mcpServerPath = "/root/mcp-server/server.py"
	}

	client, err := NewMCPClient(mcpServerPath)
	if err != nil {
		return err
	}

	globalMCPClient = client

	// 列出可用工具
	tools, err := client.ListTools()
	if err != nil {
		log.Printf("⚠️  无法列出 MCP 工具: %v", err)
	} else {
		log.Printf("📋 MCP 可用工具: %v", tools)
	}

	return nil
}

// GetMCPClient 获取全局 MCP 客户端
func GetMCPClient() *MCPClient {
	return globalMCPClient
}

// CloseMCPClient 关闭全局 MCP 客户端
func CloseMCPClient() {
	if globalMCPClient != nil {
		globalMCPClient.Close()
	}
}
