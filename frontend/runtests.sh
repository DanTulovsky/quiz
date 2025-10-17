#!/bin/bash
for file in $(find src -name "*.test.tsx" -o -name "*.test.ts" | sort); do
  if [ -f "$file" ]; then
    echo "========================================="
    echo "Testing file: $file"
    echo "========================================="
    npm test -- "$file" --reporter=verbose
    if [ $? -ne 0 ]; then
      echo ""
      echo "❌ FAILED in file: $file"
      break
    fi
    echo "✅ PASSED: $file"
    echo ""
  fi
done
