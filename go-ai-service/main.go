package main

import (
	"go-ai-service/config"
	"go-ai-service/handlers"
	"go-ai-service/llm"
	"go-ai-service/mcp"
	"go-ai-service/rag"
	"io"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 设置日志输出编码为 UTF-8（修复中文乱码）
	log.SetOutput(io.Writer(os.Stdout))
	
	// 加载配置
	cfg := config.LoadConfig()

	// 🔌 初始化 MCP Client（启动 Python MCP Server）
	log.Println("🔌 初始化 MCP Client...")
	if err := mcp.InitMCPClient(); err != nil {
		log.Fatalf("❌ MCP Client 初始化失败: %v", err)
	}
	defer mcp.CloseMCPClient()

	// 初始化 LLM 客户端
	llmClient := llm.NewDashScopeClient(cfg.DashScopeAPIKey)

	// 初始化 RAG 客户端
	ragClient := rag.NewChromaClient(cfg.ChromaHost, cfg.ChromaPort, cfg.DashScopeAPIKey)

	// 初始化 MCP 工具执行器（现在使用 MCP Client 而不是直接 HTTP）
	toolExecutor := mcp.NewToolExecutor(cfg.JavaShopURL)

	// 初始化处理器
	chatHandler := handlers.NewChatHandler(llmClient, ragClient, toolExecutor)

	// 设置路由
	router := gin.Default()

	// CORS 配置
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 聊天接口
	router.POST("/chat", chatHandler.HandleChat)

	// 启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("🚀 Go AI 服务启动在端口 %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
