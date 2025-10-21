package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	ToolName  string
	Arguments string // JSON 格式的参数
}

// parseToolCallFromXML 从 LLM 响应中解析 XML 格式的工具调用
func (h *ChatHandler) parseToolCallFromXML(response string) (ToolCallInfo, bool) {
	// 检查是否包含 <func_call> 标签
	if !strings.Contains(response, "<func_call>") {
		return ToolCallInfo{}, false
	}

	log.Printf("🔍 检测到 <func_call> 标签，开始解析...")

	// 提取 <func_call>...</func_call> 之间的内容
	funcCallRegex := regexp.MustCompile(`<func_call>([\s\S]*?)</func_call>`)
	matches := funcCallRegex.FindStringSubmatch(response)
	if len(matches) < 2 {
		log.Printf("⚠️  无法提取 <func_call> 内容")
		return ToolCallInfo{}, false
	}

	funcCallContent := matches[1]
	log.Printf("📦 提取的内容: %s", funcCallContent)

	// 提取 tool_name
	toolNameRegex := regexp.MustCompile(`<tool_name>(.*?)</tool_name>`)
	toolNameMatches := toolNameRegex.FindStringSubmatch(funcCallContent)
	if len(toolNameMatches) < 2 {
		log.Printf("⚠️  无法提取 tool_name")
		return ToolCallInfo{}, false
	}
	toolName := strings.TrimSpace(toolNameMatches[1])

	// 提取 <arguments>...</arguments> 之间的内容
	argsRegex := regexp.MustCompile(`<arguments>([\s\S]*?)</arguments>`)
	argsMatches := argsRegex.FindStringSubmatch(funcCallContent)
	if len(argsMatches) < 2 {
		log.Printf("⚠️  无法提取 arguments")
		return ToolCallInfo{}, false
	}
	argsContent := argsMatches[1]

	// 解析 arguments 中的 XML 标签，转换为 JSON
	args := make(map[string]interface{})

	// 通用 XML 标签提取器（Go 不支持反向引用，需要手动匹配）
	// 匹配格式: <key>value</key>
	tagRegex := regexp.MustCompile(`<(\w+)>([^<]*)</(\w+)>`)
	tagMatches := tagRegex.FindAllStringSubmatch(argsContent, -1)

	for _, match := range tagMatches {
		if len(match) >= 4 {
			openTag := match[1]
			value := strings.TrimSpace(match[2])
			closeTag := match[3]

			// 确保开闭标签一致
			if openTag != closeTag {
				continue
			}

			// 特殊处理：电话号码和订单号应该是字符串，不要转换为数字
			if openTag == "customerPhone" || openTag == "orderId" {
				args[openTag] = value
				continue
			}

			// 尝试转换为数字
			if intValue, err := strconv.Atoi(value); err == nil {
				args[openTag] = intValue
			} else {
				args[openTag] = value
			}
		}
	}

	// 转换为 JSON 字符串
	argsJSON, err := json.Marshal(args)
	if err != nil {
		log.Printf("❌ 参数序列化失败: %v", err)
		return ToolCallInfo{}, false
	}

	log.Printf("✅ 解析成功 - 工具: %s, 参数: %s", toolName, string(argsJSON))

	return ToolCallInfo{
		ToolName:  toolName,
		Arguments: string(argsJSON),
	}, true
}

// buildFinalReply 构建最终回复（移除 XML 标签，添加工具执行结果）
func (h *ChatHandler) buildFinalReply(llmResponse string, toolResult string) string {
	// 移除 <func_call>...</func_call> 标签
	funcCallRegex := regexp.MustCompile(`<func_call>[\s\S]*?</func_call>`)
	cleanResponse := funcCallRegex.ReplaceAllString(llmResponse, "")

	// 清理多余的空行
	cleanResponse = strings.TrimSpace(cleanResponse)

	// 如果 LLM 响应为空，只返回工具结果
	if cleanResponse == "" {
		return toolResult
	}

	// 组合 LLM 响应和工具结果
	return fmt.Sprintf("%s\n\n%s", cleanResponse, toolResult)
}
