#!/bin/bash

# Artillery runner with default environment
# Usage: ./scripts/run-artillery.sh [environment] [config-file]

set -euo pipefail

# Set default environment if not provided
ENVIRONMENT=${1:-localhost}
CONFIG_FILE=${2:-artillery/config.yaml}

echo "Running Artillery with environment: $ENVIRONMENT"
echo "Config file: $CONFIG_FILE"

# Export the environment variable
export ENVIRONMENT="$ENVIRONMENT"

# Run artillery
artillery run --environment "$ENVIRONMENT" "$CONFIG_FILE"
