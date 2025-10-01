# Rate Limiting Configuration

This document explains how rate limiting is implemented in the Quiz application's nginx configuration.

## Overview

Rate limiting is implemented using nginx's `limit_req` module with different zones for different types of endpoints. The system can be toggled on/off based on the deployment environment.

## Rate Limiting Zones

The following rate limiting zones are defined in `nginx.conf`:

```nginx
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=auth_limit:10m rate=5r/s;
limit_req_zone $binary_remote_addr zone=quiz_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=default_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=tts_limit:10m rate=5r/s;
```

Each zone configuration includes:
- **Key**: `$binary_remote_addr` - rate limiting per IP address
- **Zone name**: Memory zone for storing request counts
- **Size**: `10m` - 10MB of shared memory for the zone
- **Rate**: Requests per second limit

### Zone Details

| Zone | Rate | Purpose | Burst |
|------|------|---------|-------|
| `api_limit` | 10 r/s | General API endpoints | 15 |
| `auth_limit` | 5 r/s | Authentication endpoints (login/signup) | 10 |
| `quiz_limit` | 10 r/s | Quiz-related endpoints | 20 |
| `default_limit` | 10 r/s | General routes (SPA, static assets) | 20 |
| `tts_limit` | 5 r/s | Text-to-speech endpoints | 10 |

## How Rate Limiting Works

### Burst Configuration

Each zone uses `nodelay` configuration, which means:
- Requests exceeding the rate limit are delayed rather than rejected
- The burst parameter allows a buffer of requests that can exceed the rate
- Requests in the burst buffer are processed as server capacity allows

For example: `limit_req zone=api_limit burst=15 nodelay;`
- Allows up to 15 requests to queue up when the 10 r/s limit is exceeded
- These queued requests are processed with minimal delay rather than being rejected

### Affected Endpoints

Rate limiting is applied to different endpoints based on their function:

- **Authentication endpoints** (`/v1/auth/login`, `/v1/auth/signup`): Strictest limits (5 r/s) due to security concerns
- **Quiz endpoints** (`/v1/quiz/`): Moderate limits (10 r/s) for gameplay experience
- **TTS endpoints** (`/v1/audio/`, `/v1/voices`): Conservative limits (5 r/s) due to resource intensity
- **General API endpoints**: Standard limits (10 r/s) for most API calls
- **Static assets and SPA routes**: Standard limits (10 r/s) for frontend serving

## Environment-Based Configuration

### Production Environment

In production builds (`Dockerfile.frontend`):
```dockerfile
COPY nginx/snippets/on/ /etc/nginx/snippets/
```

The "on" snippets contain active rate limiting directives:
```nginx
# nginx/snippets/on/ratelimit-api.inc
limit_req zone=api_limit burst=15 nodelay;
```

### Test Environment

In test environments (`docker-compose.test.yml`):
```yaml
volumes:
  - ./nginx/snippets/off:/etc/nginx/snippets:ro
```

The "off" snippets are empty files, effectively disabling rate limiting:
```nginx
# nginx/snippets/off/ratelimit-api.inc
# (empty file - no rate limiting)
```

## Files Involved

### Core Configuration
- `nginx.conf` - Defines rate limiting zones and applies snippets to location blocks
- `nginx/snippets/on/` - Active rate limiting rules
- `nginx/snippets/off/` - Disabled rate limiting (empty files)

### Environment Control
- `Dockerfile.frontend` - Copies "on" snippets for production
- `docker-compose.test.yml` - Mounts "off" snippets for testing
- `docker-compose.test.from.images.yml` - Alternative test configuration

## Security Considerations

- Rate limiting helps prevent abuse and ensures fair resource usage
- Authentication endpoints have stricter limits to prevent brute force attacks
- TTS endpoints have conservative limits due to higher resource consumption
- The `nodelay` configuration provides better user experience by queuing requests rather than rejecting them
- IP-based limiting affects all requests from the same client IP address

## Monitoring

Rate limiting events are logged in nginx's access logs. When rate limits are exceeded, nginx will:
1. Queue requests up to the burst limit
2. Process queued requests as server capacity allows
3. Log the rate limiting events for monitoring and analysis
