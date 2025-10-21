package mcp

import (
	"go-ai-service/llm"
)

// GetTools 获取所有工具定义
func GetTools() []llm.Tool {
	return []llm.Tool{
		{
			Type: "function",
			Function: &llm.Function{
				Name:        "create_order",
				Description: "创建新订单。当用户明确表达购买意图(如'我要买'、'帮我下单'、'购买')并提供了商品ID、数量、姓名、电话、收货地址等完整信息时,必须使用此工具创建订单。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"productId": map[string]interface{}{
							"type":        "integer",
							"description": "商品ID",
						},
						"quantity": map[string]interface{}{
							"type":        "integer",
							"description": "购买数量",
						},
						"customerName": map[string]interface{}{
							"type":        "string",
							"description": "客户姓名",
						},
						"customerPhone": map[string]interface{}{
							"type":        "string",
							"description": "客户电话",
						},
						"shippingAddress": map[string]interface{}{
							"type":        "string",
							"description": "收货地址",
						},
					},
					"required": []string{"productId", "quantity", "customerName", "customerPhone", "shippingAddress"},
				},
			},
		},
		{
			Type: "function",
			Function: &llm.Function{
				Name:        "query_order",
				Description: "查询订单状态。当用户询问订单信息、订单状态、物流信息时使用此工具。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"orderNumber": map[string]interface{}{
							"type":        "string",
							"description": "订单号,格式如 ORD-001",
						},
					},
					"required": []string{"orderNumber"},
				},
			},
		},
		{
			Type: "function",
			Function: &llm.Function{
				Name:        "cancel_order",
				Description: "取消订单。当用户明确表示要取消订单、退单、不想要了时使用此工具。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"orderNumber": map[string]interface{}{
							"type":        "string",
							"description": "要取消的订单号,格式如 ORD-001",
						},
					},
					"required": []string{"orderNumber"},
				},
			},
		},
	}
}
