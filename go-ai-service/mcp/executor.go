package mcp

import (
"encoding/json"
"fmt"
"log"
)

// ToolExecutor 工具执行器（通过 MCP Client）
type ToolExecutor struct {
javaShopURL string
}

// NewToolExecutor 创建新的工具执行器
func NewToolExecutor(javaShopURL string) *ToolExecutor {
return &ToolExecutor{
javaShopURL: javaShopURL,
}
}

// Execute 执行工具调用 - 通过 MCP Client
func (e *ToolExecutor) Execute(toolName string, arguments string) (string, error) {
log.Printf(" 执行工具: %s, 参数: %s", toolName, arguments)

// 使用 MCP Client 调用工具
mcpClient := GetMCPClient()
if mcpClient == nil {
return "", fmt.Errorf("MCP Client 未初始化")
}

// 解析参数
var args map[string]interface{}
if err := json.Unmarshal([]byte(arguments), &args); err != nil {
return "", fmt.Errorf("参数格式错误: %w", err)
}

// 调用 MCP 工具
result, err := mcpClient.CallTool(toolName, args)
if err != nil {
return "", fmt.Errorf("工具调用失败: %w", err)
}

log.Printf(" 工具执行成功")
return result, nil
}
