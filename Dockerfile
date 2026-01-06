# 多阶段构建 Dockerfile

# 阶段 1: 前端构建
FROM node:18-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN npm install -g pnpm && pnpm install
COPY web/ ./
RUN pnpm run build

# 阶段 2: Go 编译
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
# 安装构建依赖
RUN apk add --no-cache git
# 复制 go mod 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download
# 复制源代码
COPY . .
# 复制前端构建产物
COPY --from=frontend-builder /app/web/dist ./web/dist
# 编译 Go 二进制文件（静态链接）
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mailbox .

# 阶段 3: 最终镜像
FROM alpine:latest
# 安装 CA 证书和时区数据
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
# 从构建阶段复制二进制文件
COPY --from=go-builder /app/mailbox .
# 复制配置文件模板（运行时会被环境变量覆盖）
COPY config.yaml .
# 复制数据库初始化脚本（可选）
COPY init.sql .
# 暴露端口
EXPOSE 8080 25
# 启动命令
CMD ["./mailbox"]
