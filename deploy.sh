#!/bin/bash
set -e

# MailBox 一键部署脚本
# 无需克隆项目，自动下载所需文件并使用预构建 Docker 镜像部署
# 用法: curl -fsSL https://raw.githubusercontent.com/luodaoyi/Mailbox/main/deploy.sh | bash

REPO_RAW="https://raw.githubusercontent.com/luodaoyi/Mailbox/main"
DEPLOY_DIR="${MAILBOX_DEPLOY_DIR:-./mailbox-deploy}"
IMAGE="ghcr.io/luodaoyi/mailbox:latest"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info()    { echo -e "${GREEN}[INFO]${NC} $*"; }
warning() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# 检查依赖
check_deps() {
    for cmd in docker curl; do
        command -v "$cmd" >/dev/null 2>&1 || error "未找到 $cmd，请先安装后重试"
    done
    # 检查 docker compose (v2) 或 docker-compose (v1)
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_CMD="docker compose"
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_CMD="docker-compose"
    else
        error "未找到 docker compose 或 docker-compose，请先安装后重试"
    fi
}

# 生成随机字母数字字符串（避免特殊字符导致 shell/YAML 解析问题）
gen_secret() {
    LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32
}

# 下载初始化 SQL
download_init_sql() {
    info "下载数据库初始化文件..."
    curl -fsSL "${REPO_RAW}/init.sql" -o "${DEPLOY_DIR}/init.sql" || \
        error "下载 init.sql 失败，请检查网络连接"
}

# 创建 docker-compose.yml
create_compose_file() {
    info "生成 docker-compose.yml..."
    cat > "${DEPLOY_DIR}/docker-compose.yml" <<EOF
version: '3.8'

services:
  mailbox:
    image: ${IMAGE}
    ports:
      - "\${HTTP_PORT:-8080}:8080"
      - "\${SMTP_PORT:-25}:25"
    environment:
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=mailbox
      - DB_PASSWORD=\${DB_PASSWORD}
      - DB_NAME=mailbox
      - ADMIN_USERNAME=\${ADMIN_USERNAME:-admin}
      - ADMIN_PASSWORD=\${ADMIN_PASSWORD:-admin123}
      - JWT_SECRET=\${JWT_SECRET}
    depends_on:
      mysql:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - mailbox-network

  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=\${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=mailbox
      - MYSQL_USER=mailbox
      - MYSQL_PASSWORD=\${DB_PASSWORD}
    volumes:
      - mysql_data:/var/lib/mysql
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p\${MYSQL_ROOT_PASSWORD}"]
      timeout: 5s
      retries: 10
    networks:
      - mailbox-network

networks:
  mailbox-network:
    driver: bridge

volumes:
  mysql_data:
EOF
}

# 创建 .env 文件
create_env_file() {
    if [ -f "${DEPLOY_DIR}/.env" ]; then
        warning ".env 文件已存在，跳过生成（如需重置请删除该文件）"
        return
    fi
    info "生成 .env 配置文件..."
    local jwt_secret db_password mysql_root_password admin_password
    jwt_secret=$(gen_secret)
    db_password=$(gen_secret)
    mysql_root_password=$(gen_secret)
    admin_password=$(gen_secret)
    cat > "${DEPLOY_DIR}/.env" <<EOF
# MailBox 环境配置
# 请妥善保管此文件，其中包含所有服务的访问凭据

# 数据库配置
DB_PASSWORD=${db_password}
MYSQL_ROOT_PASSWORD=${mysql_root_password}

# 管理员账号（首次启动生效）
ADMIN_USERNAME=admin
ADMIN_PASSWORD=${admin_password}

# JWT 密钥（自动生成，请勿泄露）
JWT_SECRET=${jwt_secret}

# 服务端口（如有冲突可修改）
HTTP_PORT=8080
SMTP_PORT=25
EOF
    info "管理员密码已自动生成，请记录：${admin_password}"
}

# 主流程
main() {
    info "MailBox 一键部署脚本"
    info "部署目录: ${DEPLOY_DIR}"

    check_deps

    mkdir -p "${DEPLOY_DIR}"
    cd "${DEPLOY_DIR}"

    download_init_sql
    create_compose_file
    create_env_file

    info "拉取 Docker 镜像（首次可能需要几分钟）..."
    $COMPOSE_CMD pull

    info "启动服务..."
    $COMPOSE_CMD up -d

    echo ""
    info "✅ 部署完成！"
    echo ""
    # 从 .env 中读取生成的管理员密码用于展示
    local admin_pass
    admin_pass=$(grep '^ADMIN_PASSWORD=' "${DEPLOY_DIR}/.env" | cut -d= -f2)
    local http_port
    http_port=$(grep '^HTTP_PORT=' "${DEPLOY_DIR}/.env" | cut -d= -f2)
    http_port="${http_port:-8080}"
    local smtp_port
    smtp_port=$(grep '^SMTP_PORT=' "${DEPLOY_DIR}/.env" | cut -d= -f2)
    smtp_port="${smtp_port:-25}"
    echo "  Web 界面:    http://localhost:${http_port}  （远程部署请将 localhost 替换为服务器 IP）"
    echo "  管理后台:    http://localhost:${http_port}/admin"
    echo "  SMTP 端口:   ${smtp_port}"
    echo ""
    echo "  管理员账号:  admin"
    echo "  管理员密码:  ${admin_pass}"
    echo "  （密码已保存至 ${DEPLOY_DIR}/.env）"
    echo ""
    echo "  查看日志:    cd ${DEPLOY_DIR} && $COMPOSE_CMD logs -f mailbox"
    echo "  停止服务:    cd ${DEPLOY_DIR} && $COMPOSE_CMD down"
}

main "$@"
