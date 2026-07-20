.PHONY: build test clean version help build-lecodec test-lecodec build-levad test-levad build-ledenoise test-ledenoise


# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
GO_VERSION ?= $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS = -ldflags "-X github.com/LingByte/lingllm/version.Version=$(VERSION) -X github.com/LingByte/lingllm/version.GitCommit=$(GIT_COMMIT) -X github.com/LingByte/lingllm/version.BuildTime=$(BUILD_TIME) -X github.com/LingByte/lingllm/version.GoVersion=$(GO_VERSION)"

help:
	@echo "LingLLM Build Commands"
	@echo "======================"
	@echo "make build          - Build the project"
	@echo "make test           - Run all tests"
	@echo "make test-coverage  - Run tests with coverage report"
	@echo "make test-asr        - Run ASR-tagged parser tests"
	@echo "make build-lecodec  - Build Rust liblecodec (encoder/lecodec)"
	@echo "make test-lecodec   - Build lecodec + run encoder LECodec tests"
	@echo "make build-levad    - Build Rust liblevad (vad/native/levad)"
	@echo "make test-levad     - Build levad + run vad/local Levad tests"
	@echo "make build-ledenoise - Build Rust libledenoise (media/native/ledenoise)"
	@echo "make test-ledenoise  - Build ledenoise + run media/denoise Ledenoise tests"
	@echo "make clean          - Clean build artifacts"
	@echo "make version        - Show version information"
	@echo "make fmt            - Format code with gofmt"
	@echo "make tools-demo     - Build tools demo"
	@echo "make batch-demo     - Build batch processing demo"

build:
	@echo "Building LingLLM..."
	@go build ./...
	@echo "✓ Build complete"

test:
	@echo "Running tests..."
	@go test ./... -v

test-ocr:
	@echo "Running OCR-tagged parser tests..."
	@go test ./parser/... -tags ocr -v -run 'OCR|Integration'

test-asr:
	@echo "Running ASR-tagged parser tests..."
	@go test ./parser/... -tags asr -v -run 'ASR|Integration'

build-lecodec:
	@echo "Building Rust lecodec (liblecodec)..."
	@./media/encoder/lecodec/build.sh

test-lecodec: build-lecodec
	@echo "Running lecodec encoder tests..."
	@CGO_ENABLED=1 go test -tags lingcodec ./media/encoder/ -run LECodec -v

build-levad:
	@echo "Building Rust levad (liblevad)..."
	@./vad/native/levad/build.sh

test-levad: build-levad
	@echo "Running levad VAD tests..."
	@CGO_ENABLED=1 go test -tags levad ./vad/local/ -run Levad -v

build-ledenoise:
	@echo "Building Rust ledenoise (libledenoise)..."
	@./media/native/ledenoise/build.sh

test-ledenoise: build-ledenoise
	@echo "Running ledenoise tests..."
	@CGO_ENABLED=1 go test -tags ledenoise ./media/denoise/ -run Ledenoise -v

test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -cover

clean:
	@echo "Cleaning build artifacts..."
	@go clean
	@rm -f tools-demo batch-demo
	@echo "✓ Clean complete"

version:
	@echo "LingLLM Version Information"
	@echo "============================"
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"

fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "✓ Format complete"

cli:
	@echo "Building CLI tool with version info..."
	@bash build.sh

tools-demo:
	@echo "Building tools-demo..."
	@go build -o tools-demo ./examples/tools-demo
	@echo "✓ tools-demo built"

batch-demo:
	@echo "Building batch-processing-demo..."
	@go build -o batch-demo ./examples/batch-processing-demo
	@echo "✓ batch-demo built"

# Development targets
dev-build: fmt test build
	@echo "✓ Development build complete"

# Release target
release: clean test build
	@echo "✓ Release build complete"
