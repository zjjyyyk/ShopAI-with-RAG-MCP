package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// DashScopeClient 代表 DashScope/Qwen API 客户端
type DashScopeClient struct {
	apiKey string
	client *http.Client
}

// 请求和响应结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Type     string    `json:"type"`
	Function *Function `json:"function,omitempty"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ChatRequest struct {
	Model       string       `json:"model"`
	Messages    []Message    `json:"messages"`
	Tools       []Tool       `json:"tools,omitempty"`
	TopP        float64      `json:"top_p,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
}

type ChatResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Text         string `json:"text"`           // 🔧 直接的文本回复（qwen-max 使用这个格式）
		FinishReason string `json:"finish_reason"`
		Choices      []struct {                     // 保留以防某些模式使用
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type EmbeddingRequest struct {
	Model  string   `json:"model"`
	Input  []string `json:"input"`
	TextType string `json:"text_type,omitempty"`
}

type EmbeddingResponse struct {
	RequestID string `json:"request_id"`
	Output    struct {
		Embeddings []struct {
			Embedding []float32 `json:"embedding"`
			TextIndex int       `json:"text_index"`
		} `json:"embeddings"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewDashScopeClient 创建新的 DashScope 客户端
func NewDashScopeClient(apiKey string) *DashScopeClient {
	return &DashScopeClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// Chat 发送聊天请求并获取响应
func (c *DashScopeClient) Chat(messages []Message, tools []Tool) (*ChatResponse, error) {
	log.Printf("📨 调用 Qwen Chat API, 消息数: %d, 工具数: %d", len(messages), len(tools))
	
	// DashScope 格式：需要将请求包装在 input 对象中
	payload := map[string]interface{}{
		"model": "qwen-max",
		"input": map[string]interface{}{
			"messages": messages,
		},
		"parameters": map[string]interface{}{
			"temperature": 0.1,  // 降低随机性，更倾向于调用工具
			"top_p":       0.8,
		},
	}
	
	// ✅ 如果有工具，添加 tools 并设置 result_format（注意：result_format 必须在顶层！）
	if len(tools) > 0 {
		payload["tools"] = tools
		payload["result_format"] = "message"  // ✅ 顶层参数，不在 parameters 里
		log.Printf("🔧 启用工具调用模式, result_format=message")
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码请求失败: %v", err)
	}
	
	// 🔍 打印请求 payload 用于调试
	log.Printf("🔍 请求 Payload: %s", string(reqBody))

	httpReq, err := http.NewRequest("POST",
		"https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 🔍 打印原始响应用于调试
	log.Printf("🔍 API 原始响应: %s", string(body))

	// ✅ 添加 HTTP 状态码检查
	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ API 返回非 200 状态码: %d", resp.StatusCode)
		log.Printf("❌ 响应体: %s", string(body))
		return nil, fmt.Errorf("API 错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		log.Printf("❌ 解析 JSON 失败: %v", err)
		log.Printf("❌ 响应体: %s", string(body))
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// ✅ 添加详细日志
	log.Printf("✅ Qwen API 响应成功, RequestID: %s", chatResp.RequestID)
	
	// 🔍 添加调试日志 - 检查响应结构
	log.Printf("🔍🔍🔍 调试: Choices 数量 = %d", len(chatResp.Output.Choices))
	log.Printf("🔍🔍🔍 调试: Text = '%s'", chatResp.Output.Text)
	
	if len(chatResp.Output.Choices) > 0 {
		choice := chatResp.Output.Choices[0]
		log.Printf("🔍 finish_reason: %s", choice.FinishReason)
		log.Printf("🔍 message.content: %s", choice.Message.Content)
		log.Printf("🔍 tool_calls 数量: %d", len(choice.Message.ToolCalls))
		if len(choice.Message.ToolCalls) > 0 {
			for i, tc := range choice.Message.ToolCalls {
				log.Printf("🔍   工具 %d: %s, 参数: %s", i+1, tc.Function.Name, tc.Function.Arguments)
			}
		}
	}

	if chatResp.Code != "" && chatResp.Code != "Success" {
		log.Printf("❌ API 返回错误代码: %s - %s", chatResp.Code, chatResp.Message)
		return nil, fmt.Errorf("API 错误: %s - %s", chatResp.Code, chatResp.Message)
	}

	return &chatResp, nil
}

// Embedding 生成文本的嵌入向量
func (c *DashScopeClient) Embedding(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// DashScope 标准 Embedding API 格式
	payload := map[string]interface{}{
		"model": "text-embedding-v2",
		"input": map[string]interface{}{
			"texts": texts,
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码请求失败: %v", err)
	}

	httpReq, err := http.NewRequest("POST",
		"https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	err = json.Unmarshal(body, &embeddingResp)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if embeddingResp.Code != "" && embeddingResp.Code != "Success" {
		return nil, fmt.Errorf("API 错误: %s - %s", embeddingResp.Code, embeddingResp.Message)
	}

	// 提取嵌入向量，保持原始顺序
	embeddings := make([][]float32, len(texts))
	for _, emb := range embeddingResp.Output.Embeddings {
		if emb.TextIndex < len(embeddings) {
			embeddings[emb.TextIndex] = emb.Embedding
		}
	}

	return embeddings, nil
}

// GetTextResponse 从聊天响应中提取文本内容
func (c *DashScopeClient) GetTextResponse(resp interface{}) string {
	chatResp, ok := resp.(*ChatResponse)
	if !ok {
		log.Printf("⚠️  响应不是 ChatResponse 类型")
		return ""
	}
	
	// 🔧 优先使用 text 字段（qwen-max 格式）
	if chatResp.Output.Text != "" {
		return chatResp.Output.Text
	}
	
	// 兼容 choices 格式
	if len(chatResp.Output.Choices) == 0 {
		log.Printf("⚠️  响应中没有 text 也没有 choices")
		return ""
	}
	
	content := chatResp.Output.Choices[0].Message.Content
	if content == "" {
		log.Printf("⚠️  AI 响应内容为空, FinishReason: %s", chatResp.Output.Choices[0].FinishReason)
	}
	return content
}

// GetToolCalls 从聊天响应中提取工具调用
func (c *DashScopeClient) GetToolCalls(resp interface{}) []ToolCall {
	chatResp, ok := resp.(*ChatResponse)
	if !ok {
		return nil
	}
	
	// text 格式不支持工具调用
	if chatResp.Output.Text != "" {
		return nil
	}
	
	// choices 格式支持工具调用
	if len(chatResp.Output.Choices) == 0 {
		return nil
	}
	return chatResp.Output.Choices[0].Message.ToolCalls
}

// ShouldCallTool 判断是否应该调用工具
func (c *DashScopeClient) ShouldCallTool(resp interface{}) bool {
	chatResp, ok := resp.(*ChatResponse)
	if !ok {
		return false
	}
	
	// text 格式不支持工具调用
	if chatResp.Output.Text != "" {
		return false
	}
	
	// choices 格式检查工具调用
	if len(chatResp.Output.Choices) == 0 {
		return false
	}
	
	finishReason := chatResp.Output.Choices[0].FinishReason
	return strings.Contains(finishReason, "tool_calls")
}
