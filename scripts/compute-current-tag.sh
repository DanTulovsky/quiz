#!/usr/bin/env bash
set -euo pipefail

# Compute the next semantic release tag and output shell assignments.
# Prints: NEW_TAG='vX.Y.Z'\nCOMMIT_HASH='...'\nBUILD_TIME='...'

LATEST_TAG=$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -n1 || true)

# Only print the next tag to stdout (caller may compute other metadata)
printf "%s" "$LATEST_TAG"
