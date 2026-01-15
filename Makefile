# OpenEAAP Makefile
# 提供统一的构建、测试、部署命令

# ============================================================================
# 变量定义
# ============================================================================

# 项目信息
PROJECT_NAME := openeeap
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Go 相关
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"
GO_FILES := $(shell find . -name '*.go' -type f -not -path './vendor/*')
GO_PACKAGES := $(shell go list ./... | grep -v /vendor/)

# 构建目录
BUILD_DIR := ./build
BIN_DIR := $(BUILD_DIR)/bin
DIST_DIR := $(BUILD_DIR)/dist

# 目标平台
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Docker 相关
DOCKER_REGISTRY := docker.io
DOCKER_NAMESPACE := openeeap
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(PROJECT_NAME)
DOCKER_TAG := $(VERSION)

# Kubernetes 相关
K8S_NAMESPACE := openeeap
HELM_CHART := ./deployments/helm/openeeap

# 测试相关
COVERAGE_DIR := $(BUILD_DIR)/coverage
COVERAGE_PROFILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Proto 相关
PROTO_DIR := ./api/proto
PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')
PROTO_OUT_DIR := ./pkg/pb

# OpenAPI 相关
OPENAPI_SPEC := ./api/openapi/openapi.yaml
OPENAPI_OUT_DIR := ./pkg/openapi

# ============================================================================
# 默认目标
# ============================================================================

.DEFAULT_GOAL := help

# ============================================================================
# Help
# ============================================================================

.PHONY: help
help: ## 显示帮助信息
	@echo "OpenEAAP Makefile Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================================================
# 构建相关
# ============================================================================

.PHONY: build
build: ## 构建所有二进制文件
	@echo "Building $(PROJECT_NAME) $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/server ./cmd/server
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/worker ./cmd/worker
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/cli ./cmd/cli
	@echo "Build completed: $(BIN_DIR)/"

.PHONY: build-server
build-server: ## 构建 server 二进制文件
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/server ./cmd/server

.PHONY: build-worker
build-worker: ## 构建 worker 二进制文件
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/worker ./cmd/worker

.PHONY: build-cli
build-cli: ## 构建 cli 二进制文件
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/cli ./cmd/cli

.PHONY: build-all
build-all: ## 构建所有平台的二进制文件
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTPUT_NAME=$(PROJECT_NAME)-$$OS-$$ARCH; \
		if [ $$OS = "windows" ]; then OUTPUT_NAME=$$OUTPUT_NAME.exe; fi; \
		echo "Building $$OUTPUT_NAME..."; \
		GOOS=$$OS GOARCH=$$ARCH $(GO) build $(LDFLAGS) -o $(DIST_DIR)/$$OUTPUT_NAME ./cmd/server; \
	done
	@echo "Cross-platform build completed: $(DIST_DIR)/"

.PHONY: clean
clean: ## 清理构建产物
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf vendor/
	@find . -name '*.test' -type f -delete
	@echo "Clean completed."

# ============================================================================
# 依赖管理
# ============================================================================

.PHONY: deps
deps: ## 下载依赖
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify

.PHONY: deps-update
deps-update: ## 更新依赖
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

.PHONY: deps-vendor
deps-vendor: ## 将依赖复制到 vendor 目录
	@echo "Vendoring dependencies..."
	$(GO) mod vendor

# ============================================================================
# 代码生成
# ============================================================================

.PHONY: generate
generate: generate-proto generate-openapi generate-mocks ## 生成所有代码

.PHONY: generate-proto
generate-proto: ## 生成 gRPC 代码
	@echo "Generating gRPC code from proto files..."
	@mkdir -p $(PROTO_OUT_DIR)
	@for proto in $(PROTO_FILES); do \
		protoc --go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
			--go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative \
			$$proto; \
	done
	@echo "gRPC code generation completed."

.PHONY: generate-openapi
generate-openapi: ## 生成 OpenAPI 客户端
	@echo "Generating OpenAPI client..."
	@mkdir -p $(OPENAPI_OUT_DIR)
	oapi-codegen -generate types,client -package openapi -o $(OPENAPI_OUT_DIR)/client.go $(OPENAPI_SPEC)
	@echo "OpenAPI client generation completed."

.PHONY: generate-mocks
generate-mocks: ## 生成 Mock 代码
	@echo "Generating mocks..."
	$(GO) generate ./...
	@echo "Mock generation completed."

# ============================================================================
# 测试相关
# ============================================================================

.PHONY: test
test: ## 运行单元测试
	@echo "Running unit tests..."
	$(GO) test -v -race -short $(GO_PACKAGES)

.PHONY: test-coverage
test-coverage: ## 运行测试并生成覆盖率报告
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_PROFILE) -covermode=atomic $(GO_PACKAGES)
	$(GO) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

.PHONY: test-integration
test-integration: ## 运行集成测试
	@echo "Running integration tests..."
	$(GO) test -v -race -tags=integration ./test/integration/...

.PHONY: test-e2e
test-e2e: ## 运行端到端测试
	@echo "Running e2e tests..."
	$(GO) test -v -race -tags=e2e ./test/e2e/...

