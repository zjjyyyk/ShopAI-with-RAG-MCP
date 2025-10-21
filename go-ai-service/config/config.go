package config

import (
	"log"
	"os"
)

// Config 应用配置
type Config struct {
	DashScopeAPIKey string
	ChromaHost      string
	ChromaPort      string
	JavaShopURL     string
	Port            string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		log.Fatal("错误: 必须设置 DASHSCOPE_API_KEY 环境变量")
	}

	cfg := &Config{
		DashScopeAPIKey: apiKey,
		ChromaHost:      getEnv("CHROMA_HOST", "localhost"),
		ChromaPort:      getEnv("CHROMA_PORT", "8000"),
		JavaShopURL:     getEnv("JAVA_SHOP_URL", "http://localhost:8080"),
		Port:            getEnv("PORT", "8081"),
	}

	log.Printf("✅ 配置加载完成")
	log.Printf("   - Chroma: %s:%s", cfg.ChromaHost, cfg.ChromaPort)
	log.Printf("   - Java Shop: %s", cfg.JavaShopURL)

	return cfg
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
