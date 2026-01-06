.PHONY: help build run test docker-build docker-push docker-run clean

help: ## 显示帮助信息
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## 构建前端和后端
	cd web && pnpm install && pnpm run build
	go build -o mailbox main.go

run: ## 运行应用
	go run main.go

test: ## 运行测试
	go test -v ./...

docker-build: ## 构建 Docker 镜像
	docker build -t mailbox:latest .

docker-push: ## 推送 Docker 镜像到 Docker Hub
	docker tag mailbox:latest $(DOCKER_USERNAME)/mailbox:latest
	docker push $(DOCKER_USERNAME)/mailbox:latest

docker-run: ## 使用 Docker Compose 运行
	docker-compose up -d

docker-logs: ## 查看 Docker 日志
	docker-compose logs -f mailbox

docker-stop: ## 停止 Docker 容器
	docker-compose down

docker-clean: ## 清理 Docker 资源
	docker-compose down -v
	docker rmi mailbox:latest

k8s-deploy: ## 部署到 Kubernetes
	kubectl apply -f k8s/deployment.yaml

k8s-delete: ## 从 Kubernetes 删除
	kubectl delete -f k8s/deployment.yaml

clean: ## 清理构建产物
	rm -f mailbox mailbox.exe
	rm -rf web/dist
	rm -rf web/node_modules
