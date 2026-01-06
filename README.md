# MailBox - 临时邮箱系统

一个轻量级、高性能的临时邮箱系统，支持 SMTP 接收邮件、Web 界面查看和管理后台。

## ✨ 功能特性

- **SMTP 服务器** - 内置 SMTP 服务器，自动接收邮件
- **Web 邮箱** - 现代化的 Web 界面查看邮件
- **实时推送** - SSE 实时推送新邮件通知
- **附件支持** - 支持邮件附件上传和下载
- **批量操作** - 支持批量删除邮件（按页、按月、全部）
- **管理后台** - 域名管理、邮箱管理、统计信息
- **多域名支持** - 支持多个邮箱域名配置
- **无需注册** - 直接输入邮箱地址即可查看邮件
- **高性能** - 经过压测验证，支持高并发邮件接收

## 🚀 技术栈

### 后端
- **Golang 1.21** - 高性能后端服务
- **Fiber v2** - 快速的 HTTP 框架
- **GORM** - ORM 数据库操作
- **MySQL 8.0** - 数据存储

### 前端
- **Vue 3** - 渐进式前端框架
- **Vue Router** - 路由管理
- **Axios** - HTTP 客户端
- **TailwindCSS** - 实用优先的 CSS 框架

### 部署
- **Docker** - 容器化部署
- **Docker Compose** - 本地开发和测试
- **Kubernetes** - 生产环境部署
- **GitHub Actions** - CI/CD 自动化

## 📦 快速开始

### 使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone https://github.com/luodaoyi/mailbox.git
cd mailbox

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

访问 http://localhost:8080

默认管理员账号：
- 用户名：`admin`
- 密码：`admin123`

### 使用二进制文件

