# ZAP Security Scan Configurations

This directory contains OWASP ZAP security scan configurations for the quiz application. The configurations use inheritance to avoid repetition and provide different scan types for various testing scenarios.

## Directory Structure

```
zap/
├── configs/           # Configuration files
│   ├── base.yaml     # Base configuration (inherited by others)
│   ├── baseline.yaml # Unauthenticated baseline scan
│   ├── authenticated.yaml # Full scan with login
│   ├── api-only.yaml # API endpoint testing
│   ├── quick.yaml    # Fast development scan
│   └── merged-*.yaml # Generated merged configs (gitignored)
├── reports/          # Scan reports (gitignored)
│   ├── zap-*-report.html
│   └── zap-*-report.json
└── scripts/          # Utility scripts
    └── merge-config.py # Configuration merger
```

## Configuration Structure

### Base Configuration (`configs/base.yaml`)
- Contains common settings shared across all scan types
- Defines default timeouts, alert thresholds, and scan policies
- **Uses modern AJAX spider** for better coverage of dynamic web applications
- Other configurations inherit from this base

### Scan Types

1. **Baseline** (`configs/baseline.yaml`) - Unauthenticated quick scan
   - Target: Frontend (http://localhost:3000)
   - Purpose: Quick security check of public endpoints
   - Duration: ~5-10 minutes

2. **Authenticated** (`configs/authenticated.yaml`) - Full scan with login
   - Target: Frontend (http://localhost:3000)
   - Purpose: Comprehensive testing of authenticated functionality
   - Duration: ~30-60 minutes
   - **Authentication**: Uses environment variables for credentials

3. **API-Only** (`configs/api-only.yaml`) - Backend API testing
   - Target: Backend (http://localhost:8081)
   - Purpose: Focused testing of `/v1/*` API endpoints
   - Duration: ~20-40 minutes

4. **Quick** (`configs/quick.yaml`) - Development testing
   - Target: Frontend (http://localhost:3000)
   - Purpose: Fast feedback for development workflow
   - Duration: ~2-5 minutes

## Authentication Configuration

### Environment Variables
The authenticated scan uses environment variables for security:

```bash
export ZAP_ADMIN_USERNAME=admin@example.com
export ZAP_ADMIN_PASSWORD=admin123
```

**Important**: These environment variables are **required** for authenticated scans. If they're not set, the scan will fail with a clear error message.

### `logged_in_regex` Usage
The `logged_in_regex` field in authenticated scans is used by ZAP to detect successful authentication by looking for specific text patterns in the response. For this application:

- **Pattern**: `"quiz|progress|settings|logout"`
- **Purpose**: Detects when user is successfully logged in by finding navigation elements
- **Routes**: `/quiz`, `/progress`, `/settings` (main authenticated pages)
- **Logout**: Detects logout button in header

### `logged_out_regex` Usage
The `logged_out_regex` field detects when authentication fails:

- **Pattern**: `"login|signin"`
- **Purpose**: Detects login page elements when user is not authenticated

## API Endpoints

The application uses `/v1/*` endpoints for API functionality:

- **Authentication**: `/v1/auth/login`, `/v1/auth/logout`, `/v1/auth/status`
- **Quiz**: `/v1/quiz/question`, `/v1/quiz/progress`
- **Settings**: `/v1/settings`
- **User Management**: `/v1/userz/*`

## Reports Directory

All ZAP scan reports are generated in the `zap/reports/` directory:

```
zap/reports/
├── zap-baseline-report.html
├── zap-baseline-report.json
├── zap-authenticated-report.html
├── zap-authenticated-report.json
├── zap-api-report.html
├── zap-api-report.json
├── zap-quick-report.html
└── zap-quick-report.json
```

## Usage

### Prerequisites
1. **Docker** - All scans use Docker (no local ZAP installation required)
2. Start the application: `task start-prod`
3. For authenticated scans, set environment variables:
   ```bash
   export ZAP_ADMIN_USERNAME=admin@example.com
   export ZAP_ADMIN_PASSWORD=admin123
   ```

### Running Scans

#### All scans use Docker automatically:
```bash
# Quick development scan
task zap-quick

# Baseline unauthenticated scan
task zap

# Full authenticated scan (requires env vars)
task zap-authenticated

# API-only scan
task zap-api

# Run all scans
task zap-all
```

#### Manual Docker Commands
```bash
# Quick scan with Docker
task zap-docker

# Full scan with Docker
task zap-docker-full
```

### Docker Images
The system uses the official OWASP ZAP Docker images from GitHub Container Registry:
- **Stable**: `ghcr.io/zaproxy/zaproxy:stable` (recommended for production)
- **Weekly**: `ghcr.io/zaproxy/zaproxy:weekly` (latest features)
- **Nightly**: `ghcr.io/zaproxy/zaproxy:nightly` (cutting edge)

Reference: [ZAP Docker Documentation](https://www.zaproxy.org/docs/docker/about/)

### Configuration Inheritance

The merge script (`scripts/merge-config.py`) handles configuration inheritance:

1. Loads `base.yaml` as the foundation
2. Merges specific configuration overrides
3. Generates complete merged configuration files
4. Outputs to `configs/merged-*.yaml`

### Generate All Configs
```bash
task zap-merge-configs
```

## Modern Spider Configuration

The base configuration uses the **AJAX spider** (modern spider) instead of the traditional spider:

- **Better coverage** of dynamic web applications
- **Handles JavaScript-heavy sites** more effectively
- **Supports modern frameworks** like React, Vue, Angular
- **Configurable crawl depth and duration**
- **Event-driven crawling** for better discovery

## Customization

### Adding New Scan Types
1. Create new YAML file in `configs/`
2. Add `extends: "base.yaml"` to inherit base settings
3. Override specific settings as needed
4. Add corresponding task in `Taskfile.yml`

### Modifying Authentication
Update `configs/authenticated.yaml`:
- `username`: `${ZAP_ADMIN_USERNAME}` (environment variable)
- `password`: `${ZAP_ADMIN_PASSWORD}` (environment variable)
- `logged_in_regex`: Patterns to detect successful login
- `logged_out_regex`: Patterns to detect failed login

### Adjusting Scan Intensity
- **Quick**: High threshold, minimal depth, short timeouts
- **Baseline**: Medium threshold, moderate depth
- **Full**: Low threshold, maximum depth, comprehensive testing

## Security Considerations

- Reports contain sensitive security information
- `zap/reports/` directory is gitignored
- Credentials are stored in environment variables, never in config files
- Environment variables are validated before authenticated scans
- Update credentials before running authenticated scans
- Review findings and remediate high/medium risk issues
- Consider running scans in isolated environment for production testing

## Troubleshooting

### Docker Issues
- Ensure Docker is running
- Check that the application is accessible at http://localhost:3000
- For Docker Desktop on macOS, `host.docker.internal` should resolve to localhost
- Uses `ghcr.io/zaproxy/zaproxy:stable` image

### Authentication Issues
- **Environment variables required**: `ZAP_ADMIN_USERNAME` and `ZAP_ADMIN_PASSWORD` must be set
- Verify environment variables are set correctly
- Check that the `logged_in_regex` patterns match your application
- Ensure the application is running and accessible

### Configuration Issues
- Run `task zap-merge-configs` to regenerate merged configurations
- Check that all required files exist in `configs/`
- Verify Python and PyYAML are installed for the merge script

## Test Results

The ZAP Docker setup has been tested and verified working:
- ✅ Docker image pulls successfully (`ghcr.io/zaproxy/zaproxy:stable`)
- ✅ Baseline scan completes without errors
- ✅ Reports generated in `zap/reports/` directory
- ✅ Environment variable validation works for authenticated scans
- ✅ Configuration inheritance system functional
- ✅ Modern AJAX spider configured for better coverage
