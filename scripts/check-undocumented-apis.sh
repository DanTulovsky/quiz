#!/bin/bash

# Script to check for undocumented APIs
# This script compares the endpoints defined in the router with those documented in swagger.yaml

echo "🔍 Checking for undocumented APIs..."

# Use the Go script for more accurate endpoint extraction
if command -v go &> /dev/null; then
    go run scripts/check-undocumented-apis.go
else
    echo "❌ Go is not available. Please install Go to use this script."
    echo "Alternatively, you can run: go run scripts/check-undocumented-apis.go"
    exit 1
fi
