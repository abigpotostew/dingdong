#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
STATIC_DIR="$PROJECT_ROOT/internal/handlers/static"

SRC_FILE="$STATIC_DIR/tracker.src.js"
MIN_FILE="$STATIC_DIR/tracker.min.js"

# Check if esbuild is installed
if ! command -v esbuild &> /dev/null; then
    echo "esbuild not found. Installing..."
    npm install -g esbuild
fi

echo "Minifying tracker.js..."
esbuild "$SRC_FILE" --minify --outfile="$MIN_FILE"

# Show size comparison
SRC_SIZE=$(wc -c < "$SRC_FILE" | tr -d ' ')
MIN_SIZE=$(wc -c < "$MIN_FILE" | tr -d ' ')
SAVINGS=$((100 - (MIN_SIZE * 100 / SRC_SIZE)))

echo "Source:   $SRC_SIZE bytes"
echo "Minified: $MIN_SIZE bytes"
echo "Savings:  $SAVINGS%"
echo ""
echo "Generated: $MIN_FILE"
