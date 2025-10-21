FROM python:3.11-slim

WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制 MCP 服务器代码
COPY server.py .

# 设置环境变量
ENV PYTHONUNBUFFERED=1

# 启动 MCP 服务器
CMD ["python", "server.py"]
