#!/usr/bin/env bash
set -euo pipefail

# Downloads Punkt sentence tokenizer models for languages defined in merged.config.yaml.
# Requires: yq, curl
#
# Model source (JSON): neurosnap/sentences
# Raw URL template: https://raw.githubusercontent.com/neurosnap/sentences/master/data/{name}.json
#
# Mapping from language code (config.yaml language_levels.*.code) to model file name
# en->english, it->italian, fr->french, de->german, es->spanish, ru->russian,
# hi->hindi, ja->japanese, zh->chinese

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_FILE="${QUIZ_CONFIG_FILE:-${ROOT_DIR}/merged.config.yaml}"
DEST_DIR="${ROOT_DIR}/backend/internal/resources/punkt"
BASE_URL="https://raw.githubusercontent.com/neurosnap/sentences/master/data"

if ! command -v yq >/dev/null 2>&1; then
  echo "❌ yq is required. Please install yq."
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "❌ curl is required. Please install curl."
  exit 1
fi

mkdir -p "${DEST_DIR}"

# Extract language codes from config
mapfile -t CODES < <(yq -r '.language_levels | to_entries | .[].value.code' "${CONFIG_FILE}")

code_to_name() {
  case "$1" in
    en) echo "english" ;;
    it) echo "italian" ;;
    fr) echo "french" ;;
    de) echo "german" ;;
    es) echo "spanish" ;;
    ru) echo "russian" ;;
    hi) echo "hindi" ;;
    ja) echo "japanese" ;;
    zh) echo "chinese" ;;
    *) echo "" ;;
  esac
}

updated=0
for code in "${CODES[@]}"; do
  name="$(code_to_name "${code}")"
  if [[ -z "${name}" ]]; then
    echo "⚠️  No Punkt model mapping for code '${code}'. Skipping (regex fallback will be used)."
    continue
  fi
  url="${BASE_URL}/${name}.json"
  dest="${DEST_DIR}/${name}.json"
  echo "⬇️  Downloading ${name} model (${code}) from ${url}"
  if curl -fsSL "${url}" -o "${dest}.tmp"; then
    mv "${dest}.tmp" "${dest}"
    echo "✅ Saved ${dest}"
    updated=$((updated+1))
  else
    echo "⚠️  Model not available for ${name} (${code}) at ${url}"
    echo "   This language will use regex-based sentence extraction (fallback)."
    echo "   To fix: check if the model exists in neurosnap/sentences repository."
    rm -f "${dest}.tmp" || true
  fi
done

echo "Done. Updated ${updated} model file(s) in ${DEST_DIR}."


