# CLI Worker

Generate questions for a specific user using the quiz application's AI worker.

## Quick Start

```bash
# Local execution
cd backend && go run ./cmd/cli-worker --username dant --count 5

# Docker execution
docker run --rm --network quiz-prod_default \
  -e DATABASE_URL=postgres://quiz_user:quiz_password@postgres:5432/quiz_db?sslmode=disable \
  quiz-cli-worker --username dant --count 5
```

## Usage

```bash
./cli-worker --username <username> [options]
```

### Required Flags
- `--username <name>` - Username to generate questions for

### Optional Flags
- `--count <n>` - Number of questions (default: 5)
- `--level <level>` - Override user's level (A1, A2, B1, B1+, B1++, B2, C1, C2)
- `--language <lang>` - Override user's language
- `--type <type>` - Question type: vocabulary, fill_blank, qa, reading_comprehension
- `--topic <topic>` - Specific topic for questions
- `--ai-provider <provider>` - Override AI provider (ollama, openai, anthropic, google)
- `--ai-model <model>` - Override AI model
- `--ai-api-key <key>` - Override AI API key

## Examples

```bash
# Basic usage
./cli-worker --username dant --count 10

# Override settings
./cli-worker --username admin --level B2 --language spanish --topic math

# Use different AI provider
./cli-worker --username testuser --ai-provider ollama --ai-model llama4:latest

# Override everything
./cli-worker --username admin --ai-provider openai --ai-model gpt-4.1 --ai-api-key sk-... --count 3
```

## Taskfile Commands

```bash
# Build and run locally
task build-cli-worker
task run-cli-worker CLI_ARGS="--username dant --count 5"

# Build and run in Docker
task build-cli-worker-docker
task run-cli-worker-docker CLI_ARGS="--username dant --count 5"

# Show help
task cli-worker-help
```

## Notes

- Uses user's AI settings by default
- Can override any AI setting via command line flags
- Validates all parameters against configuration
- Requires database access (local or Docker network)
- Gracefully handles AI service unavailability
