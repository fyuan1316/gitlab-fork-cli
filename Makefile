# Define variables
PROJECT_NAME := gitlab-fork-cli
VERSION ?= v0.1 # Default version is 'latest', can be overridden with make VERSION=v1.0
DOCKER_IMAGE := build-harbor.alauda.cn/test/$(PROJECT_NAME):$(VERSION)

# Define Go build-specific variables
GO_BUILD_DIR := ./cmd
GO_BINARY_NAME := $(PROJECT_NAME) # The name of the executable inside the Docker image

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

clean: ## Remove previously built Docker image
	@echo "--- Cleaning up Docker image: $(DOCKER_IMAGE) ---"
	@docker rmi $(DOCKER_IMAGE) 2>/dev/null || true
	@echo "--- Clean up complete. ---"