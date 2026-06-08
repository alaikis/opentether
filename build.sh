#!/bin/bash

# OpenTether Build Script
# Usage: ./build.sh [target]
#   target: all, linux, windows, darwin, current, help (default: all)

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
OUTPUT_DIR="output"
BINARY_NAME="opentether"
STATIC_SRC="admin-ui/build"
VERSION="${VERSION:-1.0.0}"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS="-X main.version=\${VERSION} -X main.buildTime=\${BUILD_TIME}"

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }

echo ""
log_info "OpenTether Build Script v${VERSION}"
log_info "================================="
echo ""

# ========================================
# Step 1: Verify admin-ui exists
# ========================================
log_info "Step 1: Checking admin-ui/build..."

if [ ! -d "$STATIC_SRC" ]; then
    log_error "$STATIC_SRC not found! Cannot build without admin UI."
    exit 1
fi
log_info "  admin-ui/build found"
echo ""

# ========================================
# Step 2: Build
# ========================================
TARGET="${1:-all}"

build_one() {
    local platform="$1"
    local arch="$2"
    local ext="$3"
    local out="${OUTPUT_DIR}/${BINARY_NAME}-${platform}-${arch}${ext}"

    log_info "  Building: ${out}"

    mkdir -p "$OUTPUT_DIR"
    GOOS="$platform" GOARCH="$arch" CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "$out" .
    log_info "    Done: ${out}"
}

copy_static() {
    echo ""
    log_info "Step 3: Copying admin-ui/build to output directory..."

    local dest="${OUTPUT_DIR}/admin-ui/build"

    # Remove old static files
    rm -rf "$dest"

    # Copy admin-ui/build recursively
    cp -r "$STATIC_SRC" "$dest"
    log_info "  Static files copied to ${dest}"
}

show_output() {
    echo ""
    log_info "Build outputs:"
    ls -lh "${OUTPUT_DIR}/${BINARY_NAME}"* 2>/dev/null || true
    echo ""
    log_info "Directory structure:"
    log_info "  ${OUTPUT_DIR}/"
    log_info "    +-- ${BINARY_NAME}-linux-amd64"
    log_info "    +-- ${BINARY_NAME}-windows-amd64.exe"
    log_info "    +-- ${BINARY_NAME}-darwin-amd64"
    log_info "    +-- admin-ui/build/     (static web files)"
    echo ""
    log_info "Run: cd ${OUTPUT_DIR} && ./${BINARY_NAME}-linux-amd64"
}

case "$TARGET" in
    all)
        log_info "Step 2: Building all platforms..."
        build_one linux amd64 ""
        build_one windows amd64 ".exe"
        build_one darwin amd64 ""
        copy_static
        echo ""
        log_info "All builds completed!"
        show_output
        ;;
    linux)
        log_info "Step 2: Building Linux amd64..."
        build_one linux amd64 ""
        copy_static
        show_output
        ;;
    windows)
        log_info "Step 2: Building Windows amd64..."
        build_one windows amd64 ".exe"
        copy_static
        show_output
        ;;
    darwin)
        log_info "Step 2: Building Darwin (macOS) amd64..."
        build_one darwin amd64 ""
        copy_static
        show_output
        ;;
    current)
        local goos=$(go env GOOS)
        local goarch=$(go env GOARCH)
        log_info "Step 2: Building for current platform: ${goos}/${goarch}..."
        local ext=""
        [ "$goos" = "windows" ] && ext=".exe"
        build_one "$goos" "$goarch" "$ext"
        copy_static
        show_output
        ;;
    help|--help|-h)
        cat << EOF

OpenTether Build Script
=======================

Usage: ./build.sh [target]

Targets:
  all       - Build for all platforms + copy static files (default)
  linux     - Build for Linux amd64 + copy static files
  windows   - Build for Windows amd64 + copy static files
  darwin    - Build for macOS amd64 + copy static files
  current   - Build for current platform + copy static files
  help      - Show this help

Examples:
  ./build.sh              # Build all platforms
  ./build.sh windows      # Build Windows only

Note: Static files (admin-ui/build) are automatically copied to
      the output directory after build. Run the executable from
      the output directory to serve the admin UI.
EOF
        exit 0
        ;;
    *)
        log_error "Unknown target: $TARGET"
        log_error "Valid targets: all, linux, windows, darwin, current, help"
        exit 1
        ;;
esac
