#!/usr/bin/env bash
set -euo pipefail

# Create and push the next semantic git tag
# Usage: ./scripts/release-tag.sh [remote]

REMOTE="${1:-origin}"

# Determine next tag (allow override via NEW_TAG env)
if [ -z "${NEW_TAG:-}" ]; then
  NEW_TAG=$(./scripts/compute-next-release.sh)
fi

COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Creating git tag: $NEW_TAG"

# Ensure working tree is clean
if ! git diff --quiet || [ -n "$(git ls-files -m)" ]; then
  echo "Working tree is dirty. Commit or stash changes before tagging." >&2
  exit 1
fi

git tag "$NEW_TAG"
git push "$REMOTE" "$NEW_TAG"

echo "Tag $NEW_TAG pushed to $REMOTE"

exit 0

