package handlers

import (
	"encoding/json"
	"fmt"
	"go-ai-service/llm"
	"go-ai-service/mcp"
	"go-ai-service/rag"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	llmClient    *llm.DashScopeClient
	ragClient    *rag.ChromaClient
	toolExecutor *mcp.ToolExecutor
}

// NewChatHandler 创建新的聊天处理器
func NewChatHandler(llmClient *llm.DashScopeClient, ragClient *rag.ChromaClient, toolExecutor *mcp.ToolExecutor) *ChatHandler {
	return &ChatHandler{
		llmClient:    llmClient,
		ragClient:    ragClient,
		toolExecutor: toolExecutor,
	}
}

// HistoryMessage 历史消息
type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Message   string           `json:"message" binding:"required"`
	UserID    string           `json:"userId"`
	SessionID string           `json:"sessionId"`
	History   []HistoryMessage `json:"history"` // 前端传递的历史消息
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Reply     string `json:"reply"`
	SessionID string `json:"sessionId"`
}

// HandleChat 处理聊天请求
func (h *ChatHandler) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	log.Printf("💬 收到消息 [%s]: %s", req.UserID, req.Message)

	// 1. RAG 检索 - 从知识库中搜索相关信息
	knowledgeDocs, err := h.ragClient.SearchKnowledge(req.Message, 3)
	if err != nil {
		log.Printf("⚠️  RAG 检索失败: %v", err)
		// 即使检索失败也继续处理
	}

	// 2. 构建消息历史
	messages := []llm.Message{
		{
			Role: "system",
			Content: `你是一个智能客服助手,负责帮助用户完成订单操作和解答问题。

你的能力:
1. 搜索商品 (search_product) - 当用户询问商品信息、价格、库存时
2. 创建订单 (create_order) - 当用户提供商品名称、数量、姓名、电话、地址时
3. 查询订单 (query_order) - 当用户询问订单状态时
4. 取消订单 (cancel_order) - 当用户要求取消订单时
5. 回答售后问题

⚠️ 工具调用格式规范:
当需要调用工具时,必须使用以下 XML 格式输出,参数名称必须精确匹配:

搜索商品示例:
<func_call>
<tool_name>search_product</tool_name>
<arguments>
<keyword>山地自行车</keyword>
</arguments>
</func_call>

创建订单示例:
<func_call>
<tool_name>create_order</tool_name>
<arguments>
<productName>山地自行车</productName>
<quantity>2</quantity>
<customerName>张三</customerName>
<customerPhone>13800138000</customerPhone>
<shippingAddress>北京市朝阳区建国路1号</shippingAddress>
</arguments>
</func_call>

查询订单示例:
<func_call>
<tool_name>query_order</tool_name>
<arguments>
<orderNumber>ORD-1234567890</orderNumber>
</arguments>
</func_call>

取消订单示例:
<func_call>
<tool_name>cancel_order</tool_name>
<arguments>
<orderNumber>ORD-1234567890</orderNumber>
</arguments>
</func_call>

重要:
- 必须严格按照上述 XML 格式输出
- 在 <func_call> 标签前后可以添加说明文字
- 如果信息不完整,先询问用户,不要调用工具`,
		},
	}

	// 如果有知识库检索结果,添加到上下文
	if len(knowledgeDocs) > 0 {
		contextMsg := llm.Message{
			Role:    "system",
			Content: rag.FormatContext(knowledgeDocs),
		}
		messages = append(messages, contextMsg)
		log.Printf("📚 添加知识库上下文,共 %d 个文档", len(knowledgeDocs))
	}

	// 添加历史消息（前端传来的，已经限制在5轮以内）
	if len(req.History) > 0 {
		log.Printf("📜 添加历史消息,共 %d 条", len(req.History))
		for i, histMsg := range req.History {
			// 跳过当前消息（前端会在 history 末尾包含当前消息）
			if histMsg.Content == req.Message && histMsg.Role == "user" {
				log.Printf("   跳过当前消息")
				continue
			}
			
			// 安全地截断内容用于日志
			content := histMsg.Content
			if len(content) > 50 {
				content = content[:50] + "..."
			}
			log.Printf("   [%d] %s: %s", i+1, histMsg.Role, content)
			
			messages = append(messages, llm.Message{
				Role:    histMsg.Role,
				Content: histMsg.Content,
			})
		}
	} else {
		log.Printf("⚠️  没有接收到历史消息")
	}

	// 添加当前用户消息
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: req.Message,
	})

	// 3. 调用 LLM（不再传递 tools 参数，使用 XML 格式）
	response, err := h.llmClient.Chat(messages, nil)
	if err != nil {
		log.Printf("❌ LLM 调用失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理失败,请稍后再试"})
		return
	}

	// 提取响应文本
	responseText := response.Output.Text
	log.Printf("🤖 LLM 原始响应: %s", responseText)

	// 4. 检查是否包含工具调用（XML 格式）
	if toolCall, found := h.parseToolCallFromXML(responseText); found {
		log.Printf("🔧 检测到工具调用: %s", toolCall.ToolName)
		
		// 执行工具
		result, err := h.toolExecutor.Execute(toolCall.ToolName, toolCall.Arguments)
		if err != nil {
			log.Printf("❌ 工具执行失败: %v", err)
			c.JSON(http.StatusOK, ChatResponse{
				Reply:     fmt.Sprintf("抱歉，订单处理失败: %v", err),
				SessionID: req.SessionID,
			})
			return
		}

		log.Printf("✅ 工具执行成功: %s", result)

		// 构建最终回复（包含工具执行结果）
		finalReply := h.buildFinalReply(responseText, result)
		
		c.JSON(http.StatusOK, ChatResponse{
			Reply:     finalReply,
			SessionID: req.SessionID,
		})
		return
	}

	// 5. 没有工具调用，直接返回 LLM 响应
	log.Printf("✅ 普通回复（无工具调用）")

	c.JSON(http.StatusOK, ChatResponse{
		Reply:     responseText,
		SessionID: req.SessionID,
	})
}

