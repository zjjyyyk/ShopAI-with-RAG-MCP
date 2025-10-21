FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache wget
COPY go-ai-service/go.mod ./
RUN go mod download -x
COPY go-ai-service/ .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=mod -o main .

FROM python:3.11-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates wget tzdata \
    && rm -rf /var/lib/apt/lists/*
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8
ENV TZ=Asia/Shanghai
ENV PYTHONUNBUFFERED=1
ENV MCP_SERVER_PATH=/root/mcp-server/server.py
WORKDIR /root/
COPY --from=builder /app/main .
COPY mcp-server ./mcp-server/
RUN pip install --no-cache-dir -r ./mcp-server/requirements.txt
EXPOSE 8081
CMD ["./main"]
