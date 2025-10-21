FROM python:3.11-slim

# 设置 UTF-8 编码
ENV PYTHONIOENCODING=utf-8 \
    LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

WORKDIR /app

# 安装依赖
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 复制所有初始化脚本
COPY init_knowledge*.py .

# 保持容器运行
CMD ["tail", "-f", "/dev/null"]
