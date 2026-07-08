.PHONY: build run test clean dev

# 应用名
APP_NAME = ai-gateway
BUILD_DIR = ./build

# 构建信息
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)

# Go 相关
CGO_ENABLED ?= 0
GOOS       ?= linux
GOARCH     ?= amd64

## build: 编译二进制
build:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME) \
		./cmd/server/

## run: 直接运行（开发用）
run:
	go run -ldflags "$(LDFLAGS)" ./cmd/server/

## test: 运行全部测试
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## clean: 清理构建产物
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## dev: 开发模式（自动重载，需要安装 air）
dev:
	command -v air >/dev/null 2>&1 || (echo "Installing air..."; go install github.com/air-verse/air@latest)
	air

## deps: 整理依赖
deps:
	go mod tidy
	go mod verify

## lint: 代码检查
lint:
	golangci-lint run ./...

## db-init: 初始化数据库（仅开发环境）
db-init:
	@echo "Database will be created on first run via DATABASE_PATH"

## docker-build: 构建 Docker 镜像
docker-build:
	docker compose build

## docker-up: 启动完整环境
docker-up:
	docker compose up -d

## docker-down: 停止环境
docker-down:
	docker compose down

## verify: 集成验证（需要先启动服务）
verify:
	@echo "Running integration verification..."
	@./scripts/verify.sh

## help: 显示帮助
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' Makefile | sort