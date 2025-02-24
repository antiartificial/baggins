.PHONY: all build test clean run docker-build docker-run docker-stop help

# Go parameters
BINARY_NAME=baggins
MAIN_PATH=cmd/server/main.go
DOCKER_IMAGE=baggins
DOCKER_TAG=latest

# Build directories
BUILD_DIR=build
UPLOADS_DIR=uploads
PROCESSED_DIR=processed

# Environment variables
export GO111MODULE=on

all: clean build test

help:
	@echo "Available targets:"
	@echo "  make build         - Build the application"
	@echo "  make test         - Run tests"
	@echo "  make run          - Run the application locally"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run application in Docker"
	@echo "  make docker-stop  - Stop Docker container"
	@echo "  make all          - Clean, build, and test"

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@mkdir -p $(UPLOADS_DIR)
	@mkdir -p $(PROCESSED_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	go clean
	@rm -f coverage.out

run: build
	@echo "Running application locally..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	@echo "Running Docker container..."
	docker run -d \
		--name $(BINARY_NAME) \
		-p 8080:8080 \
		-v $(PWD)/uploads:/app/uploads \
		-v $(PWD)/processed:/app/processed \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-stop:
	@echo "Stopping Docker container..."
	@docker stop $(BINARY_NAME) || true
	@docker rm $(BINARY_NAME) || true

# Development helpers
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	@echo "Running linter..."
	golangci-lint run

coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Default target
.DEFAULT_GOAL := help