1. 从 [Releases](https://github.com/luodaoyi/mailbox/releases) 下载对应平台的二进制文件

2. 配置数据库和环境变量：

```bash
# 复制配置文件
cp config.yaml.example config.yaml

# 编辑配置文件
vim config.yaml
```

3. 启动服务：

```bash
./mailbox
```

### 从源码构建

```bash
# 安装依赖
cd web
npm install

# 构建前端
npm run build

# 构建后端
cd ..
go build -o mailbox main.go

# 运行
./mailbox
```

## 🔧 配置说明

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `3306` |
| `DB_USER` | 数据库用户 | `root` |
| `DB_PASSWORD` | 数据库密码 | - |
| `DB_NAME` | 数据库名称 | `mailbox` |
| `ADMIN_USERNAME` | 管理员用户名 | `admin` |
| `ADMIN_PASSWORD` | 管理员密码 | `admin123` |

### 配置文件 (config.yaml)

```yaml
server:
  http_port: 8080
  smtp_port: 25

database:
  host: localhost
  port: 3306
  user: root
  password: your_password
  dbname: mailbox

admin:
  username: admin
  password: admin123
```

## 🐳 Docker 部署

### 构建镜像

```bash
docker build -t mailbox:latest .
```

### 运行容器

```bash
docker run -d \
  -p 8080:8080 \
  -p 25:25 \
  -e DB_HOST=mysql \
  -e DB_PASSWORD=your_password \
  --name mailbox \
  mailbox:latest
```

### Docker Compose

```bash
# 启动
docker-compose up -d

# 停止
docker-compose down

# 查看日志
docker-compose logs -f mailbox

# 重启
docker-compose restart
```

## ☸️ Kubernetes 部署

```bash
# 创建命名空间
kubectl create namespace mailbox

# 部署应用
kubectl apply -f k8s/deployment.yaml

# 查看状态
kubectl get pods -n mailbox

# 查看服务
kubectl get svc -n mailbox
```

## 📡 API 文档

### 用户 API

#### 登录邮箱
```http
POST /api/login
Content-Type: application/json

{
  "email": "test@cc.com"
}
```

#### 获取邮件列表
```http
GET /api/emails?email=test@cc.com&page=1&limit=50
```

#### 获取邮件详情
```http
GET /api/emails/:id?email=test@cc.com
```

#### 删除邮件
```http
DELETE /api/emails/:id?email=test@cc.com
```

#### 批量删除
```http
DELETE /api/emails/page?email=test@cc.com&page=1&limit=50
DELETE /api/emails/month/all?email=test@cc.com
DELETE /api/emails/all?email=test@cc.com
```

### 管理员 API

#### 登录
```http
POST /api/admin/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

#### 域名管理
```http
GET /api/admin/domains
POST /api/admin/domains
PUT /api/admin/domains/:id
DELETE /api/admin/domains/:id
```

#### 邮箱管理
```http
GET /api/admin/mailboxes?domain=cc.com&page=1&limit=50
DELETE /api/admin/mailboxes/:id
DELETE /api/admin/mailboxes/domain/:domain
```

## 🧪 性能测试

项目包含压测脚本 `test_smtp.py`，用于测试 SMTP 服务器性能。

```bash
# 安装依赖
pip install -r requirements.txt

# 运行压测
python test_smtp.py
```

测试结果（1000 封邮件）：
- 成功率：100%
- 平均速度：5.96 封/秒
- 平均延迟：1673.98 ms

## 🛠️ 开发指南

### 项目结构

```
MailBox/
├── cmd/                    # 命令行工具
├── internal/              # 内部包
│   ├── api/              # API 处理器
│   ├── config/           # 配置管理
│   ├── db/               # 数据库连接
│   ├── model/            # 数据模型
│   └── smtp/             # SMTP 服务器
├── web/                   # 前端代码
│   ├── src/
│   │   ├── api/          # API 客户端
│   │   ├── views/        # 页面组件
│   │   └── router/       # 路由配置
│   └── dist/             # 构建产物
├── k8s/                   # Kubernetes 配置
├── .github/workflows/     # GitHub Actions
├── Dockerfile            # Docker 镜像构建
├── docker-compose.yml    # Docker Compose 配置
├── Makefile              # 常用命令
└── main.go               # 入口文件
```

### 本地开发

1. 启动数据库：
```bash
docker-compose up -d mysql
```

2. 启动后端：
```bash
go run main.go
```

3. 启动前端开发服务器：
```bash
cd web
npm run dev
```

### 代码规范

- Go 代码遵循 `gofmt` 格式化标准
- Vue 代码使用 ESLint 检查
- 提交前运行 `make lint` 检查代码

## 🔄 CI/CD

项目使用 GitHub Actions 实现自动化：

### Docker 镜像构建
- 触发条件：推送到 `main` 分支或创建 tag
- 自动构建并推送到 Docker Hub
- 标签：`latest`（main 分支）、版本号（tag）

### Release 发布
- 触发条件：创建 tag（如 `v1.0.0`）
- 自动编译多平台二进制文件
- 支持平台：Linux、Windows、macOS（amd64/arm64）
- 自动创建 GitHub Release

发布新版本：
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## 📝 使用说明

### 接收邮件

1. 在管理后台添加域名（如 `cc.com`）
2. 配置 DNS MX 记录指向服务器
3. 发送邮件到 `任意用户名@cc.com`
4. 在 Web 界面输入邮箱地址查看邮件

### 管理域名

1. 访问 `/admin` 登录管理后台
2. 在域名管理中添加/删除域名
3. 启用/禁用域名接收邮件
4. 查看域名下的邮箱统计

### 邮箱管理

1. 在管理后台选择域名
2. 查看该域名下所有邮箱
3. 查看邮箱邮件数量和最后邮件时间
4. 删除单个邮箱或批量删除

## 🔒 安全建议

- 修改默认管理员密码
- 使用强密码策略
- 启用 HTTPS（使用反向代理如 Nginx）
- 定期备份数据库
- 限制 SMTP 端口访问（防火墙规则）
- 使用环境变量存储敏感信息

## 📊 监控和日志

### 日志查看

```bash
# Docker Compose
docker-compose logs -f mailbox

# Kubernetes
kubectl logs -f deployment/mailbox -n mailbox
```

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:8080/health

# SMTP 连接测试
telnet localhost 25
```

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 🙏 致谢

- [Fiber](https://gofiber.io/) - 快速的 Go Web 框架
- [Vue.js](https://vuejs.org/) - 渐进式 JavaScript 框架
- [TailwindCSS](https://tailwindcss.com/) - 实用优先的 CSS 框架
- [GORM](https://gorm.io/) - Go ORM 库

## 📮 联系方式

- 项目主页：https://github.com/yourusername/mailbox
- 问题反馈：https://github.com/yourusername/mailbox/issues

---

⭐ 如果这个项目对你有帮助，请给个 Star！
