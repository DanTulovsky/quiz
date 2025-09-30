# NGINX OpenTelemetry Tracing

This project supports distributed tracing for all HTTP traffic through NGINX using the [OpenTelemetry NGINX module](https://github.com/open-telemetry/opentelemetry-nginx).

## Requirements
- NGINX must be built with the `ngx_otel_module` (not included in the default NGINX image).
- The `nginx.conf` in this repo is pre-configured for OpenTelemetry tracing.

## Building a Custom NGINX Image with OpenTelemetry

You can build a custom NGINX Docker image with the OpenTelemetry module using the official instructions. Here is a sample Dockerfile:

```Dockerfile
FROM nginx:1.25.3

# Install build dependencies
RUN apt-get update && apt-get install -y git build-essential libpcre3 libpcre3-dev zlib1g zlib1g-dev libssl-dev wget

# Download OpenTelemetry NGINX module
RUN git clone --branch v1.0.0 https://github.com/open-telemetry/opentelemetry-nginx.git /otel-nginx-module

# Download NGINX source (matching version)
RUN wget http://nginx.org/download/nginx-1.25.3.tar.gz && \
    tar -xzvf nginx-1.25.3.tar.gz

# Build the module
WORKDIR /nginx-1.25.3
RUN ./configure --with-compat --add-dynamic-module=/otel-nginx-module && make modules

# Copy the built module to the NGINX modules directory
RUN cp objs/ngx_otel_module.so /etc/nginx/modules/

# Clean up build dependencies and sources
WORKDIR /
RUN rm -rf /nginx-1.25.3* /otel-nginx-module && apt-get remove --purge -y build-essential git wget && apt-get autoremove -y && apt-get clean

# Use the default NGINX entrypoint
```

## Usage in Docker Compose

In your `docker-compose.yml`, use the custom image:

```yaml
  nginx:
    build:
      context: .
      dockerfile: Dockerfile.nginx-otel
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./frontend/dist:/usr/share/nginx/html:ro
    ports:
      - "80:80"
    depends_on:
      - backend
      - otel-collector
```

## References
- [OpenTelemetry NGINX Module](https://github.com/open-telemetry/opentelemetry-nginx)
- [OTel Collector Docker Hub](https://hub.docker.com/r/otel/opentelemetry-collector)
