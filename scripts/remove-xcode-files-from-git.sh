#!/bin/bash

# Remove Xcode files and directories from git tracking
# ====================================================
# This script removes Xcode-related files and directories from git tracking
# while keeping them in the local filesystem. It handles all patterns defined
# in the .gitignore Xcode section.
#
# Usage:
#   ./scripts/remove-xcode-files-from-git.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    log_error "Not in a git repository"
    exit 1
fi

log_info "Finding Xcode files tracked by git..."

# Get all tracked files and filter for Xcode patterns
ALL_TRACKED_FILES=$(git ls-files)

# Array to store files to remove
FILES_TO_REMOVE=()

# Process each tracked file
while IFS= read -r file; do
    if [[ -z "$file" ]]; then
        continue
    fi

    # Check file extension patterns
    if [[ "$file" =~ \.(xcuserstate|xcscmblueprint|hmap|ipa|dSYM\.zip|dSYM|swiftmodule|swiftdoc|swiftsourceinfo|app|framework)$ ]]; then
        FILES_TO_REMOVE+=("$file")
        continue
    fi

    # Check directory patterns
    if [[ "$file" =~ ^(.*/)?(xcuserdata|DerivedData)/ ]] || [[ "$file" =~ ^ios/build/ ]]; then
        FILES_TO_REMOVE+=("$file")
        continue
    fi

    # Check path patterns
    if [[ "$file" =~ \.xcworkspace/xcuserdata/ ]] || \
       [[ "$file" =~ \.xcodeproj/xcuserdata/ ]] || \
       [[ "$file" =~ \.xcodeproj/project\.xcworkspace/xcuserdata/ ]]; then
        FILES_TO_REMOVE+=("$file")
        continue
    fi

done <<< "$ALL_TRACKED_FILES"

# Remove duplicates and sort
IFS=$'\n' FILES_TO_REMOVE=($(printf '%s\n' "${FILES_TO_REMOVE[@]}" | sort -u))

if [ ${#FILES_TO_REMOVE[@]} -eq 0 ]; then
    log_success "No Xcode files found in git tracking"
    exit 0
fi

log_info "Found ${#FILES_TO_REMOVE[@]} file(s)/directory(ies) to remove from git:"
for file in "${FILES_TO_REMOVE[@]}"; do
    echo "  - $file"
done

echo ""
read -p "Remove these files from git tracking? (y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warning "Aborted by user"
    exit 0
fi

# Optimize: if a directory is in the list, remove child files from the list
# to avoid redundant operations
OPTIMIZED_FILES=()
for file in "${FILES_TO_REMOVE[@]}"; do
    IS_CHILD=false
    for other_file in "${FILES_TO_REMOVE[@]}"; do
        if [[ "$file" != "$other_file" ]] && [[ "$file" =~ ^"$other_file"/ ]]; then
            IS_CHILD=true
            break
        fi
    done
    if [[ "$IS_CHILD" == "false" ]]; then
        OPTIMIZED_FILES+=("$file")
    fi
done

FILES_TO_REMOVE=("${OPTIMIZED_FILES[@]}")

# Remove files from git tracking
REMOVED_COUNT=0
FAILED_COUNT=0

for file in "${FILES_TO_REMOVE[@]}"; do
    if git rm --cached -r "$file" 2>/dev/null; then
        REMOVED_COUNT=$((REMOVED_COUNT + 1))
    else
        log_warning "Failed to remove: $file"
        FAILED_COUNT=$((FAILED_COUNT + 1))
    fi
done

echo ""
if [ $FAILED_COUNT -eq 0 ]; then
    log_success "Removed $REMOVED_COUNT file(s)/directory(ies) from git tracking"
    log_info "Files remain in your local filesystem"
    log_info "Run 'git status' to see the changes"
else
    log_warning "Removed $REMOVED_COUNT file(s), but $FAILED_COUNT failed"
    exit 1
fi

