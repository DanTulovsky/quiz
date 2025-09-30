#!/usr/bin/env sh
set -e

MAIN_CONFIG=${1:-config.yaml}
LOCAL_CONFIG=${2:-config.local.yaml}
MERGED_CONFIG=${3:-merged.config.yaml}

if [ -f "$LOCAL_CONFIG" ]; then
    yq eval-all 'select(fileIndex == 0) * select(fileIndex == 1)' "$MAIN_CONFIG" "$LOCAL_CONFIG" >"$MERGED_CONFIG"
else
    cp "$MAIN_CONFIG" "$MERGED_CONFIG"
fi
