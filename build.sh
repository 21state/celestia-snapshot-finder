#!/bin/bash

set -e

VERSION=$(grep -r "const Version = " internal/version/version.go | cut -d'"' -f2)

echo "Downloading dependencies..."
go mod tidy
mkdir -p build

build() {
    local GOOS=$1
    local GOARCH=$2
    local SUFFIX=$3
    
    echo "Building for $GOOS/$GOARCH..."
    
    local BINARY="celestia-snapshot-finder"
    if [ ! -z "$SUFFIX" ]; then
        BINARY="${BINARY}${SUFFIX}"
    fi
    
    GOOS=$GOOS GOARCH=$GOARCH go build -o "build/${BINARY}-${GOOS}-${GOARCH}${SUFFIX}" -ldflags "-X main.Version=${VERSION}"
}

echo "Cleaning build directory..."
rm -rf build/*

build "linux" "amd64" ""           # Linux AMD64
build "linux" "arm64" ""           # Linux ARM64
build "darwin" "amd64" ""          # macOS AMD64
build "darwin" "arm64" ""          # macOS ARM64

echo "Generating checksums..."
cd build
shasum -a 256 * > checksums.txt
cd ..

echo "Build complete! Binaries are in the build directory:"
ls -lh build/
