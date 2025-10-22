# AI 智能商城客服系统 - 重建指南

## 项目概述
基于 RAG+MCP 的电商智能客服系统。Java 商城提供商品/订单 API，Go AI 引擎通过 RAG 检索知识库并调用 Python MCP 工具执行订单操作，支持多轮对话上下文记忆。

## 技术栈
- **Go 1.21** + Gin (AI 引擎)
- **Java 17** + Spring Boot 3.2 + H2 (商城服务)
- **Python 3.11** + FastMCP (MCP 工具服务器)
- **Chroma** (向量数据库)
- **阿里云 DashScope** (qwen-max LLM + text-embedding-v2)
- **Docker Compose** (部署编排)

## 目录结构
```
project/
├── go-ai-service/          # Go AI 引擎
│   ├── main.go             # 入口: 初始化 MCP/LLM/RAG, 启动 HTTP 服务
│   ├── handlers/
│   │   ├── chat_handler.go  # 核心: 处理聊天, RAG 检索, 工具调用
│   │   └── xml_parser.go    # 解析 LLM 输出的 XML 格式工具调用
│   ├── llm/dashscope_client.go  # DashScope API 客户端
│   ├── rag/chroma_client.go     # Chroma 向量检索
│   ├── mcp/
│   │   ├── client.go       # MCP STDIO 协议客户端
│   │   └── executor.go     # 工具执行调度器
│   └── config/config.go    # 环境变量配置
├── java-shop/              # Java 商城服务
│   ├── controller/         # REST API (商品/订单/聊天)
│   ├── service/            # 业务逻辑 + AI 代理
│   ├── model/              # Product/Order 实体
│   └── resources/
│       ├── application.yml # 配置
│       └── templates/index.html  # Web UI
├── mcp-server/
│   └── server.py           # MCP 工具: search_product, create_order, query_order, cancel_order
├── knowledge/
│   └── init_knowledge_rest.py  # 知识库初始化脚本
└── docker-compose.yml      # 服务编排
```

## 核心实现

### 1. RAG 模块 (Go)
**向量化**: DashScope `text-embedding-v2`, 1536 维
**检索策略**: Chroma 余弦相似度, Top-K=3
**关键逻辑**:
```go
// 1. 生成查询向量
embedding := dashscope.Embedding(query)

// 2. Chroma 查询 (REST API v2)
POST /api/v2/tenants/{tenant}/databases/{db}/collections/{id}/query
Body: { "query_embeddings": [[...]], "n_results": 3 }

// 3. 格式化上下文注入 LLM System Prompt
context := "以下是相关的知识库信息:\n\n1. 文档内容...\n2. ..."
```

### 2. MCP 集成 (Go ↔ Python)
**协议**: JSON-RPC 2.0 over STDIO
**通信流程**:
```go
// Go 启动 Python MCP Server 作为子进程
cmd := exec.Command("python3", "server.py")
stdin, stdout := cmd.StdinPipe(), cmd.StdoutPipe()

// 初始化握手
Send: {"jsonrpc":"2.0", "id":1, "method":"initialize", "params":{...}}
Recv: {"jsonrpc":"2.0", "id":1, "result":{...}}

// 调用工具
Send: {"jsonrpc":"2.0", "id":2, "method":"tools/call", "params":{"name":"create_order", "arguments":{...}}}
Recv: {"jsonrpc":"2.0", "id":2, "result":{"content":[{"type":"text", "text":"✅ 订单创建成功..."}]}}
```

**Python MCP Server** (FastMCP):
```python
from mcp.server.fastmcp import FastMCP
mcp = FastMCP("OrderManager")

@mcp.tool()
def create_order(productName: str, quantity: int, customerName: str, 
                customerPhone: str, shippingAddress: str) -> str:
    # 1. 搜索商品 ID: GET {JAVA_SHOP_URL}/api/products/search?keyword={productName}
    # 2. 创建订单: POST {JAVA_SHOP_URL}/api/orders
    # 3. 返回格式化结果
    return "✅ 订单创建成功！订单号: ORD-..."

mcp.run(transport='stdio')
```

### 3. 工具调用机制
**LLM 输出格式** (XML):
```xml
用户想要购买商品，我来帮您下单。
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
```

**解析逻辑** (Go):
```go
// 1. 正则提取 <func_call>...</func_call>
toolCall := extractToolCall(llmResponse)

// 2. 解析 XML 标签转 JSON
args := parseXMLToJSON(toolCall.arguments) // {"productName": "山地自行车", ...}

// 3. 通过 MCP Client 调用工具
result := mcpClient.CallTool(toolCall.toolName, args)

// 4. 移除 XML 标签, 拼接结果
finalReply := removeXML(llmResponse) + "\n\n" + result
```

### 4. 多轮对话 (历史管理)
**前端**: 维护最近 20 轮对话 (40 条消息), 每次请求携带
**后端**: 接收历史 → 拼接到 LLM messages
```go
messages := []Message{
    {Role: "system", Content: "你是智能客服..."},
    {Role: "system", Content: "以下是相关知识库信息..."}, // RAG 上下文
    {Role: "user", Content: "历史消息1"},
    {Role: "assistant", Content: "历史回复1"},
    ...
    {Role: "user", Content: "当前用户消息"},
}
```

### 5. Java 商城核心
**实体关系**:
```java
@Entity Product { id, name, price, stock, category, description }
@Entity Order { id, orderNumber, @ManyToOne product, quantity, totalPrice, 
                customerName, customerPhone, shippingAddress, status }
```

