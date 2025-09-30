# Artillery Load Testing & Security Testing

This directory contains Artillery load testing and security testing configurations for the Quiz Application.

## 🎯 Overview

Artillery is used for:
- **Load Testing**: Performance testing under various load conditions
- **Security Testing**: Fuzzing tests to identify vulnerabilities
- **API Testing**: Functional testing of API endpoints
- **Integration Testing**: End-to-end workflow testing

## 📁 Test Files

### Core Tests

- **`tests/login-test.yml`**: Basic login functionality testing
  - Tests both valid and invalid login scenarios
  - Validates proper status code responses
  - Uses the `expect` plugin for assertions

- **`tests/signup-login-test.yml`**: Complete user registration and login workflow
  - Tests the full signup → login → profile access flow
  - Validates user creation and session management
  - Tests error handling for invalid signup data

### Security Testing

- **`tests/login-fuzzer-test.yml`**: Security fuzzing tests
  - Uses the `fuzzer` plugin to send "naughty strings"
  - Tests SQL injection, XSS, path traversal, and other attack vectors
  - Validates that the application handles malicious inputs gracefully
  - Expects only 400 (Bad Request) or 401 (Unauthorized) responses

## 🛠️ Setup

### Prerequisites

The Artillery setup is included in the main project's `install-tooling.sh` script, which installs:
- Artillery CLI
- Artillery fuzzer plugin
- Artillery expect plugin

### Docker Setup

The Artillery tests run in a Docker container with all plugins pre-installed:

```bash
# Build the Artillery Docker image
task build-artillery

# Run tests using Docker
task test-artillery-run TEST_NAME=login-test
```

## 🚀 Running Tests

### Quick Commands

```bash
# Run a specific test
task test-artillery-run TEST_NAME=login-test
task test-artillery-run TEST_NAME=login-fuzzer-test
task test-artillery-run TEST_NAME=signup-login-test

# Run all tests sequentially
task test-artillery-all

# Run in solo mode (single request per test)
task test-artillery-run TEST_NAME=login-test SOLO=1
```

### Manual Commands

```bash
# Run directly with Artillery (requires local installation)
artillery run artillery/tests/login-test.yml
artillery run artillery/tests/login-fuzzer-test.yml

# Run specific scenario
artillery run artillery/tests/login-test.yml --scenario-name "Login with unknown user"

# Run with output report
artillery run artillery/tests/login-test.yml --output report.json
```

## 🔧 Configuration

### Plugins Used

- **`expect`**: Validates response status codes and content
- **`fuzzer`**: Generates malicious input strings for security testing

### Test Environment

Tests run against the test environment:
- **Target**: `http://host.docker.internal:3001`
- **Database**: Separate test database with golden data
- **Services**: Isolated test containers

## 📊 Test Scenarios

### Login Test (`login-test.yml`)

**Scenarios:**
1. **Login with existing user** (70% weight)
   - Uses `generateRandomUser` processor
   - Expects 200 status codes
   - Tests successful authentication flow

2. **Login with unknown user** (30% weight)
   - Tests invalid credentials
   - Expects 401 for login, 200 for status check
   - Validates proper error handling

### Fuzzer Test (`login-fuzzer-test.yml`)

**Scenarios:**
1. **Login with fuzzed username** (50% weight)
   - Uses `{{ naughtyString }}` as username
   - Tests SQL injection, XSS, and other attack vectors

2. **Login with fuzzed password** (30% weight)
   - Uses `{{ naughtyString }}` as password
   - Tests password field security

3. **Login with fuzzed JSON payload** (20% weight)
   - Uses `{{ naughtyString }}` as entire JSON payload
   - Tests malformed request handling

### Signup-Login Test (`signup-login-test.yml`)

**Scenarios:**
1. **Complete signup and login workflow** (80% weight)
   - Generates unique user data
   - Tests registration → login → profile access
   - Validates session management

2. **Signup with invalid data** (20% weight)
   - Tests invalid email formats
   - Tests weak passwords
   - Validates input validation

## 🔍 Understanding Results

### Success Indicators

- **Exit code 0**: All tests passed
- **Exit code 21**: Some expectations failed (this is expected for fuzzer tests)
- **`plugins.expect.ok`**: Number of passed assertions
- **`plugins.expect.failed`**: Number of failed assertions

### Expected Behaviors

- **Valid login**: 200 status code
- **Invalid login**: 401 status code
- **Malformed requests**: 400 status code
- **Fuzzer tests**: Should return 400 or 401, never 500

### Security Testing Goals

The fuzzer tests should **never** return 500 status codes. A 500 error indicates:
- Server crash or internal error
- Potential security vulnerability
- Bug that needs immediate fixing

## 🐛 Troubleshooting

### Common Issues

1. **Plugin not found errors**
   ```bash
   # Rebuild the Docker image
   task build-artillery
   ```

2. **Test environment not running**
   ```bash
   # Start test environment
   task start-test-environment
   ```

3. **Database connection issues**
   ```bash
   # Reset test database
   task setup-test-db
   ```

### Debug Mode

Enable debug output for the fuzzer plugin:
```bash
DEBUG=plugin:fuzzer artillery run artillery/tests/login-fuzzer-test.yml
```

## 📈 Performance Testing

For load testing beyond basic functionality:

```bash
# Run load test with multiple phases
artillery run artillery/artillery.config.yml

# Custom load test
artillery run --overrides '{"config": {"phases": [{"duration": 60, "arrivalRate": 10}]}}' artillery/tests/login-test.yml
```

## 🔐 Security Best Practices

1. **Never expect 500 errors** in security tests
2. **Always validate input handling** with fuzzer tests
3. **Test both positive and negative scenarios**
4. **Monitor for unexpected behavior** during fuzzer tests
5. **Regular security testing** as part of CI/CD pipeline

## 📚 Resources

- [Artillery Documentation](https://www.artillery.io/docs/)
- [Artillery Expect Plugin](https://www.artillery.io/docs/reference/extensions/expect)
- [Artillery Fuzzer Plugin](https://www.artillery.io/docs/reference/extensions/fuzzer)
- [Big List of Naughty Strings](https://github.com/minimaxir/big-list-of-naughty-strings)
