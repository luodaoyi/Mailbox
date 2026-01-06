# MailBox 部署快速开始

## 本地开发部署

### 使用 Docker Compose（推荐）

```bash
# 1. 启动所有服务
docker-compose up -d

# 2. 查看日志
docker-compose logs -f mailbox

# 3. 访问应用
# HTTP: http://localhost:8080
# SMTP: localhost:25

# 4. 停止服务
docker-compose down
```

### 手动构建和运行

```bash
# 1. 构建前端
cd web
pnpm install
pnpm run build
cd ..

# 2. 构建后端
go build -o mailbox main.go

# 3. 运行（需要先启动 MySQL）
./mailbox
```

## 生产环境部署

### Docker Hub 部署

```bash
# 1. 构建镜像
docker build -t your-username/mailbox:latest .

# 2. 推送到 Docker Hub
docker push your-username/mailbox:latest

# 3. 在服务器上拉取并运行
docker pull your-username/mailbox:latest
docker-compose up -d
```

### Kubernetes 部署

```bash
# 1. 修改 k8s/deployment.yaml 中的镜像地址
# 将 <your-dockerhub-username> 替换为你的 Docker Hub 用户名

# 2. 部署到集群
kubectl apply -f k8s/deployment.yaml

# 3. 查看部署状态
kubectl get pods -n mailbox
kubectl get svc -n mailbox

# 4. 查看日志
kubectl logs -f deployment/mailbox -n mailbox
```

## CI/CD 配置

### GitHub Actions 设置

1. 在 GitHub 仓库设置中添加 Secrets：
   - `DOCKER_USERNAME`: Docker Hub 用户名
   - `DOCKER_PASSWORD`: Docker Hub 访问令牌

2. 推送代码到 main 分支会自动触发 Docker 镜像构建

3. 创建 tag 会自动发布新版本：
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## 环境变量配置

复制 `.env.example` 并修改：

```bash
cp .env.example .env
# 编辑 .env 文件，修改数据库密码等敏感信息
```

支持的环境变量：
- `DB_HOST`: 数据库主机
- `DB_PORT`: 数据库端口
- `DB_USER`: 数据库用户
- `DB_PASSWORD`: 数据库密码
- `DB_NAME`: 数据库名称
- `ADMIN_USERNAME`: 管理员用户名
- `ADMIN_PASSWORD`: 管理员密码

## 常用命令

```bash
# 使用 Makefile
make help              # 显示所有可用命令
make build             # 构建应用
make docker-build      # 构建 Docker 镜像
make docker-run        # 运行 Docker Compose
make docker-logs       # 查看日志
make k8s-deploy        # 部署到 Kubernetes
```

## 故障排查

### 容器无法启动

```bash
# 查看容器日志
docker-compose logs mailbox

# 检查容器状态
docker-compose ps
```

### 数据库连接失败

```bash
# 检查 MySQL 容器是否运行
docker-compose ps mysql

# 进入 MySQL 容器
docker-compose exec mysql mysql -u root -p

# 检查数据库是否创建
SHOW DATABASES;
```

### 端口冲突

如果端口 8080 或 25 已被占用，修改 `docker-compose.yml` 中的端口映射：

```yaml
ports:
  - "8081:8080"  # 将主机端口改为 8081
  - "2525:25"    # 将主机端口改为 2525
```

## 安全建议

1. 修改默认密码
2. 使用 HTTPS（配置 Nginx 反向代理）
3. 限制 SMTP 端口访问
4. 定期备份数据库
5. 使用强密码策略

详细部署文档请参考 [DEPLOYMENT.md](DEPLOYMENT.md)
