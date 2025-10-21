package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	collectionName             = "shop_knowledge"
	dashScopeEmbeddingAPI      = "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"
	embeddingModel             = "text-embedding-v2"
	defaultTopK                = 3
)

// ChromaClient Chroma 向量数据库客户端
type ChromaClient struct {
	baseURL      string
	apiKey       string
	httpClient   *http.Client
	tenant       string
	database     string
	collectionID string
}

// NewChromaClient 创建新的 Chroma 客户端
func NewChromaClient(host, port, apiKey string) *ChromaClient {
	return &ChromaClient{
		baseURL:    fmt.Sprintf("http://%s:%s", host, port),
		apiKey:     apiKey,
		httpClient: &http.Client{},
		tenant:     "default_tenant",
		database:   "default_database",
	}
}

// Document 文档结构
type Document struct {
	ID       string  `json:"id"`
	Text     string  `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
	Distance float64 `json:"distance"`
}

// SearchKnowledge 搜索知识库
func (c *ChromaClient) SearchKnowledge(query string, topK int) ([]Document, error) {
	if topK <= 0 {
		topK = defaultTopK
	}

	log.Printf("🔍 搜索知识库: %s (Top %d)", query, topK)

	// 初始化 collection ID（首次调用时）
	if c.collectionID == "" {
		if err := c.initializeCollection(); err != nil {
			return nil, fmt.Errorf("初始化集合失败: %w", err)
		}
	}

	// 1. 生成查询向量
	embedding, err := c.generateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("生成嵌入向量失败: %w", err)
	}

	// 2. 在 Chroma 中查询
	documents, err := c.queryChroma(embedding, topK)
	if err != nil {
		return nil, fmt.Errorf("查询 Chroma 失败: %w", err)
	}

	log.Printf("✅ 找到 %d 个相关文档", len(documents))

	return documents, nil
}

// generateEmbedding 使用 DashScope 生成嵌入向量
func (c *ChromaClient) generateEmbedding(text string) ([]float64, error) {
	// DashScope Embedding API 标准格式
	reqBody := map[string]interface{}{
		"model": embeddingModel,
		"input": map[string]interface{}{
			"texts": []string{text},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", dashScopeEmbeddingAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API 错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Output struct {
			Embeddings []struct {
				Embedding []float32 `json:"embedding"`
				TextIndex int       `json:"text_index"`
			} `json:"embeddings"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != "Success" && result.Code != "" {
		return nil, fmt.Errorf("embedding API 错误: %s - %s", result.Code, result.Message)
	}

	if len(result.Output.Embeddings) == 0 {
		return nil, fmt.Errorf("未返回嵌入向量")
	}

	// 转换 float32 数组为 float64
	embedding32 := result.Output.Embeddings[0].Embedding
	embedding := make([]float64, len(embedding32))
	for i, v := range embedding32 {
		embedding[i] = float64(v)
	}

	return embedding, nil
}

// initializeCollection 初始化集合 ID（从 Chroma v2 API 获取）
func (c *ChromaClient) initializeCollection() error {
	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections", c.baseURL, c.tenant, c.database)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("获取集合列表失败: %s", string(body))
	}

	var collections []map[string]interface{}
	if err := json.Unmarshal(body, &collections); err != nil {
		return err
	}

	// 查找 shop_knowledge 集合
	for _, col := range collections {
		if name, ok := col["name"].(string); ok && name == collectionName {
			if id, ok := col["id"].(string); ok {
				c.collectionID = id
				log.Printf("✅ 找到集合 '%s' (ID: %s)", collectionName, id)
				return nil
			}
		}
	}

	return fmt.Errorf("集合 '%s' 不存在", collectionName)
}

