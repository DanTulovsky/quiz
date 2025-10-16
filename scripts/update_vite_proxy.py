#!/usr/bin/env python3
"""
Script to automatically update vite.config.ts proxy configuration based on nginx.conf routes.
Parses nginx.conf for location blocks and proxy_pass directives, then maps them to dev ports
extracted from docker-compose.yml and generates an updated proxy object for Vite's dev server.
"""

import re
import os
import yaml
from pathlib import Path

# Paths to files
DOCKER_COMPOSE_PATH = 'docker-compose.yml'
NGINX_CONF_PATH = 'nginx.conf'
VITE_CONFIG_PATH = 'frontend/vite.config.ts'
VITE_PROXY_CONFIG_PATH = 'frontend/vite.proxy.config.ts'

def parse_docker_compose(compose_path):
    """
    Parse docker-compose.yml to extract service ports.
    Returns a dict of service names to their external ports.
    """
    if not os.path.exists(compose_path):
        print(f"❌ Error: {compose_path} not found.")
        return {}

    with open(compose_path, 'r') as f:
        compose = yaml.safe_load(f)

    ports = {}
    services = compose.get('services', {})
    for service, config in services.items():
        port_list = config.get('ports', [])
        for port_mapping in port_list:
            # Parse port mapping like "8080:8080" or "${TTS_PORT:-5050}:5050"
            if isinstance(port_mapping, str) and ':' in port_mapping:
                # Split from the right to handle environment variables with colons
                parts = port_mapping.rsplit(':', 1)
                external, internal = parts[0], parts[1]
                # Handle environment variables like ${VAR:-default} or '${VAR:-default}'
                if external.startswith(('"', "'")) and external.endswith(('"', "'")):
                    external = external[1:-1]  # Remove quotes
                if external.startswith('${'):
                    # Extract the default value after ':-'
                    inner = external[2:-1]  # Remove ${ and }
                    if ':-' in inner:
                        external = inner.split(':-', 1)[1]
                    else:
                        # If no default, skip or handle as needed
                        continue
                try:
                    ports[service] = int(external)
                    break  # Assume one primary port per service
                except ValueError:
                    print(f"Warning: Could not parse port for {service}: {external}")
                    continue

    return ports

def parse_nginx_conf(conf_path):
    """
    Parse nginx.conf to extract location blocks and their proxy_pass targets.
    Returns a list of tuples: (location_pattern, proxy_target)
    """
    with open(conf_path, 'r') as f:
        content = f.read()

    # Regex to find location blocks with proxy_pass
    # Matches: location [~]? /path { ... proxy_pass http://service:port; ... }
    location_regex = re.compile(
        r'location\s+(?:~\s*)?([^{\s]+)\s*\{[^}]*?proxy_pass\s+(\S+);',
        re.MULTILINE | re.DOTALL
    )

    routes = []
    for match in location_regex.finditer(content):
        location_pattern = match.group(1).strip()
        proxy_target = match.group(2).strip().rstrip(';')
        routes.append((location_pattern, proxy_target))

    return routes

def map_to_dev_ports(routes, service_ports):
    """
    Map NGINX proxy targets to dev ports based on docker-compose.yml.
    """
    port_mappings = {}
    for service, port in service_ports.items():
        if service == 'backend':
            port_mappings[f'http://backend:8080'] = f'http://localhost:{port}'
        elif service == 'worker':
            port_mappings[f'http://worker:8081'] = f'http://localhost:{port}'
        elif service == 'tts':
            port_mappings[f'http://tts:5050'] = f'http://localhost:{port}'

    dev_routes = {}
    for location, target in routes:
        if target in port_mappings:
            dev_routes[location] = port_mappings[target]
        else:
            # Fallback for unmapped targets (e.g., use backend port)
            dev_routes[location] = f'http://localhost:{service_ports.get("backend", 8080)}'

    return dev_routes

def generate_vite_proxy_code(dev_routes):
    """
    Generate the proxy config code for the separate file.
    """
    lines = []
    lines.append('// Auto-generated proxy config from nginx.conf')
    lines.append('export const proxyConfig = {')

    for location, target in dev_routes.items():
        lines.append(f"  // Proxy {location} to {target}")
        lines.append(f"  '{location}': {{")
        lines.append(f"    target: '{target}',")
        lines.append("    changeOrigin: true,")
        lines.append("  },")

    lines.append('};')
    return '\n'.join(lines)

def update_vite_config(vite_path, new_proxy_code):
    """
    Generate the proxy config file and format both files.
    """
    import subprocess

    # Write the proxy config to a separate file
    with open(VITE_PROXY_CONFIG_PATH, 'w') as f:
        f.write(new_proxy_code)
    print(f"✅ Generated {VITE_PROXY_CONFIG_PATH} with proxy configuration.")

    # Run formatting on both files
    try:
        frontend_dir = os.path.dirname(vite_path)
        subprocess.run(['npx', 'prettier', '--write', 'vite.config.ts', 'vite.proxy.config.ts'], cwd=frontend_dir, check=True)
        print(f"✅ Formatted {vite_path} and {VITE_PROXY_CONFIG_PATH}.")
    except subprocess.CalledProcessError as e:
        print(f"❌ Formatting failed (exit code {e.returncode}). Please fix manually or run 'npx prettier --write frontend/vite.config.ts frontend/vite.proxy.config.ts'.")
        raise
    except FileNotFoundError:
        print("❌ Prettier not found; skipping formatting.")

def main():
    # Parse docker-compose.yml for ports
    service_ports = parse_docker_compose(DOCKER_COMPOSE_PATH)
    if not service_ports:
        print("❌ No service ports found in docker-compose.yml.")
        return

    print(f"📋 Extracted service ports from {DOCKER_COMPOSE_PATH}:")
    for service, port in service_ports.items():
        print(f"  {service}: {port}")

    # Parse NGINX conf
    routes = parse_nginx_conf(NGINX_CONF_PATH)
    if not routes:
        print("❌ No proxy routes found in nginx.conf.")
        return

    print(f"📋 Found {len(routes)} proxy routes in nginx.conf:")
    for location, target in routes:
        print(f"  {location} -> {target}")

    # Map to dev ports
    dev_routes = map_to_dev_ports(routes, service_ports)
    print("🔄 Mapped to dev ports:")
    for location, target in dev_routes.items():
        print(f"  {location} -> {target}")

    # Generate new proxy code
    new_proxy_code = generate_vite_proxy_code(dev_routes)
    print(f"📝 Generated proxy code:\n{new_proxy_code}")

    # Update vite.config.ts
    update_vite_config(VITE_CONFIG_PATH, new_proxy_code)
    print("🎉 Update complete! Run 'npm run dev' to test the new proxy setup.")

if __name__ == '__main__':
    main()
