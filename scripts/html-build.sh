#!/bin/bash
# HTML Build Orchestration Script
# Executes the complete HTML build pipeline in the correct order:
# 1. templ generate - Generate Go code from templ templates
# 2. Go build - Build the generator binary
# 3. Frontend assets - UnoCSS extraction + esbuild bundle
# 4. HTML generation - Generate static HTML pages

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

# Preflight checks
check_dependency() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is not installed or not in PATH" >&2
        exit 1
    fi
}

check_go_tool() {
    if ! go tool "$1" version &> /dev/null; then
        echo "Error: go tool $1 is not available. Install with: go install github.com/a-h/templ/cmd/templ@latest" >&2
        exit 1
    fi
}

echo "=== HTML Build Orchestration ==="
echo "Project root: $PROJECT_ROOT"
echo ""

# Check required dependencies
echo "Checking dependencies..."
check_dependency go
check_dependency npm
check_go_tool templ
echo "      ✓ All dependencies available"
echo ""

# Step 1: Generate templ templates
echo "[1/4] Generating templ templates..."
go tool templ generate
echo "      ✓ templ templates generated"

# Step 2: Build Go binary
echo "[2/4] Building generator binary..."
mkdir -p bin
go build -o bin/generator ./cmd/generator
echo "      ✓ Generator binary built"

# Step 3: Build frontend assets (UnoCSS + esbuild)
echo "[3/4] Building frontend assets..."
npm run build
echo "      ✓ UnoCSS styles extracted to dist/styles.css"
echo "      ✓ esbuild bundled components to dist/components.js"

# Step 4: Generate HTML pages
echo "[4/4] Generating HTML pages..."
./bin/generator -content content -dist dist
echo "      ✓ HTML pages generated in dist/"

echo ""
echo "=== HTML build completed successfully ==="
echo ""
echo "Output files in dist/:"
ls -la dist/
