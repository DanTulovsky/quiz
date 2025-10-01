#!/usr/bin/env bash
set -euo pipefail

# Compute the next semantic release tag and output shell assignments.
# Prints: NEW_TAG='vX.Y.Z'\nCOMMIT_HASH='...'\nBUILD_TIME='...'

LATEST_TAG=$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n1 || true)
if [ -z "$LATEST_TAG" ]; then
  NEW_TAG="v0.1.0"
else
  MAJOR=$(echo $LATEST_TAG | cut -d. -f1 | tr -d 'v')
  MINOR=$(echo $LATEST_TAG | cut -d. -f2)
  PATCH=$(echo $LATEST_TAG | cut -d. -f3)
  PATCH=$((PATCH + 1))
  NEW_TAG="v${MAJOR}.${MINOR}.${PATCH}"
fi

# Only print the next tag to stdout (caller may compute other metadata)
printf "%s" "$NEW_TAG"
