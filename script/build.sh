#!/bin/bash

# OpenTether Full Build Script
# Flow: clean frontend build -> validate embedded assets -> Go build
# Usage: ./build.sh [target]
# Targets: all|windows|linux|darwin|dev|help

set -euo pipefail

OUTPUT_DIR="output"
BINARY_NAME="opentether"
VERSION="${VERSION:-1.0.0}"
STATIC_SRC="admin-ui/build"
TARGET="${1:-all}"

print_header() {
    echo ""
    echo "=========================================="
    echo " OpenTether Build Script v${VERSION}"
    echo "=========================================="
    echo ""
}

print_help() {
    cat << EOF
Usage: ./build.sh [target]

Targets:
  all       Build Linux, Windows, and macOS amd64 binaries (default)
  windows   Build Windows amd64 binary
  linux     Build Linux amd64 binary
  darwin    Build macOS amd64 binary
  dev       Build dev binary from ./cmd (filesystem frontend mode)
  help      Show this help

Environment:
  VERSION   Version string injected into the binary (default: 1.0.0)

Flow:
  1. Clean admin-ui/build
  2. npm --prefix admin-ui run build
  3. Restore admin-ui/static/setup/index.html into admin-ui/build/setup/index.html
  4. Validate SvelteKit _app assets required by //go:embed all:admin-ui/build
  5. go build
EOF
}

validate_target() {
    case "$TARGET" in
        all|windows|linux|darwin|dev|help|--help|-h)
            ;;
        *)
            echo "[ERROR] Unknown target: $TARGET"
            echo "Valid targets: all, windows, linux, darwin, dev, help"
            exit 1
            ;;
    esac
}

build_frontend() {
    echo "[1/3] Building frontend..."

    if [ ! -f "admin-ui/package.json" ]; then
        echo "[ERROR] admin-ui/package.json not found; embedded builds require the frontend source"
        exit 1
    fi

    echo "  Cleaning ${STATIC_SRC}..."
    rm -rf "${STATIC_SRC}"

    if [ ! -d "admin-ui/node_modules" ]; then
        echo "  Installing dependencies..."
        npm --prefix admin-ui install --silent
    fi

    echo "  Building SvelteKit app..."
    npm --prefix admin-ui run build

    if [ -f "admin-ui/static/setup/index.html" ]; then
        mkdir -p "${STATIC_SRC}/setup"
        cp "admin-ui/static/setup/index.html" "${STATIC_SRC}/setup/index.html"
        echo "  Setup page restored"
    else
        echo "[ERROR] admin-ui/static/setup/index.html not found"
        exit 1
    fi

    echo "  Frontend built: ${STATIC_SRC}"
    echo ""
}

validate_frontend_build() {
    echo "[2/3] Validating embedded frontend assets..."

    required_files=(
        "${STATIC_SRC}/index.html"
        "${STATIC_SRC}/setup/index.html"
        "${STATIC_SRC}/_app/version.json"
    )

    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            echo "[ERROR] Required frontend file missing: $file"
            echo "        Run: npm --prefix admin-ui run build"
            exit 1
        fi
    done

    if ! find "${STATIC_SRC}/_app/immutable/entry" -maxdepth 1 -type f -name '*.js' | grep -q .; then
        echo "[ERROR] Missing SvelteKit entry JavaScript under ${STATIC_SRC}/_app/immutable/entry"
        echo "        The Go binary uses //go:embed all:admin-ui/build, so _app must exist before go build."
        exit 1
    fi

    node <<'NODE'
const fs = require('fs');
const path = require('path');

const root = path.join('admin-ui', 'build');
const html = fs.readFileSync(path.join(root, 'index.html'), 'utf8');
const assets = [...new Set([...html.matchAll(/\/admin\/(_app\/[^"'<>\s]+)/g)].map((match) => match[1]))].sort();

if (assets.length === 0) {
  console.error('[ERROR] index.html does not reference any /admin/_app assets');
  process.exit(1);
}

const missing = assets.filter((asset) => !fs.existsSync(path.join(root, asset)));
if (missing.length > 0) {
  for (const asset of missing) {
    console.error(`[ERROR] index.html references missing asset: /admin/${asset}`);
  }
  console.error('[ERROR] Frontend build validation failed; stale or incomplete SvelteKit assets');
  process.exit(1);
}
NODE

    echo "  Embedded asset validation passed"
    echo ""
}

go_build() {
    local os="$1"
    local arch="$2"
    local ext="$3"
    local out="${OUTPUT_DIR}/${BINARY_NAME}-${os}-${arch}${ext}"

    echo "  Building: ${out}"
    mkdir -p "$OUTPUT_DIR"
    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -ldflags "-X main.version=${VERSION}" -o "$out" .
}

build_go() {
    echo "[3/3] Building Go binary..."

    case "$TARGET" in
        all)
            go_build linux amd64 ""
            go_build windows amd64 ".exe"
            go_build darwin amd64 ""
            ;;
        linux)
            go_build linux amd64 ""
            ;;
        windows)
            go_build windows amd64 ".exe"
            ;;
        darwin)
            go_build darwin amd64 ""
            ;;
        dev)
            echo "  Building dev mode (filesystem frontend)..."
            mkdir -p "$OUTPUT_DIR"
            go build -ldflags "-X main.version=${VERSION}" -o "${OUTPUT_DIR}/${BINARY_NAME}-dev" ./cmd
            ;;
    esac
}

print_done() {
    echo ""
    echo "=========================================="
    echo " Build outputs in ${OUTPUT_DIR}/"
    echo "=========================================="
    ls -lh "${OUTPUT_DIR}/${BINARY_NAME}"* 2>/dev/null || true
    echo ""
    if [ "$TARGET" = "dev" ]; then
        echo "  NOTE: Dev binary serves admin-ui/build from filesystem"
        echo "  Run: ${OUTPUT_DIR}/${BINARY_NAME}-dev"
    else
        echo "  NOTE: Binary has admin-ui embedded via //go:embed all:admin-ui/build"
        echo "  Run: ${OUTPUT_DIR}/${BINARY_NAME}-linux-amd64"
    fi
}

validate_target

if [ "$TARGET" = "help" ] || [ "$TARGET" = "--help" ] || [ "$TARGET" = "-h" ]; then
    print_help
    exit 0
fi

print_header
build_frontend
validate_frontend_build
build_go
print_done
