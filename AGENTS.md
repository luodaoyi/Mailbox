# Repository Guidelines

## 项目结构与模块组织
- `internal/` 为后端核心逻辑：`api/` 路由与处理、`config/` 配置加载、`db/` 数据库初始化、`model/` 数据模型、`smtp/` SMTP 接收服务。
- `web/` 为前端应用：`src/` 源码、`dist/` 构建产物（被 Go `embed` 打包）。
- `k8s/` Kubernetes 部署清单，`.github/workflows/` CI/CD。
- 根目录包含 `config.yaml`、`docker-compose.yml`、`init.sql` 等运行与初始化资源。

## 构建、测试与开发命令
- 一键构建：`make build`（前端 `pnpm install && pnpm run build`，后端 `go build -o mailbox`）。
- 本地运行：`make run` 或 `go run main.go`。
- 前端开发：`cd web` 后执行 `pnpm install`、`pnpm run dev`。
- 运行测试：`make test` 或 `go test -v ./...`。
- 容器运行：`docker-compose up -d`；日志：`docker-compose logs -f mailbox`。

## 编码风格与命名约定
- Go 代码遵循 `gofmt`（使用制表符缩进），包名小写、函数名驼峰。
- Vue 组件采用 PascalCase 文件名（如 `Inbox.vue`），模板缩进 2 空格。
- JavaScript 使用单引号（见 `web/src/router/index.js`）。

## 测试指南
- 后端单元测试遵循 Go 习惯：文件名 `*_test.go`，位于对应包内。
- SMTP 联调使用脚本：`python test_smtp.py`（需本地 SMTP:25、启用 `cc.com` 域名）。

## 提交与 PR 指南
- 提交信息以中文动词开头、描述清晰（如“修复…/新增…/重构…”），单行优先。
- PR 需包含：变更概述、验证步骤、相关 issue；UI 变更附截图。

## 安全与配置提示
- 运行前必须设置 `JWT_SECRET`；敏感信息使用环境变量或 `.env.example` 参考。
- 数据库配置在 `config.yaml` 与环境变量中，初始化脚本见 `init.sql`。
- 默认端口：HTTP `8080`、SMTP `25`。

## 架构概览（快速理解）
- Go 后端提供 API 与内置 SMTP 服务，前端为 Vue 3；邮件列表与详情通过 API 获取，SSE 推送新邮件。
- 前端构建产物位于 `web/dist`，由后端 `embed` 打包到可执行文件中。
