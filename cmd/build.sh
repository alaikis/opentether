#!/bin/bash

# OpenTether Build Script
# Builds both frontend and backend, embeds frontend into binary

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  OpenTether Build Script${NC}"
echo -e "${GREEN}========================================${NC}"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# Parse arguments
BUILD_MODE="full"  # default: full build
SKIP_FRONTEND=false
SKIP_BACKEND=false
OUTPUT_NAME="wisehoof"

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-frontend)
            SKIP_FRONTEND=true
            shift
            ;;
        --skip-backend)
            SKIP_BACKEND=true
            shift
            ;;
        --output)
            OUTPUT_NAME="$2"
            shift 2
            ;;
        --dev)
            BUILD_MODE="dev"
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --skip-frontend    Skip frontend build"
            echo "  --skip-backend     Skip backend build"
            echo "  --output NAME      Output binary name (default: wisehoof)"
            echo "  --dev              Development mode (keep local files)"
            echo "  -h, --help         Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo ""
echo -e "${YELLOW}Project root: ${PROJECT_ROOT}${NC}"
echo -e "${YELLOW}Build mode: ${BUILD_MODE}${NC}"
echo ""

# Step 1: Build Frontend
if [ "$SKIP_FRONTEND" = false ]; then
    echo -e "${GREEN}[1/2] Building frontend...${NC}"

    if [ ! -d "admin-ui" ]; then
        echo -e "${YELLOW}Warning: admin-ui directory not found, skipping frontend build${NC}"
    else
        cd admin-ui

        # Check if node_modules exists
        if [ ! -d "node_modules" ]; then
            echo "Installing Node.js dependencies..."
            npm install
        fi

        # Build the frontend
        echo "Running npm run build..."
        npm run build

        cd "$PROJECT_ROOT"

        if [ -d "admin-ui/build" ]; then
            echo -e "${GREEN}Frontend built successfully!${NC}"
        else
            echo -e "${RED}Frontend build failed - build directory not found${NC}"
            exit 1
        fi
    fi
else
    echo -e "${YELLOW}[1/2] Skipping frontend build${NC}"
fi

# Step 2: Build Backend
if [ "$SKIP_BACKEND" = false ]; then
    echo -e "${GREEN}[2/2] Building backend...${NC}"

    # Create output directory
    mkdir -p output

    if [ "$BUILD_MODE" = "dev" ]; then
        # Development build - use main.go directly
        echo "Development build (using main.go)..."
        go build -o "output/${OUTPUT_NAME}" .
    else
        # Production build - use cmd/main.go with embedded frontend
        echo "Production build (embedding frontend)..."

        # Verify frontend exists
        if [ ! -d "admin-ui/build" ] || [ -z "$(ls -A admin-ui/build 2>/dev/null)" ]; then
            echo -e "${RED}Error: admin-ui/build is empty or not found${NC}"
            echo -e "${RED}Please run with frontend build first or use --skip-frontend=false${NC}"
            exit 1
        fi

        # Build with embedded frontend
        go build -o "output/${OUTPUT_NAME}" ./cmd
    fi

    if [ -f "output/${OUTPUT_NAME}" ]; then
        echo -e "${GREEN}Backend built successfully!${NC}"
    else
        echo -e "${RED}Backend build failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}[2/2] Skipping backend build${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Build Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Output: ${PROJECT_ROOT}/output/${OUTPUT_NAME}"
echo ""

# Show file info
if [ -f "output/${OUTPUT_NAME}" ]; then
    echo "Binary info:"
    ls -lh "output/${OUTPUT_NAME}"
fi
