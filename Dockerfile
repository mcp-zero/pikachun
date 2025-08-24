# 使用官方Go镜像作为构建环境
FROM golang:1.24-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制go mod和sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pikachun .

# 使用轻量级的Alpine镜像作为运行环境
FROM alpine:latest

# 安装ca-certificates以支持HTTPS请求
RUN apk --no-cache add ca-certificates

# 创建非root用户
RUN adduser -D -s /bin/sh pikachun

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/pikachun .

# 复制web目录
COPY --from=builder /app/web ./web

# 复制默认配置文件
COPY --from=builder /app/config.yaml ./config.yaml

# 创建数据和日志目录
RUN mkdir -p ./data ./logs

# 更改文件所有者
RUN chown -R pikachun:pikachun ./pikachun ./web ./config.yaml ./data ./logs

# 切换到非root用户
USER pikachun

# 暴露端口
EXPOSE 8668

# 启动应用
CMD ["./pikachun"]