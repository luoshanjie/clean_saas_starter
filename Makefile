# Makefile for service - basic build/test helpers

# 可执行文件名
APP_NAME := service
# 项目展示名（日志用）
PROJECT := service
# 入口文件（暂未创建时会提示）
SRC := cmd/service/main.go
# 编译产物目录
BUILD_DIR := build
# go build 缓存目录（避免默认缓存权限问题）
GOCACHE := $(CURDIR)/.gocache

# 默认 OS/ARCH 取 go env，可用 OS=linux ARCH=arm64 覆盖
OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)

# 可选的编译注入信息（Git 短 SHA、UTC 构建时间）
GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo "nogit")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'main.gitSHA=$(GIT_SHA)' -X 'main.buildTime=$(BUILD_TIME)'

.PHONY: default build mac linux build-linux run clean test test-cover test-gate dep check-src swagger dev test-integration
.PHONY: sqlc-generate sqlc-vet sqlc sqlc-check
.PHONY: cli

default: build  # 默认执行 build

check-src:
	@if [ ! -f "$(SRC)" ]; then \
		echo "[${PROJECT}] Missing entrypoint: $(SRC)"; \
		echo "[${PROJECT}] Create it first (e.g. cmd/service/main.go)."; \
		exit 1; \
	fi

build: check-src
	@echo "🔧 Building $(OS)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(OS) GOARCH=$(ARCH) CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o "$(BUILD_DIR)/$(APP_NAME)$(if $(filter windows,$(OS)),.exe,)" "$(SRC)"
	@echo "✅ Output: $(BUILD_DIR)/$(APP_NAME)$(if $(filter windows,$(OS)),.exe,)"

mac:   OS=darwin
mac:   build
linux: OS=linux
linux: build
build-linux: linux

run: check-src
	@echo "🚀 Running (go run)..."
	go run "$(SRC)"

## -------- Dev --------
dev: check-src
	@echo "[${PROJECT}] Running in dev mode (load .env, enable swagger)..."
	@set -a; [ -f .env ] && . ./.env; set +a; \
	GOFLAGS="-tags=swagger" go run "$(SRC)"

cli:
	@echo "[${PROJECT}] Running CLI..."
	go run ./cmd/cli/main.go $(CLI_ARGS)


## -------- Tests --------
test:
	@echo "[${PROJECT}] Running tests..."
	go test ./... -v

test-integration:
	@echo "[${PROJECT}] Running integration tests..."
	@set -a; [ -f .env ] && . ./.env; set +a; \
	go test -tags=integration ./internal/integration -v

test-cover:
	@echo "[${PROJECT}] Running tests (coverage)..."
	go test ./... -coverprofile=coverage.out
	@go tool cover -func=coverage.out | tail -n 1

# 提交前质量门禁：
# 1) 关键包测试必须通过
# 2) 覆盖率达到阈值（默认 55，可通过 COVER_MIN=80 覆盖）
COVER_MIN ?= 55
test-gate:
	@echo "[${PROJECT}] Running quality gate..."
	@go test ./internal/app/usecase ./internal/delivery/http/handler -coverprofile=coverage.out
	@COVER=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
		echo "[${PROJECT}] coverage=$${COVER}% (min=$(COVER_MIN)%)"; \
		awk "BEGIN {exit !($${COVER} >= $(COVER_MIN))}" || (echo "[${PROJECT}] coverage gate failed"; exit 1)

## -------- Deps --------
dep:
	@echo "[${PROJECT}] Tidy dependencies..."
	go mod tidy

## -------- Swagger --------
swagger:
	@echo "[${PROJECT}] Generating swagger docs..."
	swag init -g main.go -o docs/swagger --dir ./cmd/service,./internal/delivery/http/handler,./internal/delivery/http/resp

## -------- SQLC --------
sqlc-generate:
	@echo "[${PROJECT}] Generating sqlc code..."
	sqlc generate

sqlc-vet:
	@echo "[${PROJECT}] Running sqlc vet..."
	sqlc vet

sqlc: sqlc-generate sqlc-vet

sqlc-check: sqlc
	@echo "[${PROJECT}] Checking generated files are committed..."
	git diff --exit-code

## -------- Clean --------
clean:
	@echo "[${PROJECT}] Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(APP_NAME) coverage.out
