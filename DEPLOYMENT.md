# 部署文档

## 快速开始

### 使用 Docker Compose（推荐）

1. 克隆仓库并进入项目目录
2. 启动服务：
```bash
docker-compose up -d
```

3. 访问应用：http://localhost:8080
   - 默认管理员账号：admin
   - 默认密码：admin123

4. 查看日志：
```bash
docker-compose logs -f mailbox
```

5. 停止服务：
```bash
docker-compose down
```

### 使用 Docker 镜像

```bash
# 拉取镜像
docker pull <your-dockerhub-username>/mailbox:latest

# 运行容器（需要先启动 MySQL）
docker run -d \
  --name mailbox \
  -p 8080:8080 \
  -p 25:25 \
  -e DB_HOST=mysql \
  -e DB_PORT=3306 \
  -e DB_USER=mailbox \
  -e DB_PASSWORD=mailbox123 \
  -e DB_NAME=mailbox \
  -e ADMIN_USERNAME=admin \
  -e ADMIN_PASSWORD=admin123 \
  <your-dockerhub-username>/mailbox:latest
```

## CI/CD 配置

### GitHub Actions Secrets

需要在 GitHub 仓库设置以下 Secrets：

1. `DOCKER_USERNAME` - Docker Hub 用户名
2. `DOCKER_PASSWORD` - Docker Hub 访问令牌

### 自动化流程

1. **Docker 构建和推送** (`.github/workflows/docker-build.yml`)
   - 触发条件：推送到 main 分支或创建 tag
   - 自动构建多架构镜像并推送到 Docker Hub
   - 标签策略：
     - main 分支 → `latest`
     - tag v1.2.3 → `1.2.3`, `1.2`, `latest`

2. **Release 发布** (`.github/workflows/release.yml`)
   - 触发条件：推送 tag (v*)
   - 编译多平台二进制文件
   - 自动创建 GitHub Release
   - 支持平台：
     - Linux (amd64, arm64)
     - Windows (amd64)
     - macOS (amd64, arm64)

### 发布新版本

```bash
# 创建并推送 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## 环境变量配置

应用支持通过环境变量覆盖配置文件：

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| DB_HOST | 数据库主机 | 192.168.11.5 |
| DB_PORT | 数据库端口 | 3306 |
| DB_USER | 数据库用户 | root |
| DB_PASSWORD | 数据库密码 | - |
| DB_NAME | 数据库名称 | mailbox |
| ADMIN_USERNAME | 管理员用户名 | admin |
| ADMIN_PASSWORD | 管理员密码 | admin123 |

## 健康检查

- HTTP API: `http://localhost:8080/api/health`
- SMTP: 端口 25

## 回滚策略

### Docker Compose 回滚

```bash
# 回滚到指定版本
docker-compose down
docker pull <your-dockerhub-username>/mailbox:1.0.0
docker-compose up -d
```

### Kubernetes 回滚

```bash
kubectl rollout undo deployment/mailbox
kubectl rollout status deployment/mailbox
```

## 监控建议

1. 容器健康检查：监控容器状态
2. 日志收集：使用 ELK 或 Loki
3. 指标监控：Prometheus + Grafana
4. 告警：配置关键指标告警（CPU、内存、磁盘、错误率）

## 生产环境注意事项

1. **安全性**
   - 修改默认管理员密码
   - 使用强密码策略
   - 配置 HTTPS（使用 Nginx 反向代理）
   - 限制 SMTP 端口访问

2. **数据持久化**
   - 定期备份 MySQL 数据卷
   - 配置数据库主从复制

3. **性能优化**
   - 根据负载调整容器资源限制
   - 配置数据库连接池
   - 启用 Redis 缓存（如需要）

4. **高可用**
   - 使用 Kubernetes 部署多副本
   - 配置负载均衡器
   - 设置自动扩缩容策略
