#!/bin/bash

# CLI Worker Docker Runner
# Usage: ./scripts/run-cli-worker.sh [options]
#
# Examples:
#   ./scripts/run-cli-worker.sh --env prod --username dant --count 5
#   ./scripts/run-cli-worker.sh -e prod -u dant -c 5
#   ./scripts/run-cli-worker.sh --env test --username admin --level B2 --language spanish --topic math
#   ./scripts/run-cli-worker.sh -e test -u admin -l B2 -lang spanish -t math

set -e

# Function to show usage
show_usage() {
    echo "CLI Worker Docker Runner"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Required options:"
    echo "  -e, --env <env>        Environment: prod or test (default: prod)"
    echo "  -u, --username <name>  Username to generate questions for"
    echo ""
    echo "Optional options:"
    echo "  -c, --count <n>        Number of questions (default: 5)"
    echo "  -l, --level <level>    Override user's level (A1, A2, B1, B1+, B1++, B2, C1, C2)"
    echo "  -lang, --language <lang> Override user's language"
    echo "  -type, --type <type>   Question type: vocabulary, fill_blank, qa, reading_comprehension"
    echo "  -t, --topic <topic>    Specific topic for questions"
    echo "  -p, --ai-provider <provider> Override AI provider (ollama, openai, anthropic, google)"
    echo "  -m, --ai-model <model> Override AI model"
    echo "  -k, --ai-api-key <key> Override AI API key"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --env prod --username dant --count 10"
    echo "  $0 -e prod -u dant -c 10"
    echo "  $0 --env test --username admin --level B2 --language spanish --topic math"
    echo "  $0 -e test -u admin -l B2 -lang spanish -t math"
    echo "  $0 --env prod --username testuser --ai-provider ollama --ai-model llama4:latest"
    echo "  $0 -e prod -u testuser -p ollama -m llama4:latest"
    echo ""
    echo "Note: Make sure the Docker image is built with 'task build-cli-worker-docker'"
    echo "      and the specified environment is running"
}

# Function to convert short flags to long flags
convert_flags() {
    local args=()
    local i=1

    while [ $i -le $# ]; do
        local arg="${!i}"
        case "$arg" in
            -e)
                args+=("--env")
                ;;
            -u)
                args+=("--username")
                ;;
            -c)
                args+=("--count")
                ;;
            -l)
                args+=("--level")
                ;;
            -lang)
                args+=("--language")
                ;;
            -type)
                args+=("--type")
                ;;
            -t)
                args+=("--topic")
                ;;
            -p)
                args+=("--ai-provider")
                ;;
            -m)
                args+=("--ai-model")
                ;;
            -k)
                args+=("--ai-api-key")
                ;;
            -h)
                args+=("--help")
                ;;
            *)
                args+=("$arg")
                ;;
        esac
        ((i++))
    done

    echo "${args[@]}"
}

# Function to parse docker-compose configuration
parse_docker_config() {
    local env="$1"
    local compose_file=""

    if [ "$env" = "prod" ]; then
        compose_file="docker-compose.yml"
    elif [ "$env" = "test" ]; then
        compose_file="docker-compose.test.yml"
    else
        echo "Error: Invalid environment '$env'. Must be 'prod' or 'test'"
        exit 1
    fi

    if [ ! -f "$compose_file" ]; then
        echo "Error: Docker compose file '$compose_file' not found"
        exit 1
    fi

    # Extract project name and build network name
    local project_name=$(grep "^name:" "$compose_file" | head -1 | awk '{print $2}')
    if [ -z "$project_name" ]; then
        echo "Error: Could not find project name in $compose_file"
        exit 1
    fi
    local network_name="${project_name}_default"

    # Extract database URL based on environment
    local db_url=""
    if [ "$env" = "prod" ]; then
        db_url="postgres://quiz_user:quiz_password@postgres:5432/quiz_db?sslmode=disable"
    else
        db_url="postgres://quiz_user:quiz_password@postgres-test:5432/quiz_test_db?sslmode=disable"
    fi

    echo "$network_name:$db_url"
}

# Check if no arguments provided
if [ $# -eq 0 ]; then
    show_usage
    exit 1
fi

# Convert short flags to long flags
CONVERTED_ARGS=$(convert_flags "$@")

# Check for help flag
if [[ "$CONVERTED_ARGS" == *"--help"* ]]; then
    show_usage
    exit 0
fi

# Extract environment from arguments
ENV="prod"  # default
ENV_INDEX=-1
ARGS_ARRAY=($CONVERTED_ARGS)

for i in "${!ARGS_ARRAY[@]}"; do
    if [ "${ARGS_ARRAY[$i]}" = "--env" ] && [ $((i+1)) -lt ${#ARGS_ARRAY[@]} ]; then
        ENV="${ARGS_ARRAY[$((i+1))]}"
        ENV_INDEX=$i
        break
    fi
done

# Remove --env and its value from arguments for CLI worker
if [ $ENV_INDEX -ge 0 ]; then
    unset ARGS_ARRAY[$ENV_INDEX]
    unset ARGS_ARRAY[$((ENV_INDEX+1))]
    CONVERTED_ARGS="${ARGS_ARRAY[@]}"
fi

# Check if username is provided
if [[ "$CONVERTED_ARGS" != *"--username"* ]]; then
    echo "Error: --username flag is required"
    echo ""
    show_usage
    exit 1
fi

# Parse Docker configuration
DOCKER_CONFIG=$(parse_docker_config "$ENV")
NETWORK=$(echo "$DOCKER_CONFIG" | cut -d: -f1)
DATABASE_URL=$(echo "$DOCKER_CONFIG" | cut -d: -f2-)
DOCKER_IMAGE="quiz-cli-worker"

# Check if Docker image exists
if ! docker image inspect "$DOCKER_IMAGE" >/dev/null 2>&1; then
    echo "Error: Docker image '$DOCKER_IMAGE' not found"
    echo "Please build it first with: task build-cli-worker-docker"
    exit 1
fi

# Check if Docker network exists
if ! docker network inspect "$NETWORK" >/dev/null 2>&1; then
    echo "Error: Docker network '$NETWORK' not found"
    echo "Please start the $ENV environment first:"
    if [ "$ENV" = "prod" ]; then
        echo "  task start-prod"
    else
        echo "  task test-e2e-keep-running"
    fi
    exit 1
fi

echo "🚀 Running CLI worker in Docker..."
echo "Environment: $ENV"
echo "Image: $DOCKER_IMAGE"
echo "Network: $NETWORK"
echo "Database: $DATABASE_URL"
echo "Arguments: $CONVERTED_ARGS"
echo ""

# Run the CLI worker in Docker
docker run --rm \
    --network "$NETWORK" \
    -e DATABASE_URL="$DATABASE_URL" \
    "$DOCKER_IMAGE" $CONVERTED_ARGS

echo ""
echo "✅ CLI worker completed!"
