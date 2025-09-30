# ZAP Security Scan Configurations

This directory contains OWASP ZAP security scan configurations for the quiz application. The configurations use inheritance to avoid repetition and provide different scan types for various testing scenarios.

## Configuration Structure

### Base Configuration (`base.yaml`)
- Contains common settings shared across all scan types
- Defines default timeouts, alert thresholds, and scan policies
- Other configurations inherit from this base

### Scan Types

1. **Baseline** (`baseline.yaml`) - Unauthenticated quick scan
   - Target: Frontend (http://localhost:3000)
   - Purpose: Quick security check of public endpoints
   - Duration: ~5-10 minutes

2. **Authenticated** (`authenticated.yaml`) - Full scan with login
   - Target: Frontend (http://localhost:3000)
   - Purpose: Comprehensive testing of authenticated functionality
   - Duration: ~30-60 minutes
   - **Authentication**: Uses form-based login with credentials from config

3. **API-Only** (`api-only.yaml`) - Backend API testing
   - Target: Backend (http://localhost:8081)
   - Purpose: Focused testing of `/v1/*` API endpoints
   - Duration: ~20-40 minutes

4. **Quick** (`quick.yaml`) - Development testing
   - Target: Frontend (http://localhost:3000)
   - Purpose: Fast feedback for development workflow
   - Duration: ~2-5 minutes

## Authentication Configuration

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

All ZAP scan reports are generated in the `zap-reports/` directory:

```
zap-reports/
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
1. Install OWASP ZAP: https://www.zaproxy.org/download/
2. Start the application: `task start-prod`
3. Update credentials in `authenticated.yaml` if needed

### Running Scans

```bash
# Quick development scan
task zap-quick

# Baseline unauthenticated scan
task zap

# Full authenticated scan
task zap-authenticated

# API-only scan
task zap-api

# Generate all merged configs
task zap-merge-configs

# Run all scans
task zap-all
```

### Configuration Inheritance

The merge script (`merge-config.py`) handles configuration inheritance:

1. Loads `base.yaml` as the foundation
2. Merges specific configuration overrides
3. Generates complete merged configuration files
4. Outputs to `zap-configs/merged-*.yaml`

### Docker Alternative

If you don't have ZAP installed locally, use Docker:

```bash
# Quick scan with Docker
task zap-docker

# Full scan with Docker
task zap-docker-full
```

## Customization

### Adding New Scan Types
1. Create new YAML file in `zap-configs/`
2. Add `extends: "base.yaml"` to inherit base settings
3. Override specific settings as needed
4. Add corresponding task in `Taskfile.yml`

### Modifying Authentication
Update `authenticated.yaml`:
- `username`: Admin username
- `password`: Admin password
- `logged_in_regex`: Patterns to detect successful login
- `logged_out_regex`: Patterns to detect failed login

### Adjusting Scan Intensity
- **Quick**: High threshold, minimal depth, short timeouts
- **Baseline**: Medium threshold, moderate depth
- **Full**: Low threshold, maximum depth, comprehensive testing

## Security Considerations

- Reports contain sensitive security information
- `zap-reports/` directory is gitignored
- Update credentials before running authenticated scans
- Review findings and remediate high/medium risk issues
- Consider running scans in isolated environment for production testing
