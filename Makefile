# 项目配置
PROJECT_NAME := platform
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD)

# Go 配置
GO_VERSION := 1.21
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# 目录配置
BIN_DIR := bin
DIST_DIR := dist
DOCKER_DIR := docker
K8S_DIR := k8s

# 服务列表
SERVICES := gateway user bobing school yearbill

# Docker 配置
DOCKER_REGISTRY := your-registry.com
IMAGE_NAME := platform
DOCKER_TAG := $(VERSION)

# 测试配置
TEST_TIMEOUT := 10m
COVERAGE_THRESHOLD := 80

# 默认目标
.PHONY: all
all: clean build test

# 帮助信息
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build all services"
	@echo "  test           - Run all tests"
	@echo "  test-unit      - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-performance - Run performance tests"
	@echo "  test-security  - Run security tests"
	@echo "  test-load      - Run load tests"
	@echo "  lint           - Run linters"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-push    - Push Docker images"
	@echo "  docker-run     - Run services with Docker Compose"
	@echo "  k8s-deploy     - Deploy to Kubernetes"
	@echo "  k8s-delete     - Delete from Kubernetes"
	@echo "  proto          - Generate protobuf code"
	@echo "  swagger        - Generate Swagger documentation"
	@echo "  coverage       - Generate coverage report"
	@echo "  benchmark      - Run benchmarks"
	@echo "  profile        - Run profiling"

# 构建
.PHONY: build
build: proto
	@echo "Building $(PROJECT_NAME) $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
			-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)" \
			-o $(BIN_DIR)/$$service \
			./app/$$service/cmd; \
	done
	@echo "Build completed!"

# 测试
.PHONY: test
test: test-unit test-integration test-performance test-security

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	@go test -v -race -timeout=$(TEST_TIMEOUT) -coverprofile=coverage.out ./...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration -timeout=$(TEST_TIMEOUT) ./test/...

.PHONY: test-performance
test-performance:
	@echo "Running performance tests..."
	@go test -v -tags=performance -bench=. -timeout=$(TEST_TIMEOUT) ./test/...

.PHONY: test-security
test-security:
	@echo "Running security tests..."
	@go test -v -tags=security -timeout=$(TEST_TIMEOUT) ./test/...

.PHONY: test-load
test-load:
	@echo "Running load tests..."
	@go test -v -tags=load -timeout=30m ./test/...

# 代码质量
.PHONY: lint
lint:
	@echo "Running linters..."
	@golangci-lint run --timeout=5m

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...

.PHONY: security
security:
	@echo "Running security scan..."
	@gosec ./...

# 清理
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@go clean -cache -testcache

# Docker 相关
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		docker build -t $(DOCKER_REGISTRY)/$(IMAGE_NAME)-$$service:$(DOCKER_TAG) \
			-f ./app/$$service/cmd/Dockerfile .; \
	done

.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		echo "Pushing $$service image..."; \
		docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME)-$$service:$(DOCKER_TAG); \
	done

.PHONY: docker-run
docker-run:
	@echo "Running services with Docker Compose..."
	@docker-compose up -d

.PHONY: docker-stop
docker-stop:
	@echo "Stopping services..."
	@docker-compose down

.PHONY: docker-logs
docker-logs:
	@echo "Showing logs..."
	@docker-compose logs -f

# Kubernetes 相关
.PHONY: k8s-deploy
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f $(K8S_DIR)/namespace.yaml
	@kubectl apply -f $(K8S_DIR)/gateway-deployment.yaml

.PHONY: k8s-delete
k8s-delete:
	@echo "Deleting from Kubernetes..."
	@kubectl delete -f $(K8S_DIR)/gateway-deployment.yaml
	@kubectl delete -f $(K8S_DIR)/namespace.yaml

.PHONY: k8s-status
k8s-status:
	@echo "Checking Kubernetes status..."
	@kubectl get pods -n $(PROJECT_NAME)
	@kubectl get services -n $(PROJECT_NAME)

# 代码生成
.PHONY: proto
proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		idl/*.proto

.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g ./app/gateway/cmd/main.go -o ./docs

# 覆盖率
.PHONY: coverage
coverage: test-unit
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@coverage=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$coverage < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "Coverage $$coverage% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi
	@echo "Coverage: $$coverage%"

# 基准测试
.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# 性能分析
.PHONY: profile
profile:
	@echo "Running profiling..."
	@go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@go tool pprof cpu.prof
	@go tool pprof mem.prof

# 依赖管理
.PHONY: deps
deps:
	@echo "Managing dependencies..."
	@go mod tidy
	@go mod verify

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# 开发环境
.PHONY: dev
dev: docker-run
	@echo "Starting development environment..."
	@echo "Services are running at:"
	@echo "  Gateway: http://localhost:10811"
	@echo "  MySQL: localhost:10821"
	@echo "  Redis: localhost:10916"
	@echo "  etcd: localhost:62379"

.PHONY: dev-stop
dev-stop: docker-stop

# 生产环境
.PHONY: prod
prod: k8s-deploy
	@echo "Deploying to production..."

.PHONY: prod-stop
prod-stop: k8s-delete

# 监控
.PHONY: monitor
monitor:
	@echo "Starting monitoring..."
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000"
	@echo "Jaeger: http://localhost:16686"

# 数据库迁移
.PHONY: migrate-up
migrate-up:
	@echo "Running database migrations..."
	@migrate -path ./migrations -database "mysql://root:password@localhost:3306/platform" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back database migrations..."
	@migrate -path ./migrations -database "mysql://root:password@localhost:3306/platform" down

# 数据初始化
.PHONY: data-init
data-init:
	@echo "Initializing data..."
	@go run ./app/yearBill/script/data.go

# 健康检查
.PHONY: health
health:
	@echo "Checking service health..."
	@curl -f http://localhost:10811/health || echo "Gateway is not healthy"
	@curl -f http://localhost:10811/ready || echo "Gateway is not ready"

# 版本信息
.PHONY: version
version:
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)"

# 安装工具
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 检查环境
.PHONY: check-env
check-env:
	@echo "Checking environment..."
	@go version
	@docker --version
	@kubectl version --client
	@protoc --version
	@echo "Environment check completed!"

# 完整检查
.PHONY: check
check: check-env lint vet security test coverage
	@echo "All checks passed!"

# 发布
.PHONY: release
release: check docker-build docker-push
	@echo "Release $(VERSION) completed!"

# 默认目标
.DEFAULT_GOAL := help