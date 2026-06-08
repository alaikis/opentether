#!/bin/bash

# OpenTether Full Build Script
# Step 1: Build frontend (npm run build)
# Step 2: Go build with embedded admin-ui (single binary)
# Usage: ./build.sh [target]   target: all|windows|linux|darwin|dev|help

set -e

OUTPUT_DIR="output"
BINARY_NAME="opentether"
VERSION="${VERSION:-1.0.0}"
STATIC_SRC="admin-ui/build"

echo ""
echo "=========================================="
echo " OpenTether Build Script v${VERSION}"
echo "=========================================="
echo ""

# ========================================
# Step 1: Build Frontend
# ========================================
echo "[1/3] Building frontend..."

if [ -f "admin-ui/package.json" ]; then
    cd admin-ui

    if [ ! -d "node_modules" ]; then
        echo "  Installing dependencies..."
        npm install --silent
    fi

    echo "  Building SvelteKit app..."
    npm run build
    cd ..
    echo "  Frontend built: ${STATIC_SRC}"

    # Restore setup wizard page (served at /setup, separate from SvelteKit SPA)
    if [ -f "admin-ui/static/setup/index.html" ]; then
        mkdir -p "${STATIC_SRC}/setup"
        cp "admin-ui/static/setup/index.html" "${STATIC_SRC}/setup/index.html"
        echo "  Setup page restored"
    fi
else
    echo "  [WARN] admin-ui/package.json not found, skipping frontend build"
fi

echo ""

# Verify build exists
if [ ! -f "${STATIC_SRC}/index.html" ]; then
    echo "[ERROR] ${STATIC_SRC}/index.html not found! Run: cd admin-ui && npm run build"
    exit 1
fi

# ========================================
# Step 2: Go Build
# ========================================
TARGET="${1:-all}"

go_build() {
    local os="$1"
    local arch="$2"
    local ext="$3"
    local out="${OUTPUT_DIR}/${BINARY_NAME}-${os}-${arch}${ext}"

    echo "  Building: ${out}"
    mkdir -p "$OUTPUT_DIR"
    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o "$out" .
}

echo "[2/3] Building Go binary (embedded mode)..."

case "$TARGET" in
    all)
        go_build linux amd64 ""
        go_build windows amd64 ".exe"
        go_build darwin amd64 ""
        echo ""
        echo "[3/3] All builds completed!"
        ;;
    linux)
        go_build linux amd64 ""
        echo "[3/3] Build completed!"
        ;;
    windows)
        go_build windows amd64 ".exe"
        echo "[3/3] Build completed!"
        ;;
    darwin)
        go_build darwin amd64 ""
        echo "[3/3] Build completed!"
        ;;
    dev)
        echo "  Building dev mode (cmd entry, filesystem)..."
        mkdir -p "$OUTPUT_DIR"
        go build -ldflags "-X main.version=${VERSION}" -o "${OUTPUT_DIR}/${BINARY_NAME}-dev" ./cmd
        echo "[3/3] Dev build completed!"
        echo "  Run: ${OUTPUT_DIR}/${BINARY_NAME}-dev"
        exit 0
        ;;
    help|--help|-h)
        cat << EOF

Usage: ./build.sh [target]

Targets:
  all       Build all platforms (default)
  windows   Build Windows amd64
  linux     Build Linux amd64
  darwin    Build macOS amd64
  dev       Build dev mode (filesystem, no embed)
  help      Show this help

Flow: npm build (admin-ui) -> go build (embed) -> single binary
EOF
        exit 0
        ;;
    *)
        echo "[ERROR] Unknown target: $TARGET"
        echo "Valid targets: all, windows, linux, darwin, dev, help"
        exit 1
        ;;
esac

# ========================================
# Done
# ========================================
echo ""
echo "=========================================="
echo " Build outputs in ${OUTPUT_DIR}/"
echo "=========================================="
ls -lh "${OUTPUT_DIR}/${BINARY_NAME}"* 2>/dev/null || true
echo ""
echo "  NOTE: Binary has admin-ui embedded (no external files needed)"
echo "  Run: cd ${OUTPUT_DIR} && ./${BINARY_NAME}-linux-amd64"
