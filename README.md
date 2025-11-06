# AI-Powered Quiz Application

An adaptive learning quiz platform that uses AI to generate personalized questions and provide feedback. The application learns from your performance and adapts difficulty to help you improve in weak areas.

## 🌟 Features

- 🧠 **Adaptive Learning**: Adjusts difficulty based on performance and prioritizes weak areas
- 🌍 **Multi-Language Support**: Starting with Italian, expandable to other languages
- 📊 **Performance Tracking**: Comprehensive analytics and progress visualization
- 🎯 **Multiple Question Types**: Multiple choice, fill-in-blank, Q&A, reading comprehension
- 🔄 **Smart Caching**: Pre-generates questions to minimize AI API calls
- 👥 **Multi-User Support**: Individual profiles with per-user question generation
- 🎨 **Modern UI**: Responsive design with smooth animations
- 🏆 **CEFR Levels**: Support for A1-C2 language proficiency levels
- 💡 **AI Explanations**: Detailed explanations for incorrect answers
- 🤖 **Per-User AI Models**: Users can select different AI providers and models
- ⚡ **AI Concurrency Control**: Smart request limiting to prevent AI service overload

## 🏗️ Architecture

### Services

- **Backend** (Port 8080): User-facing API, authentication, quiz logic
- **Worker** (Port 8081): Background AI question generation with user-specific settings
- **Frontend** (Port 3000): React application with TypeScript
- **Database**: PostgreSQL with automated migrations

### Tech Stack

- **Backend**: Go with Gin framework
- **Frontend**: React 18 with TypeScript and Tailwind CSS
- **Database**: PostgreSQL with go-admin database explorer
- **AI Integration**: OpenAI-compatible API (supports Ollama, OpenAI, Anthropic)
- **Containerization**: Docker with docker-compose
- **Testing**: Comprehensive test suite (unit, integration, E2E)

### Service Architecture

```text
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │    Backend      │    │     Worker      │
│   (React)       │◄──►│   (Port 8080)   │◄──►│   (Port 8081)   │
│                 │    │                 │    │                 │
│ - Quiz UI       │    │ - User API      │    │ - AI Generation │
│ - Settings      │    │ - Auth          │    │ - Quiz Logic    │
│ - Admin Panel   │    │ - Quiz Logic    │    │ - Worker Admin  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                └───────┬───────────────┘
                                        │
                                ┌─────────────────┐
                                │   PostgreSQL    │
                                │   Database      │
                                │                 │
                                │ - Users         │
                                │ - Questions     │
                                │ - Responses     │
                                │ - Worker Status │
                                └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose
- **Ollama with deepseek-r1 model** (recommended)
- OR OpenAI API key (alternative)

### 1. Setup Ollama (Recommended)

```bash
# Install Ollama
brew install ollama  # macOS
# Or download from https://ollama.ai

# Pull and run deepseek-r1 model
ollama pull deepseek-r1
ollama serve
```

### 2. Clone and Run

```bash
git clone <your-repo-url>
cd quiz

# Start all services
task start-prod
```

### 3. Access the Application

1. Open `http://localhost:3000`
2. Login with:
   - **Username**: `admin`
   - **Password**: `admin` (default, change in production)
3. Start learning!

## 📖 Usage

### Getting Started

1. **Login**: Use admin credentials
2. **Set Preferences**: Choose language and CEFR level
3. **Start Quiz**: Answer personalized questions
4. **Review Progress**: Check analytics and weak areas
5. **Adjust Settings**: Update level as you improve

### Question Types

- **Multiple Choice**: Select from 4 options
- **Fill in the Blank**: Complete sentences
- **Question & Answer**: Written responses
- **Reading Comprehension**: Answer based on text passages

### Adaptive Learning

The system tracks performance and:

- Adjusts difficulty based on accuracy
- Prioritizes weak areas
- Avoids repeating recent correct answers
- Suggests level changes when appropriate

## 🔧 Configuration

### Environment Variables

Create `.env` file from `.env.example`:

```bash
# Server Configuration
PORT=8080
ADMIN_PASSWORD=change_this_in_production

# Database
DATABASE_URL=postgres://quiz_user:quiz_password@postgres:5432/quiz_db?sslmode=disable

# Worker
START_WORKER_PAUSED=false
```

### Admin Tools

- **Backend Admin** (`http://localhost:8080/adminz`): User stats, analytics, and AI concurrency monitoring
- **Worker Admin** (`http://localhost:8081/adminz`): Worker status, controls, and cross-service AI metrics
- **Database Explorer** (`http://localhost:8080/db-admin/login`): Direct database management

### AI Concurrency Control

The system includes intelligent request limiting to prevent AI service overload:

- **Global Limits**: Maximum concurrent AI requests across all users (default: 10)
- **Per-User Limits**: Maximum concurrent requests per individual user (default: 2)
- **Real-Time Monitoring**: Live stats on both admin dashboards
- **Fail-Fast Behavior**: Clear error messages when limits exceeded (no queuing delays)
- **API Endpoint**: Programmatic access via `/admin/ai-concurrency`

Configure limits via environment variables:

```bash
MAX_AI_CONCURRENT=10  # Global concurrent request limit
MAX_AI_PER_USER=2     # Per-user concurrent request limit
```

## 🛠️ Development