// queryChroma 在 Chroma v2 中查询（使用更新的 API）
func (c *ChromaClient) queryChroma(embedding []float64, topK int) ([]Document, error) {
	// 使用 Chroma v2 API 格式
	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/query", 
		c.baseURL, c.tenant, c.database, c.collectionID)

	reqBody := map[string]interface{}{
		"query_embeddings": [][]float64{embedding},
		"n_results":        topK,
		"include":          []string{"documents", "metadatas", "distances"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Chroma 查询错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		IDs       [][]string                   `json:"ids"`
		Documents [][]string                   `json:"documents"`
		Metadatas [][]map[string]interface{}   `json:"metadatas"`
		Distances [][]float64                  `json:"distances"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 解析结果
	var documents []Document
	if len(result.Documents) > 0 && len(result.Documents[0]) > 0 {
		for i := 0; i < len(result.Documents[0]); i++ {
			doc := Document{
				ID:   result.IDs[0][i],
				Text: result.Documents[0][i],
			}

			if len(result.Metadatas) > 0 && len(result.Metadatas[0]) > i {
				doc.Metadata = result.Metadatas[0][i]
			}

			if len(result.Distances) > 0 && len(result.Distances[0]) > i {
				doc.Distance = result.Distances[0][i]
			}

			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// FormatContext 格式化检索到的上下文
func FormatContext(documents []Document) string {
	if len(documents) == 0 {
		return ""
	}

	context := "以下是相关的知识库信息:\n\n"
	for i, doc := range documents {
		context += fmt.Sprintf("%d. %s\n", i+1, doc.Text)
		if category, ok := doc.Metadata["category"].(string); ok {
			context += fmt.Sprintf("   分类: %s\n", category)
		}
	}

	return context
}

// generateBatchEmbeddings 批量生成嵌入向量
func (c *ChromaClient) generateBatchEmbeddings(texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	// DashScope Embedding API 标准格式
	reqBody := map[string]interface{}{
		"model": embeddingModel,
		"input": map[string]interface{}{
			"texts": texts,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", dashScopeEmbeddingAPI, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API 错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Output struct {
			Embeddings []struct {
				Embedding []float32 `json:"embedding"`
				TextIndex int       `json:"text_index"`
			} `json:"embeddings"`
		} `json:"output"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Code != "Success" && result.Code != "" {
		return nil, fmt.Errorf("embedding API 错误: %s - %s", result.Code, result.Message)
	}

	// 转换结果，保持顺序
	embeddings := make([][]float64, len(texts))
	for _, emb := range result.Output.Embeddings {
		embedding64 := make([]float64, len(emb.Embedding))
		for i, v := range emb.Embedding {
			embedding64[i] = float64(v)
		}
		embeddings[emb.TextIndex] = embedding64
	}

	return embeddings, nil
}

// AddDocuments 添加文档到知识库（使用 Chroma v2 API）
func (c *ChromaClient) AddDocuments(docs []Document) error {
	if len(docs) == 0 {
		return nil
	}

	// 初始化 collection ID（首次调用时）
	if c.collectionID == "" {
		if err := c.initializeCollection(); err != nil {
			return fmt.Errorf("初始化集合失败: %w", err)
		}
	}

	// 生成嵌入向量
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Text
	}

	embeddings, err := c.generateBatchEmbeddings(texts)
	if err != nil {
		return fmt.Errorf("生成嵌入向量失败: %w", err)
	}

	// 准备 Chroma 请求
	ids := make([]string, len(docs))
	documents := make([]string, len(docs))
	metadatas := make([]map[string]interface{}, len(docs))

	for i, doc := range docs {
		ids[i] = doc.ID
		documents[i] = doc.Text
		metadatas[i] = doc.Metadata
	}

	// 使用 Chroma v2 API 格式
	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s/add", 
		c.baseURL, c.tenant, c.database, c.collectionID)

	reqBody := map[string]interface{}{
		"ids":         ids,
		"documents":   documents,
		"metadatas":   metadatas,
		"embeddings":  embeddings,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Chroma 添加文档错误 (状态码 %d): %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ 成功添加 %d 条文档到 Chroma", len(docs))
	return nil
}