// chatWithToolCalling 支持工具调用的聊天
func (h *ChatHandler) chatWithToolCalling(messages []llm.Message, tools []llm.Tool) (string, error) {
	maxIterations := 5 // 最多允许 5 轮工具调用
	currentMessages := messages

	for i := 0; i < maxIterations; i++ {
		// 调用 LLM
		response, err := h.llmClient.Chat(currentMessages, tools)
		if err != nil {
			return "", err
		}

		// 检查是否需要调用工具
		if h.llmClient.ShouldCallTool(response) {
			toolCalls := h.llmClient.GetToolCalls(response)
			log.Printf("🔧 LLM 请求调用 %d 个工具", len(toolCalls))

			// 添加 assistant 消息
			assistantMsg := llm.Message{
				Role:    "assistant",
				Content: "",
			}
			currentMessages = append(currentMessages, assistantMsg)

			// 执行所有工具调用
			for _, toolCall := range toolCalls {
				log.Printf("   - 工具: %s", toolCall.Function.Name)

				// 执行工具
				result, err := h.toolExecutor.Execute(toolCall.Function.Name, toolCall.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf("工具执行失败: %v", err)
					log.Printf("❌ 工具执行失败: %v", err)
				}

				// 添加工具结果到消息历史
				toolResultMsg := llm.Message{
					Role:    "tool",
					Content: result,
				}

				// 如果工具结果是 JSON,尝试美化
				if json.Valid([]byte(result)) {
					var prettyJSON map[string]interface{}
					if err := json.Unmarshal([]byte(result), &prettyJSON); err == nil {
						prettyBytes, _ := json.MarshalIndent(prettyJSON, "", "  ")
						toolResultMsg.Content = string(prettyBytes)
					}
				}

				currentMessages = append(currentMessages, toolResultMsg)
			}

			// 继续下一轮对话
			continue
		}

		// 没有工具调用,返回最终回复
		return h.llmClient.GetTextResponse(response), nil
	}

	return "抱歉,处理您的请求时遇到了问题,请稍后再试。", nil
}

