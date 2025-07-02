# Define variables
PROJECT_NAME := gitlab-fork-cli
VERSION ?= v0.1 # Default version is 'latest', can be overridden with make VERSION=v1.0
DOCKER_IMAGE := build-harbor.alauda.cn/test/$(PROJECT_NAME):$(VERSION)

# Define Go build-specific variables
GO_BUILD_DIR := ./cmd
GO_BINARY_NAME := $(PROJECT_NAME) # The name of the executable inside the Docker image

# 定义应用程序的名称和输出路径
# 这将是编译后的可执行文件的名称和它所在的目录
APP_NAME := model-promote-clone

# 定义 Go 源代码文件
# 默认是当前目录下的 main.go
# 如果你的入口文件是其他名称，请修改这里
SRC_FILE := main.go

# Go 编译器命令
GO := go

.PHONY: all build-image clean help

all: build-image ## Build the Docker image

build-image: ## Build the Docker image for the application
	@echo "--- Building Docker image: $(DOCKER_IMAGE) ---"
	docker build \
		-t $(DOCKER_IMAGE) \
		--build-arg GO_BUILD_DIR=. \
		--build-arg GO_BINARY_NAME=$(GO_BINARY_NAME) \
		-f Dockerfile .
	@echo "--- Docker image $(DOCKER_IMAGE) built successfully! ---"

image-clean: ## Remove previously built Docker image
	@echo "--- Cleaning up Docker image: $(DOCKER_IMAGE) ---"
	@docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "--- Clean up complete. ---"

build:
	@echo "Creating bin directory if it doesn't exist..."
	@mkdir -p bin
	@echo "Building $(APP_NAME)..."
	$(GO) build -o bin/$(APP_NAME) $(SRC_FILE)
	@echo "Build complete. Executable: ./$(APP_NAME)"

# run 目标：编译并运行应用程序
run: build
	@echo "Running $(APP_NAME)..."
	./$(APP_NAME)

# clean 目标：删除编译生成的可执行文件和 bin 目录
clean:
	@echo "Cleaning up..."
	@rm -f $(APP_NAME)
	@rm -rf bin # 删除整个 bin 目录
	@echo "Cleanup complete."