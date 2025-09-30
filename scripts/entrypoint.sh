#!/usr/bin/env sh
set -e

BAKED="/app/merged.config.yaml"
LOCAL="/etc/quiz/config.local.yaml"
TMP="/app/merged.config.yaml.tmp"

if [ -f "$LOCAL" ]; then
  echo "Found local override $LOCAL — merging with baked $BAKED"
  /app/scripts/merge-config.sh "$BAKED" "$LOCAL" "$TMP"
  mv "$TMP" "$BAKED"
fi

# Decide which binary to run: worker or backend. If both exist, prefer worker when invoked in worker image.
if [ -x "/app/quiz-worker" ]; then
  exec /app/quiz-worker "$@"
else
  exec /app/quiz-app "$@"
fi


