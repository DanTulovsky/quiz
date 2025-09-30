# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Architecture Overview

This is an AI-powered adaptive language learning quiz application with a three-service architecture:

### Services
- **Backend** (Port 8080): Go-based API server using Gin framework, handles user auth, quiz logic, and admin endpoints
- **Worker** (Port 8081): Autonomous background service for AI question generation with user-specific AI model settings
- **Frontend** (Port 3000): React 18 + TypeScript application with Mantine UI components

### Service Communication
- Frontend ↔ Backend: REST API calls via axios, session-based auth with cookies
- Backend ↔ Worker: Admin analytics and status queries (worker operates independently)
- Worker ↔ AI Providers: Configurable AI providers (Ollama, OpenAI, Anthropic, Google) via OpenAI-compatible API
- All services → PostgreSQL: Shared database for users, questions, responses, and system state

### Data Flow
1. User takes quiz → Backend serves cached questions from database
2. **Worker autonomously monitors** question pool levels and user performance
3. Worker automatically generates new questions → Calls configured AI provider based on user's language level and performance
4. Questions stored in database → Available for future quiz sessions
5. Backend serves pre-generated questions to users

## Essential Development Commands

### Quick Start
```bash
# Start all production services (recommended)
task start-prod

# Start development services (builds locally)
task start-dev

# Restart production services
task restart-prod
```

### Testing
```bash
# Run all tests (unit + integration + frontend)
task test

# Backend tests only
task test-go                    # Both unit and integration
task test-go-unit              # Unit tests (no database)
task test-go-integration       # Integration tests (requires database)

# Frontend tests
task test-frontend             # Unit tests with Vitest

# E2E tests with Playwright
task test-e2e                 # Full E2E test suite
task test-e2e-api             # API-focused E2E tests
task test-e2e-keep-running    # Keep servers running after tests
```

### Development Tools
```bash
# Code formatting and linting
task format                   # Format all code (Go + TypeScript)
task lint                    # Lint all code
task deadcode                # Find unused code

# API development
task generate-api-types      # Generate TypeScript types from swagger.yaml
task validate-api           # Check for undocumented API endpoints

# Database management
task migrate-up             # Apply database migrations
task reset-prod-db          # Reset database (with confirmation)
task setup-test-db          # Setup test database with sample data
```

### Specialized Commands
```bash
# CLI worker tool (for manual batch question generation)
task run-cli-worker CLI_ARGS="--user admin --count 10"

# Load testing with Artillery
task test-artillery-run TEST_NAME=login-test
task test-artillery-all

# Security scanning with ZAP
task zap                     # Baseline security scan
task zap-quick              # Fast security scan
```

## Configuration System

The application uses a sophisticated configuration system centered on `config.yaml`:

### Config Structure
- **AI Providers**: Ollama (local), OpenAI, Anthropic, Google with model-specific settings
- **Languages**: Italian, French, German, Russian, Japanese, Chinese with CEFR/HSK levels
- **Question Generation**: Batch sizes, concurrent limits, variety settings, refill thresholds
- **Email**: SMTP configuration for daily reminders and notifications

### Worker Configuration
- `question_refill_threshold`: Minimum questions before worker generates more (default: 5)
- `daily_fresh_question_ratio`: Fraction of fresh questions to maintain (default: 35%)
- `daily_horizon_days`: Days ahead to generate questions (default: 1)
- `daily_repeat_avoid_days`: Days to avoid repeating correct answers (default: 7)

### Config Overrides
- `config.local.yaml`: Local overrides for any environment
- Environment variables: Override specific settings (e.g., `DATABASE_URL`)
- Deep merging: Override individual keys without duplicating entire config

### AI Integration
- Per-user AI provider and model selection
- Concurrent request limiting (global and per-user)
- Grammar support varies by provider (Google doesn't support grammar constraints)
- Question batch sizes vary by provider (Ollama: 1, OpenAI/Anthropic: 5)

## Key Architectural Patterns

### Autonomous Worker Architecture
- Worker runs continuously and monitors user question pools
- Automatically generates questions when pools fall below threshold
- Prioritizes users with low question counts or poor performance
- Pre-generates daily questions within configurable horizon
- Maintains variety through topic categories, grammar focus, and difficulty modifiers

### OpenAPI-First Development
- `swagger.yaml` is the single source of truth for API contracts
- Backend generates Go types with oapi-codegen
- Frontend generates TypeScript types with orval
- Prevents type mismatches between services

### Adaptive Learning Algorithm
- Questions adapt to user performance and prioritize weak areas
- Avoids repeating recently correct answers
- Maintains balance between fresh content and difficulty progression
- Configurable difficulty modifiers and grammar focus areas per language level

### Database Design
- PostgreSQL with automated migrations
- Session-based authentication with secure cookies
- Question caching minimizes AI API calls through autonomous pre-generation
- Comprehensive analytics and user performance tracking

### Containerized Development
- Docker Compose for all environments (dev, test, prod)
- BuildKit optimizations with layer caching
- Multi-stage builds for minimal production images
- Health checks and service dependencies

## Security & Production Features

### Content Security Policy
- Automated CSP nonce generation during build
- Self-hosted fonts (no external CDN dependencies)
- Nginx configuration with security headers

### AI Concurrency Control
- Global and per-user concurrent request limits
- Real-time monitoring via admin dashboards
- Fail-fast behavior prevents queue buildup

### Email System
- Daily reminder emails with timezone support
- Comprehensive error tracking and retry logic
- Admin dashboard for notification management

## Testing Strategy

### Multi-Layer Testing
- **Unit Tests**: Go services with testify, frontend with Vitest
- **Integration Tests**: Database-backed testing with test containers
- **E2E Tests**: Playwright covering user journeys and API contracts
- **Load Tests**: Artillery for performance and security fuzzing
- **Security Tests**: ZAP baseline and authenticated scans

### Test Database Management
- Isolated test database (port 5433)
- Automatic setup/teardown with Docker Compose
- Golden data fixtures for consistent E2E testing

## Development Workflow

### Code Generation
1. Modify `swagger.yaml` for API changes
2. Run `task generate-api-types` to sync Go and TypeScript types
3. Update backend handlers and frontend components
4. Run `task validate-api` to ensure completeness

### AI Provider Integration
1. Add provider config to `config.yaml`
2. Test with CLI worker: `task run-cli-worker --provider new-provider`
3. Verify question generation in admin dashboard
4. Add to frontend provider selection

### Database Migrations
1. Create migration: `task migrate-create NAME=add_new_feature`
2. Edit generated `.up.sql` and `.down.sql` files
3. Test: `task migrate-up` and `task migrate-down`
4. Verify with integration tests

## Important Files & Directories

- `config.yaml` / `config.local.yaml`: Application configuration
- `swagger.yaml`: API specification (single source of truth)
- `Taskfile.yml`: Build system with all development commands
- `docker-compose.yml`: Development services
- `docker-compose.prod.yml`: Production services
- `backend/migrations/`: Database schema changes
- `frontend/src/api/`: Generated API client code
- `.cursor/rules/`: IDE-specific development guidance

## Cursor/AI Assistant Rules

Based on `.cursor/rules/`:

### Frontend Development
- Prefer Mantine UI components over custom CSS/TypeScript
- Use established libraries rather than coding from scratch
- Follow the existing component patterns and file structure

### Generated Files
- Never manually edit files in `frontend/src/api/` - they are auto-generated
- Always use `task generate-api-types` after changing `swagger.yaml`
- Respect the build system's automation for CSP nonces and font management
