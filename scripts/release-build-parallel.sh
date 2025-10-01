#!/usr/bin/env bash
set -euo pipefail

# Parallel release build helper
# Usage: ./scripts/release-build-parallel.sh <TAG> <COMMIT_HASH> <BUILD_TIME>

TAG="${1:-latest}"
COMMIT_HASH="${2:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_TIME="${3:-$(date -u +"%Y-%m-%dT%H:%M:%SZ") }"
APP_VERSION="${4:-prod}"

# This script now delegates concurrency to docker buildx bake via docker-bake.hcl

echo "Starting buildx bake for tag: $TAG at commit: $COMMIT_HASH"

# Store current branch/commit for restoration later
ORIGINAL_REF=$(git symbolic-ref -q --short HEAD || git rev-parse --short HEAD || echo "unknown")

# Checkout to the specified commit
if [ "$COMMIT_HASH" != "unknown" ] && [ "$COMMIT_HASH" != "$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" ]; then
  echo "Checking out to commit: $COMMIT_HASH"
  git checkout "$COMMIT_HASH" || {
    echo "Failed to checkout commit $COMMIT_HASH" >&2
    exit 1
  }
fi

# Function to restore original ref
restore_original_ref() {
  if [ "$ORIGINAL_REF" != "unknown" ] && [ "$ORIGINAL_REF" != "$(git symbolic-ref -q --short HEAD || git rev-parse --short HEAD || echo 'unknown')" ]; then
    echo "Restoring original ref: $ORIGINAL_REF"
    git checkout "$ORIGINAL_REF" || echo "Warning: Failed to restore original ref $ORIGINAL_REF" >&2
  fi
}

# Set up trap to restore original ref on exit
trap restore_original_ref EXIT

# Use docker buildx bake (preferred) if available
if command -v docker >/dev/null 2>&1; then
  # Export variables for HCL to consume
  export APP_VERSION="$TAG"
  export COMMIT_HASH="$COMMIT_HASH"
  export BUILD_TIME="$BUILD_TIME"

  # Ensure bake file exists
  if [ ! -f docker-bake.hcl ]; then
    echo "docker-bake.hcl not found; aborting" >&2
    exit 1
  fi

  # Run bake with optimized settings for cross-compilation performance
  # Use registry cache for better cross-platform performance
  docker buildx bake -f docker-bake.hcl \
    --progress=auto \
    --push \
    --set "*.cache-from=type=registry,ref=mrwetsnow/quiz-backend:buildcache" \
    --set "*.cache-from=type=registry,ref=mrwetsnow/quiz-worker:buildcache" \
    --set "*.cache-from=type=registry,ref=mrwetsnow/quiz-frontend:buildcache" \
    --set "*.cache-to=type=registry,ref=mrwetsnow/quiz-backend:buildcache,mode=max" \
    --set "*.cache-to=type=registry,ref=mrwetsnow/quiz-worker:buildcache,mode=max" \
    --set "*.cache-to=type=registry,ref=mrwetsnow/quiz-frontend:buildcache,mode=max" \
    default || {
    echo "One or more bake targets failed" >&2
    exit 1
  }
else
  echo "docker not found; cannot run buildx bake" >&2
  exit 1
fi

echo "All images built and pushed successfully via bake"

# Restore original ref (trap handles this automatically, but being explicit)
restore_original_ref

exit 0