### Key Commands

```bash
# Start services
task start-prod
task restart-prod

# Run tests
task test
task test-go-unit
task test-go-integration

# Linting
task lint

# Database
task setup-test-db
task reset-prod-db

# Load Testing & Security Testing
task test-artillery-run TEST_NAME=login-test
task test-artillery-run TEST_NAME=login-fuzzer-test
task test-artillery-all
```

### Project Structure

```text
quiz/
├── backend/              # Go backend application
│   ├── cmd/
│   │   ├── server/      # Main server (port 8080)
│   │   └── worker/      # Worker service (port 8081)
│   ├── internal/        # Application code
│   └── migrations/      # Database migrations
├── frontend/            # React frontend
│   ├── src/
│   └── tests/
├── artillery/           # Load testing & security testing
│   ├── tests/           # Artillery test scenarios
│   └── README.md        # Artillery documentation
├── docker-compose.yml   # Production services
└── Taskfile.yml        # Build tasks
```

## 🚀 Deployment

### Production Deployment

#### Automatic
Run

```bash
task release-tag
```

This will create and push a new tag.  Github Actions will build and then ssh into `vm1` to deploy.

#### Manual

```bash
task restart-prod
```

### Manual Setup

1. Configure environment variables
2. Set up PostgreSQL database
3. Build and run backend and worker services
4. Build and serve frontend
5. Configure reverse proxy (nginx) for production

### CSP Nonce & Font Self-Hosting Automation

- **Every frontend build** (for production or tests) automatically injects a secure nonce into all `<script>` and `<style>` tags in the built HTML, and into `nginx.conf` for the CSP header.
- **Google Fonts are now fully self-hosted and automated**: The build process downloads the required font files and generates the correct CSS. No external font CDN is used in production.
- **This process is fully automated inside the Docker build** (see `Dockerfile.frontend`).
- You do **not** need to run any font or nonce scripts on the host; any `docker build` or `docker compose up --build` will always produce a correct image with all fonts and CSP nonces in place.
- This is CI/CD and Docker Compose safe.
- If you see CSP or font errors, rebuild the Docker image or rerun the deploy task.

## 🔐 Security

- Session-based authentication with secure cookies
- Input validation and sanitization
- SQL injection prevention
- XSS protection
- AI request concurrency limits to prevent service overload
- Rate limiting for AI API calls

## 📝 License

MIT License

## 🆘 Support

If you encounter issues:

1. Check service logs: `docker-compose logs`
2. Verify Ollama is running: `ollama list`
3. Check database connectivity
4. Ensure all ports are available

## Configuration System

The application uses a YAML config file (`config.yaml`) for all AI provider and system settings. You can override any setting by creating a `config.local.yaml` file in the same directory. The override file will be deep-merged with the main config, so you can override individual keys (e.g., just `signups_disabled`) without duplicating the entire config.

### Environment Variable

You can specify a custom config file path by setting the `QUIZ_CONFIG_FILE` environment variable. If not set, the app looks for `config.yaml` in the executable's directory or the current directory.

### Example: Overriding a Single Key

Suppose your `config.yaml` contains:

```yaml
system:
  auth:
    signups_disabled: false
```

To override just `signups_disabled`, create a `config.local.yaml`:

```yaml
system:
  auth:
    signups_disabled: true
```

### Error Handling

If the config or override file cannot be merged (e.g., due to incompatible types), the application will fail to start and print a clear error message describing the merge failure.

### Test Coverage

The config system is fully tested for deep merging, partial overrides, and error handling. See `backend/internal/config/config_test.go` for details.

## Configuration: Local Overrides

The application supports a local config override mechanism for all environments (development, test, production) using a `config.local.yaml` file. This allows you to override any settings from the main `config.yaml` without modifying the main file.

- By default, the app will look for `config.local.yaml` in the same directory as `config.yaml`.
- You can explicitly specify the path to the local override file using the `QUIZ_CONFIG_LOCAL_FILE` environment variable.
- The local config is deep-merged into the main config using the [mergo](https://github.com/imdario/mergo) library.

### Usage in Docker Compose

To enable a local override in production or test, add the following environment variable to your `docker-compose.yml` for the backend and worker services:

```yaml
services:
  backend:
    environment:
      - QUIZ_CONFIG_FILE=/app/config.yaml
      # - QUIZ_CONFIG_LOCAL_FILE=/app/config.local.yaml  # Uncomment to use a local override in production
  worker:
    environment:
      - QUIZ_CONFIG_FILE=/app/config.yaml
      # - QUIZ_CONFIG_LOCAL_FILE=/app/config.local.yaml  # Uncomment to use a local override in production
```

Make sure to copy or mount `config.local.yaml` into the container if you want to use it.

### Example: Enabling Signups in Test

To enable user signups for integration tests (when the main config disables them), create a `config.local.yaml` with:

```yaml
system:
  auth:
    signups_disabled: false
```

Set the environment variable in your test runner or Taskfile:

```yaml
env:
  QUIZ_CONFIG_LOCAL_FILE: "{{.TASKFILE_DIR}}/config.local.yaml"
```

This mechanism works for any config override you need in any environment.

---

## Hacks

- `OTEL_SEMCONV_STABILITY_OPT_IN=http/dup` -> https://github.com/SigNoz/signoz/issues/8406
