.PHONY: help env build up down restart logs ps deploy deploy-seed clean shell

COMPOSE ?= docker compose
IMAGE ?= soulnexus:latest
export DOCKER_BUILDKIT := 1
export COMPOSE_DOCKER_CLI_BUILD := 1

help: ## 显示可用命令
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z0-9_.-]+:.*##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

env: ## 从 env.example 复制 .env（已存在则跳过；可不改也能本地启动）
	@test -f .env || cp env.example .env
	@echo ".env ready — 生产请改 SESSION_SECRET / PLATFORM_ADMIN_* / DSN"

build: ## 构建 Docker 镜像
	$(COMPOSE) build

up: env ## 启动容器（后台）
	$(COMPOSE) up -d

down: ## 停止并移除容器
	$(COMPOSE) down

restart: ## 重启容器
	$(COMPOSE) restart

logs: ## 跟踪容器日志
	$(COMPOSE) logs -f

ps: ## 查看容器状态
	$(COMPOSE) ps

deploy: env build up ## 一键部署（构建 + 启动，浏览器打开控制台）
	@echo ""
	@echo "✓ SoulNexus ready → http://localhost:$${HTTP_PORT:-8080}"
	@echo "  Platform admin: $${PLATFORM_ADMIN_EMAIL:-admin@lingecho.com} / $${PLATFORM_ADMIN_PASSWORD:-admin123}"

deploy-seed: env ## 一键部署并写入演示种子数据（非生产）
	LINGECHO_SEED=true $(MAKE) deploy

clean: down ## 停止容器并删除数据卷（会清空数据库与上传文件）
	$(COMPOSE) down -v

shell: ## 进入运行中容器 shell
	$(COMPOSE) exec lingecho bash
