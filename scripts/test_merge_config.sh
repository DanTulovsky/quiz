#!/usr/bin/env bash
# Test script for scripts/merge-config.sh
# Verifies that config merging works as expected
set -euo pipefail

ORIG_PWD="$(pwd)"
SCRIPT_PATH="$ORIG_PWD/${BASH_SOURCE[0]}"
SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_PATH")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MERGE_SCRIPT="$PROJECT_ROOT/scripts/merge-config.sh"

TMPDIR=$(mktemp -d)
cd "$TMPDIR"

# --- Simple merge test ---
cat >main.yaml <<EOF
foo: bar
nested:
  value: 1
  keep: yes
list:
  - a
  - b
EOF

cat >local.yaml <<EOF
foo: override
nested:
  value: 2
newkey: newval
list:
  - c
EOF

if [ ! -f "$MERGE_SCRIPT" ]; then
    echo "FAIL: merge-config.sh not found at $MERGE_SCRIPT"
    exit 1
fi

bash "$MERGE_SCRIPT" main.yaml local.yaml merged.yaml

grep -q 'foo: override' merged.yaml || {
    echo "FAIL: foo not overridden"
    exit 1
}
grep -q 'keep: yes' merged.yaml || {
    echo "FAIL: nested.keep not preserved"
    exit 1
}
grep -q 'newkey: newval' merged.yaml || {
    echo "FAIL: newkey not added"
    exit 1
}
grep -q 'value: 2' merged.yaml || {
    echo "FAIL: nested.value not overridden"
    exit 1
}
grep -q 'list:' merged.yaml && grep -q '  - c' merged.yaml || {
    echo "FAIL: list not overridden"
    exit 1
}
echo "PASS: simple merge-config.sh test"

# --- Realistic config.yaml/config.local.yaml merge test ---
cat >main.yaml <<EOF
providers:
  - name: Ollama
    code: ollama
    url: "http://localhost:11434/v1"
    supports_grammar: true
    question_batch_size: 1
    models:
      - name: "Llama 4"
        code: "llama4:latest"
        max_tokens: 8192
system:
  auth:
    signups_disabled: true
language_levels:
  italian:
    levels: ["A1", "A2"]
    descriptions:
      A1: "Beginner"
      A2: "Elementary"
EOF

cat >local.yaml <<EOF
system:
  auth:
    signups_disabled: false
providers:
  - name: LocalAI
    code: localai
    url: "http://localhost:1234/v1"
    supports_grammar: false
    question_batch_size: 2
    models:
      - name: "Test Model"
        code: "test-model"
        max_tokens: 1000
EOF

bash "$MERGE_SCRIPT" main.yaml local.yaml merged.yaml

# Check that system.auth.signups_disabled is overridden
if grep -A2 'system:' merged.yaml | grep -A1 'auth:' | grep -q 'signups_disabled: false'; then
    echo "PASS: system.auth.signups_disabled overridden"
else
    echo "FAIL: system.auth.signups_disabled not overridden"
    exit 1
fi
# Check that only the local provider is present (arrays are overridden, not merged)
if grep -q 'name: LocalAI' merged.yaml && ! grep -q 'name: Ollama' merged.yaml; then
    echo "PASS: providers array overridden correctly"
else
    echo "FAIL: providers not overridden correctly"
    exit 1
fi
# Check that language_levels is preserved
if grep -q 'italian:' merged.yaml && grep -q 'A1: "Beginner"' merged.yaml; then
    echo "PASS: language_levels preserved"
else
    echo "FAIL: language_levels not preserved"
    exit 1
fi

rm -rf "$TMPDIR"
