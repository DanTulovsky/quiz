#!/bin/bash

# Script to check for fmt.Errorf usage in production code
# This enforces the use of standardized error handling utilities

set -e

echo "🔍 Checking for fmt.Errorf usage in production code..."

# Find all Go files excluding test files and the errors utility file
# Exclude generated code and known utility files
FILES=$(find . -name "*.go" \
  -not -name "*_test.go" \
  -not -path "./backend/internal/utils/errors.go" \
  -not -path "./internal/utils/errors.go" \
  -not -path "./backend/internal/api/*" \
  -not -path "./internal/api/*")

# Check for fmt.Errorf usage
ERROR_COUNT=0
for file in $FILES; do
    if grep -q "fmt\.Errorf" "$file"; then
        echo "❌ Found fmt.Errorf usage in: $file"
        grep -n "fmt\.Errorf" "$file" | sed 's/^/  /'
        ERROR_COUNT=$((ERROR_COUNT + 1))
    fi
done

if [ $ERROR_COUNT -eq 0 ]; then
    echo "✅ No fmt.Errorf usage found in production code"
    echo "✅ All error handling follows the standardized pattern"
    exit 0
else
    echo ""
    echo "❌ Found $ERROR_COUNT file(s) with fmt.Errorf usage"
    echo ""
    echo "Please replace fmt.Errorf with one of the following:"
    echo "  - contextutils.WrapError(err, \"message\")"
    echo "  - contextutils.WrapErrorf(err, \"message with %s\", arg)"
    echo "  - contextutils.ErrorWithContextf(\"message with %s\", arg)"
    echo ""
    echo "This ensures consistent error handling across the codebase."
    exit 1
fi