// handleOrderIntent 处理订单相关的用户意图
func (h *ChatHandler) handleOrderIntent(message string) (string, bool) {
	// 简单的关键词匹配识别订单操作意图
	
	// 1. 检查是否是创建订单意图
	if strings.Contains(message, "下单") || strings.Contains(message, "购买") || strings.Contains(message, "买") {
		// 尝试从消息中提取订单信息
		orderInfo := h.extractOrderInfo(message)
		if orderInfo != nil {
			// 调用 create_order 工具
			args, _ := json.Marshal(orderInfo)
			result, err := h.toolExecutor.Execute("create_order", string(args))
			if err != nil {
				return fmt.Sprintf("订单创建失败：%v。请访问网站直接下单。", err), true
			}
			return result, true
		}
		return "我理解您想要下单，但订单信息不完整。请提供：商品ID、数量、姓名、电话、地址。或者您可以访问网站直接下单。", true
	}
	
	// 2. 检查是否是查询订单意图
	if strings.Contains(message, "查询订单") || strings.Contains(message, "订单状态") {
		// 提取订单号
		orderNumber := h.extractOrderNumber(message)
		if orderNumber != "" {
			args, _ := json.Marshal(map[string]string{"orderNumber": orderNumber})
			result, err := h.toolExecutor.Execute("query_order", string(args))
			if err != nil {
				return fmt.Sprintf("订单查询失败：%v", err), true
			}
			return result, true
		}
		return "请提供订单号，格式如：ORD-1729512345", true
	}
	
	// 3. 检查是否是取消订单意图
	if strings.Contains(message, "取消订单") || strings.Contains(message, "退单") {
		orderNumber := h.extractOrderNumber(message)
		if orderNumber != "" {
			args, _ := json.Marshal(map[string]string{"orderNumber": orderNumber})
			result, err := h.toolExecutor.Execute("cancel_order", string(args))
			if err != nil {
				return fmt.Sprintf("订单取消失败：%v", err), true
			}
			return result, true
		}
		return "请提供要取消的订单号，格式如：ORD-1729512345", true
	}
	
	return "", false // 不是订单意图
}

// extractOrderInfo 从消息中提取订单信息
func (h *ChatHandler) extractOrderInfo(message string) map[string]interface{} {
	// 使用正则表达式提取订单信息
	// 格式示例："下单：商品ID=1，数量1，鹿城，13800138000，北京朝阳区建国路1号"
	
	var productID int
	var quantity int
	var name, phone, address string
	
	// 提取商品ID
	if matched := regexp.MustCompile(`商品ID[=是:：\s]*(\d+)`).FindStringSubmatch(message); len(matched) > 1 {
		productID, _ = strconv.Atoi(matched[1])
	} else if matched := regexp.MustCompile(`productId[=:]\s*(\d+)`).FindStringSubmatch(message); len(matched) > 1 {
		productID, _ = strconv.Atoi(matched[1])
	}
	
	// 提取数量
	if matched := regexp.MustCompile(`数量[=是:：\s]*(\d+)`).FindStringSubmatch(message); len(matched) > 1 {
		quantity, _ = strconv.Atoi(matched[1])
	} else if matched := regexp.MustCompile(`quantity[=:]\s*(\d+)`).FindStringSubmatch(message); len(matched) > 1 {
		quantity, _ = strconv.Atoi(matched[1])
	}
	
	// 提取姓名（简单规则：2-4个汉字）
	if matched := regexp.MustCompile(`[姓名客户收货人][=是:：\s]*([\\p{Han}]{2,4})`).FindStringSubmatch(message); len(matched) > 1 {
		name = matched[1]
	} else if matched := regexp.MustCompile(`customerName[=:]\s*([\\p{Han}]+)`).FindStringSubmatch(message); len(matched) > 1 {
		name = matched[1]
	} else {
		// 尝试找到独立的中文名字
		if matched := regexp.MustCompile(`[，,]\s*([\\p{Han}]{2,4})[，,]`).FindStringSubmatch(message); len(matched) > 1 {
			name = matched[1]
		}
	}
	
	// 提取电话（11位数字）
	if matched := regexp.MustCompile(`1[3-9]\d{9}`).FindStringSubmatch(message); len(matched) > 0 {
		phone = matched[0]
	}
	
	// 提取地址（包含"市"、"区"、"路"等关键字的文本）
	if matched := regexp.MustCompile(`[地址配送收货][=是:：\s]*(.+?)(?:[，,。]|$)`).FindStringSubmatch(message); len(matched) > 1 {
		address = matched[1]
	} else if matched := regexp.MustCompile(`([\\p{Han}]+[市区县][\\p{Han}]+[路街道号]\d*号?[\\p{Han}\\d]*)`).FindStringSubmatch(message); len(matched) > 0 {
		address = matched[0]
	}
	
	// 验证是否所有必需信息都有
	if productID > 0 && quantity > 0 && name != "" && phone != "" && address != "" {
		return map[string]interface{}{
			"productId":       productID,
			"quantity":        quantity,
			"customerName":    name,
			"customerPhone":   phone,
			"shippingAddress": address,
		}
	}
	
	return nil
}

// extractOrderNumber 从消息中提取订单号
func (h *ChatHandler) extractOrderNumber(message string) string {
	// 匹配 ORD-开头的订单号
	if matched := regexp.MustCompile(`ORD-\d+`).FindStringSubmatch(message); len(matched) > 0 {
		return matched[0]
	}
	return ""
}