FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o qianwen2api main.go

# 运行阶段
FROM chromedp/headless-shell:latest

# 安装 ca-certificates 和 dumb-init
RUN apt-get update && \
    apt-get install -y ca-certificates dumb-init && \
    rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN groupadd -r appuser && useradd -r -g appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/qianwen2api .

# 创建数据目录并设置权限
RUN mkdir -p /app/data && chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8000

# 使用 dumb-init 启动服务（避免僵尸进程）
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["./qianwen2api"]