**REST API**:
- `GET /api/products/search?keyword={kw}` → 搜索商品
- `POST /api/orders` → 创建订单 (扣减库存)
- `GET /api/orders` → 查询所有订单
- `DELETE /api/orders/{orderNumber}` → 取消订单 (恢复库存)
- `POST /api/chat` → 转发到 Go AI 服务

**数据初始化**: H2 数据库启动时自动插入 5 个商品 (DataInitializer)

### 6. 知识库初始化
```python
# 知识数据: 15 条 (商品信息/FAQ/文档链接/品牌信息)
for item in knowledge_data:
    embedding = dashscope.embedding(item["text"])
    chroma.add(id=item["id"], document=item["text"], 
               embedding=embedding, metadata={"category": item["category"]})
```

**Chroma 操作** (REST API v2):
```bash
# 创建集合
POST /api/v2/tenants/{tenant}/databases/{db}/collections
Body: {"name": "shop_knowledge"}

# 添加文档
POST /api/v2/.../collections/{id}/add
Body: {"ids": [...], "documents": [...], "embeddings": [[...]], "metadatas": [...]}
```

## 配置

```bash
# 必需
DASHSCOPE_API_KEY=sk-xxx              # 阿里云 API Key

# 可选
CHROMA_HOST=chroma
CHROMA_PORT=8000
JAVA_SHOP_URL=http://java-shop:8080
GO_AI_SERVICE_URL=http://go-ai-service:8081
PORT=8081                              # Go 服务端口
MAX_CHAT_HISTORY_ROUNDS=20            # 最大历史轮数

# UTF-8 支持 (修复中文乱码)
LANG=C.UTF-8
LC_ALL=C.UTF-8
```

## 数据流
```
用户: "你们有山地自行车吗？"
  ↓
Java Shop → Go AI Service
  ↓
RAG 检索 Chroma → 找到商品知识
  ↓
LLM 生成回复: "🔍 找到 1 个商品：山地自行车 Pro X1, 价格 ¥3999"
  ↓
前端显示 + 保存历史

用户: "我要买 2 辆，张三，13800138000，北京朝阳区建国路1号"
  ↓
Go AI Service (携带历史上下文)
  ↓
LLM 理解意图 → 输出 XML 工具调用
  ↓
解析 XML → MCP Client 调用 create_order
  ↓
Python MCP → 搜索商品 → POST Java Shop 创建订单
  ↓
返回: "✅ 订单创建成功！订单号: ORD-1729512345..."
```

## 快速启动

### 1. 环境准备
```bash
# 创建 .env 文件
echo "DASHSCOPE_API_KEY=sk-your-api-key" > .env
```

### 2. 启动服务
```bash
docker-compose up -d --build

# 等待 30 秒后初始化知识库
sleep 30
docker-compose exec -T knowledge python /app/init_knowledge_rest.py
```

### 3. 验证
```bash
# 访问 Web UI
http://localhost:8080

# 测试对话
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "你们有什么自行车？"}'

# 预期响应
{"reply": "🔍 找到 3 个商品：\n\n1. 山地自行车 Pro X1, 价格 ¥3999..."}
```

## 关键依赖

**Go** (`go.mod`):
```go
github.com/gin-gonic/gin v1.9.1
github.com/gin-contrib/cors v1.5.0
```

**Java** (`pom.xml`):
```xml
spring-boot-starter-web:3.2.0
spring-boot-starter-data-jpa:3.2.0
h2database
lombok
```

**Python** (`requirements.txt`):
```
mcp>=1.0.0
requests>=2.31.0
```

## Dockerfile 要点

**Go AI Service**: 多阶段构建 (Go builder → Python runtime)
```dockerfile
FROM golang:1.21-alpine AS builder
RUN go build -o main .

FROM python:3.11-slim
COPY --from=builder /app/main .
COPY mcp-server ./mcp-server/
RUN pip install -r ./mcp-server/requirements.txt
CMD ["./main"]
```

**Java Shop**: Maven 构建
```dockerfile
FROM maven:3.9-eclipse-temurin-17 AS build
RUN mvn clean package -DskipTests

FROM eclipse-temurin:17-jre-alpine
COPY --from=build /app/target/*.jar app.jar
CMD ["java", "-jar", "/app/app.jar"]
```

## 常见问题

**Q: 中文乱码**  
A: 设置环境变量 `LANG=C.UTF-8`, `LC_ALL=C.UTF-8`

**Q: MCP 工具调用失败**  
A: 检查 Python 子进程是否启动成功 (查看 Go 日志 `[MCP Server]`)

**Q: RAG 检索无结果**  
A: 确认知识库已初始化 (Chroma 集合 `shop_knowledge` 存在)

**Q: LLM 不调用工具**  
A: System Prompt 中已明确工具调用格式 (XML), 确保参数完整

## 重建检查清单
- [ ] 阿里云 API Key 有效
- [ ] Docker Compose 网络互通 (ai-shop-network)
- [ ] Chroma 健康检查通过 (端口 8000)
- [ ] H2 数据持久化 (volume: shop-data)
- [ ] Go 子进程成功启动 Python MCP Server
- [ ] 知识库初始化成功 (15 条文档)
- [ ] 前端能正常展示商品列表
- [ ] 多轮对话上下文连贯