.PHONY: test-all
test-all: test test-integration test-e2e ## 运行所有测试

.PHONY: benchmark
benchmark: ## 运行基准测试
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem $(GO_PACKAGES)

# ============================================================================
# 代码质量
# ============================================================================

.PHONY: fmt
fmt: ## 格式化代码
	@echo "Formatting code..."
	gofmt -s -w $(GO_FILES)
	goimports -w $(GO_FILES)

.PHONY: lint
lint: ## 运行代码检查
	@echo "Running linters..."
	golangci-lint run --timeout 5m

.PHONY: vet
vet: ## 运行 go vet
	@echo "Running go vet..."
	$(GO) vet $(GO_PACKAGES)

.PHONY: check
check: fmt vet lint ## 运行所有代码检查

# ============================================================================
# 数据库迁移
# ============================================================================

.PHONY: migrate-up
migrate-up: ## 执行数据库迁移（向上）
	@echo "Running database migrations (up)..."
	migrate -path ./migrations -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## 执行数据库迁移（向下）
	@echo "Running database migrations (down)..."
	migrate -path ./migrations -database "$(DATABASE_URL)" down

.PHONY: migrate-create
migrate-create: ## 创建新的迁移文件 (usage: make migrate-create NAME=add_users_table)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=add_users_table"; exit 1; fi
	@echo "Creating migration: $(NAME)..."
	migrate create -ext sql -dir ./migrations -seq $(NAME)

# ============================================================================
# Docker 相关
# ============================================================================

.PHONY: docker-build
docker-build: ## 构建 Docker 镜像
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f ./deployments/docker/Dockerfile .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

.PHONY: docker-push
docker-push: ## 推送 Docker 镜像
	@echo "Pushing Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run: ## 运行 Docker 容器
	@echo "Running Docker container..."
	docker run -d --name $(PROJECT_NAME) \
		-p 8080:8080 -p 9090:9090 \
		-e DATABASE_URL=$(DATABASE_URL) \
		-e REDIS_URL=$(REDIS_URL) \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-stop
docker-stop: ## 停止 Docker 容器
	@echo "Stopping Docker container..."
	docker stop $(PROJECT_NAME)
	docker rm $(PROJECT_NAME)

# ============================================================================
# Kubernetes 部署
# ============================================================================

.PHONY: k8s-deploy
k8s-deploy: ## 部署到 Kubernetes
	@echo "Deploying to Kubernetes..."
	kubectl apply -f ./deployments/kubernetes/ -n $(K8S_NAMESPACE)

.PHONY: k8s-delete
k8s-delete: ## 从 Kubernetes 删除
	@echo "Deleting from Kubernetes..."
	kubectl delete -f ./deployments/kubernetes/ -n $(K8S_NAMESPACE)

.PHONY: helm-install
helm-install: ## 使用 Helm 安装
	@echo "Installing with Helm..."
	helm install $(PROJECT_NAME) $(HELM_CHART) -n $(K8S_NAMESPACE) --create-namespace

.PHONY: helm-upgrade
helm-upgrade: ## 使用 Helm 升级
	@echo "Upgrading with Helm..."
	helm upgrade $(PROJECT_NAME) $(HELM_CHART) -n $(K8S_NAMESPACE)

.PHONY: helm-uninstall
helm-uninstall: ## 使用 Helm 卸载
	@echo "Uninstalling with Helm..."
	helm uninstall $(PROJECT_NAME) -n $(K8S_NAMESPACE)

# ============================================================================
# 本地运行
# ============================================================================

.PHONY: run-server
run-server: build-server ## 运行 server
	@echo "Running server..."
	$(BIN_DIR)/server --config ./configs/development.yaml

.PHONY: run-worker
run-worker: build-worker ## 运行 worker
	@echo "Running worker..."
	$(BIN_DIR)/worker --config ./configs/development.yaml

.PHONY: run-cli
run-cli: build-cli ## 运行 cli
	@echo "Running cli..."
	$(BIN_DIR)/cli --help

.PHONY: dev
dev: ## 以开发模式运行（带热重载）
	@echo "Running in development mode with hot reload..."
	air -c .air.toml

# ============================================================================
# 文档生成
# ============================================================================

.PHONY: docs
docs: ## 生成文档
	@echo "Generating documentation..."
	godoc -http=:6060

.PHONY: docs-swagger
docs-swagger: ## 生成 Swagger 文档
	@echo "Generating Swagger documentation..."
	swag init -g cmd/server/main.go -o ./docs/swagger

# ============================================================================
# 工具安装
# ============================================================================

.PHONY: install-tools
install-tools: ## 安装开发工具
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest
	go install github.com/cosmtrek/air@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ============================================================================
# 信息显示
# ============================================================================

.PHONY: version
version: ## 显示版本信息
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"

.PHONY: info
info: version ## 显示项目信息
	@echo ""
	@echo "Go Version: $(shell $(GO) version)"
	@echo "GOPATH: $(GOPATH)"
	@echo "GOOS: $(shell go env GOOS)"
	@echo "GOARCH: $(shell go env GOARCH)"

# Personal.AI order the ending
