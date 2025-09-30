#!/bin/bash

# Default to current directory if no workspace root provided
if [ $# -eq 0 ]; then
    WORKSPACE_ROOT="$(pwd)"
    echo "Using current directory as workspace root: $WORKSPACE_ROOT"
else
    WORKSPACE_ROOT="$1"
fi
BACKEND_DIR="$WORKSPACE_ROOT/backend"

# Check if backend directory exists
if [ ! -d "$BACKEND_DIR" ]; then
    echo "Error: Backend directory not found at $BACKEND_DIR"
    exit 1
fi

cd "$BACKEND_DIR"

# Create a temporary directory for the merge operation
TEMP_DIR=$(mktemp -d /tmp/coverage-merge-XXXXXX)
echo "Using temporary directory: $TEMP_DIR"

# Copy coverage files to temp directory
cp coverage-unit.out "$TEMP_DIR/"
cp coverage-integration.out "$TEMP_DIR/"

cd "$TEMP_DIR"

# Use deduplication merge logic

echo "Using deduplication for coverage merging..."
echo "mode: atomic" > coverage-merged.out
{
    tail -n +2 coverage-unit.out
    tail -n +2 coverage-integration.out
} | sort -u >> coverage-merged.out

# Copy the merged result back to the backend directory
cp coverage-merged.out "$BACKEND_DIR/coverage.out"

# Remove intermediate coverage files from backend directory
cd "$BACKEND_DIR"
rm -f coverage-unit.out coverage-integration.out coverage-merged.out

# Clean up temporary directory
rm -rf $TEMP_DIR
echo "Temporary files cleaned up from: $TEMP_DIR"
