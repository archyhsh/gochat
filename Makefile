.PHONY: all build clean run test lint proto docker help

# Go 参数
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# 二进制输出目录
BIN_DIR=bin

# 服务列表
SERVICES=gateway message relation push file

# 版本信息
VERSION?=1.0.0
BUILD_TIME=$(shell date +%FT%T%z)
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建参数
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# 默认目标
all: build

# 帮助信息
help:
	@echo "GoChat - 高并发IM即时通讯系统"
	@echo ""
	@echo "Usage:"
	@echo "  make build         - 构建所有服务"
	@echo "  make build-gateway - 构建 Gateway 服务"
	@echo "  make build-message - 构建 Message 服务"
	@echo "  make build-relation- 构建 Relation 服务"
	@echo "  make build-push    - 构建 Push 服务"
	@echo "  make build-file    - 构建 File 服务"
	@echo "  make run-gateway   - 运行 Gateway 服务"
	@echo "  make test          - 运行测试"
	@echo "  make lint          - 代码检查"
	@echo "  make proto         - 生成 Protobuf 代码"
	@echo "  make docker        - 构建 Docker 镜像"
	@echo "  make docker-push   - 推送 Docker 镜像"
	@echo "  make infra-up      - 启动基础设施 (MySQL, Redis, Kafka, MinIO)"
	@echo "  make infra-down    - 停止基础设施"
	@echo "  make migrate       - 执行数据库迁移"
	@echo "  make clean         - 清理构建产物"
	@echo "  make deps          - 下载依赖"
	@echo "  make tidy          - 整理依赖"

# 下载依赖
deps:
	$(GOGET) -v ./...

# 整理依赖
tidy:
	$(GOMOD) tidy

# 构建所有服务
build: $(addprefix build-,$(SERVICES))

# 构建单个服务
build-%:
	@echo "Building $*..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$* ./cmd/$*

# 运行服务
run-gateway:
	$(GOCMD) run ./cmd/gateway -config configs/gateway.yaml

run-message:
	$(GOCMD) run ./cmd/message -config configs/message.yaml

run-relation:
	$(GOCMD) run ./cmd/relation -config configs/relation.yaml

run-push:
	$(GOCMD) run ./cmd/push -config configs/push.yaml

run-file:
	$(GOCMD) run ./cmd/file -config configs/file.yaml

# 运行所有服务
run: infra-up
	@echo "Starting all services..."
	@$(MAKE) -j5 run-gateway run-message run-relation run-push run-file

# 测试
test:
	$(GOTEST) -v -race -cover ./...

# 测试覆盖率
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# 代码检查
lint:
	$(GOLINT) run ./...

# 代码格式化
fmt:
	$(GOFMT) -s -w .

# 生成 Protobuf
proto:
	@echo "Generating protobuf..."
	@find api/proto -name "*.proto" -exec protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		{} \;

# 启动基础设施
infra-up:
	docker-compose -f deployments/docker-compose.yml up -d mysql redis kafka zookeeper minio

# 停止基础设施
infra-down:
	docker-compose -f deployments/docker-compose.yml down

# 数据库迁移
migrate:
	@echo "Running database migrations..."
	$(GOCMD) run scripts/migrate.go

# Docker 构建
docker: $(addprefix docker-build-,$(SERVICES))

docker-build-%:
	@echo "Building Docker image for $*..."
	docker build -t gochat/$*:$(VERSION) -f deployments/docker/Dockerfile.$* .

# Docker 推送
docker-push: $(addprefix docker-push-,$(SERVICES))

docker-push-%:
	docker push gochat/$*:$(VERSION)

# Docker Compose 启动所有服务
up:
	docker-compose -f deployments/docker-compose.yml up -d

# Docker Compose 停止所有服务
down:
	docker-compose -f deployments/docker-compose.yml down

# 查看日志
logs:
	docker-compose -f deployments/docker-compose.yml logs -f

# 清理
clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

# 开发模式 (热重载)
dev-gateway:
	air -c .air.gateway.toml

# 生成 API 文档
docs:
	swag init -g cmd/gateway/main.go -o docs/swagger
