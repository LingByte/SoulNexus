#!/bin/bash

# Build script for LingLLM with version information injection

set -e

# Get version information
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "unknown")
GO_VERSION=$(go version | awk '{print $3}')

# Replace spaces with underscores in BUILD_TIME for ldflags compatibility
BUILD_TIME_SAFE=$(echo "$BUILD_TIME" | tr ' ' '_')

echo "Building LingLLM..."
echo "Version:    $VERSION"
echo "Commit:     $GIT_COMMIT"
echo "Build Time: $BUILD_TIME"
echo "Go Version: $GO_VERSION"
echo ""

# Build the CLI tool with ldflags
# Note: Use = syntax for ldflags and replace spaces with underscores
go build \
  "-ldflags=-X=github.com/LingByte/lingllm/version.Version=$VERSION -X=github.com/LingByte/lingllm/version.GitCommit=$GIT_COMMIT -X=github.com/LingByte/lingllm/version.BuildTime=$BUILD_TIME_SAFE -X=github.com/LingByte/lingllm/version.GoVersion=$GO_VERSION" \
  -o lingllm ./cmd/lingllm

echo "✓ Build complete: ./lingllm"
echo ""
echo "Usage:"
echo "  ./lingllm -v       # Show version"
echo "  ./lingllm -version # Show full version info"
