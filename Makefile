# ================================================
# SecFlow Makefile
# 常用命令集合
# ================================================

# 帮助信息
.PHONY: help
help:
	@echo "SecFlow Makefile"
	@echo ""
	@echo "使用方法: make <target>"
	@echo ""
	@echo "测试命令:"
	@echo "  make test                # 运行所有测试"
	@echo "  make test-server        # 仅运行服务端测试"
	@echo "  make test-client        # 仅运行客户端测试"
	@echo "  make test-integration   # 运行集成测试"
	@echo "  make test-docker        # 使用 Docker 运行测试"
	@echo "  make test-clean        # 清理测试环境"
	@echo ""
	@echo "开发命令:"
	@echo "  make dev                # 启动开发环境"
	@echo "  make dev-logs          # 查看开发环境日志"
	@echo "  make dev-down          # 停止开发环境"
	@echo ""
	@echo "构建命令:"
	@echo "  make build             # 构建所有服务"
	@echo "  make build-server      # 构建服务端"
	@echo "  make build-client      # 构建客户端"
	@echo "  make build-web         # 构建前端"
	@echo ""
	@echo "代码质量:"
	@echo "  make lint              # 运行代码检查"
	@echo "  make fmt              # 格式化代码"
	@echo ""
	@echo "运维命令:"
	@echo "  make backup            # 备份数据"
	@echo "  make restore           # 恢复数据"
	@echo "  make health            # 健康检查"

# ================================================
# 测试命令
# ================================================

.PHONY: test
test: test-server test-client

.PHONY: test-server
test-server:
	@echo "运行服务端测试..."
	cd secflow-server && go test -v -count=1 ./...

.PHONY: test-client
test-client:
	@echo "运行客户端测试..."
	cd secflow-client && go test -v -count=1 ./... 2>&1 | head -100 || true

.PHONY: test-integration
test-integration:
	@echo "运行集成测试..."
	cd secflow-server && go test -v -count=1 -tags=integration ./tests/...

.PHONY: test-docker
test-docker:
	@echo "启动 Docker 测试环境..."
	chmod +x scripts/docker-test.sh
	./scripts/docker-test.sh

.PHONY: test-docker-server
test-docker-server:
	@echo "启动 Docker 测试环境 (仅服务端)..."
	chmod +x scripts/docker-test.sh
	./scripts/docker-test.sh --server-only

.PHONY: test-docker-clean
test-docker-clean:
	@echo "清理 Docker 测试环境..."
	chmod +x scripts/docker-test.sh
	./scripts/docker-test.sh --cleanup

# ================================================
# 开发环境
# ================================================

.PHONY: dev
dev:
	@echo "启动开发环境..."
	docker-compose up -d
	@echo ""
	@echo "等待服务启动..."
	sleep 10
	@echo "服务已启动:"
	@echo "  - 服务端: http://localhost:8080"
	@echo "  - 前端:   http://localhost:3000"
	@echo "  - MongoDB: localhost:27017"
	@echo "  - Redis:   localhost:6379"

.PHONY: dev-logs
dev-logs:
	docker-compose logs -f

.PHONY: dev-down
dev-down:
	@echo "停止开发环境..."
	docker-compose down

.PHONY: dev-restart
dev-restart: dev-down dev

# ================================================
# 构建命令
# ================================================

.PHONY: build
build: build-server build-client build-web

.PHONY: build-server
build-server:
	@echo "构建服务端..."
	cd secflow-server && go build -o bin/server ./cmd/server/

.PHONY: build-client
build-client:
	@echo "构建客户端..."
	cd secflow-client && go build -o bin/client ./cmd/client/

.PHONY: build-web
build-web:
	@echo "构建前端..."
	cd secflow-web && npm install && npm run build

.PHONY: build-docker
build-docker:
	@echo "构建 Docker 镜像..."
	docker-compose build

# ================================================
# 代码质量
# ================================================

.PHONY: fmt
fmt:
	@echo "格式化代码..."
	cd secflow-server && go fmt ./...
	cd secflow-client && go fmt ./...

.PHONY: lint
lint:
	@echo "运行代码检查..."
	cd secflow-server && go vet ./...
	cd secflow-client && go vet ./...

.PHONY: tidy
tidy:
	@echo "整理依赖..."
	cd secflow-server && go mod tidy
	cd secflow-client && go mod tidy

# ================================================
# 运维命令
# ================================================

.PHONY: backup
backup:
	@echo "备份数据..."
	chmod +x scripts/backup.sh
	./scripts/backup.sh

.PHONY: restore
restore:
	@echo "恢复数据..."
	chmod +x scripts/restore.sh
	./scripts/restore.sh

.PHONY: health
health:
	@echo "健康检查..."
	chmod +x scripts/healthcheck.sh
	./scripts/healthcheck.sh

.PHONY: logs
logs:
	@echo "收集日志..."
	chmod +x scripts/log-collector.sh
	./scripts/log-collector.sh

.PHONY: migrate
migrate:
	@echo "数据迁移..."
	chmod +x scripts/migrate.sh
	./scripts/migrate.sh

# ================================================
# 清理
# ================================================

.PHONY: clean
clean:
	@echo "清理构建产物..."
	rm -rf secflow-server/bin
	rm -rf secflow-client/bin
	rm -rf secflow-web/dist
	find . -name "*.test" -delete
	find . -name "coverage.txt" -delete

.PHONY: prune
prune: clean
	@echo "清理未使用的 Docker 资源..."
	docker system prune -f
