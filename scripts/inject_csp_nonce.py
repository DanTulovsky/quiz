#!/usr/bin/env python3
"""
Script to extract CSP nonce from built frontend HTML and inject it into nginx.conf.
This creates a complete CSP nonce workflow for the frontend.
"""

import re
import os
import sys

def extract_nonce_from_html(html_file_path):
    """Extract nonce from HTML comment in the built frontend."""
    try:
        with open(html_file_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: {html_file_path} not found")
        return None

    # Look for the CSP-NONCE comment
    match = re.search(r'<!-- CSP-NONCE: ([a-zA-Z0-9+/=]+) -->', content)
    if match:
        return match.group(1)
    else:
        print("Error: No CSP-NONCE comment found in HTML")
        return None

def update_nginx_with_nonce(nonce, nginx_file_path='nginx.conf'):
    """Update nginx.conf with the extracted nonce."""
    try:
        with open(nginx_file_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: {nginx_file_path} not found")
        return False

    # Replace the placeholder with the actual nonce
    new_content = content.replace('{CSP_NONCE}', nonce)

    # Write the updated configuration
    with open(nginx_file_path, 'w') as f:
        f.write(new_content)

    print(f"Updated {nginx_file_path} with nonce: {nonce}")
    return True

def update_security_headers_with_nonce(nonce, security_headers_path='nginx/snippets/on/security-headers.inc'):
    """Update security-headers.inc with the extracted nonce."""
    try:
        with open(security_headers_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: {security_headers_path} not found")
        sys.stdout.flush()
        os._exit(1)
        return False

    # Replace the placeholder with the actual nonce
    new_content = content.replace('jJv1dXcHOmvbRO2xN0o2uQ==', nonce)

    # Write the updated configuration
    with open(security_headers_path, 'w') as f:
        f.write(new_content)

    print(f"Updated {security_headers_path} with nonce: {nonce}")
    return True

def main():
    """Main function to extract nonce and update nginx."""
    # Check multiple possible paths for the built HTML
    possible_html_paths = [
        'frontend/dist/index.html',  # Host context
        '/app/dist/index.html',      # Docker context
        'dist/index.html'            # Relative to current directory
    ]

    html_file = None
    for path in possible_html_paths:
        if os.path.exists(path):
            html_file = path
            break

    if not html_file:
        print("Error: Built HTML not found. Please run 'npm run build' in the frontend directory first.")
        print("Checked paths:")
        for path in possible_html_paths:
            print(f"  - {path}")
        return False

    # Extract nonce from HTML
    nonce = extract_nonce_from_html(html_file)
    if not nonce:
        return False

    # Check multiple possible paths for nginx.conf
    possible_nginx_paths = [
        'nginx.conf',        # Host context
        '/app/nginx.conf'    # Docker context
    ]

    nginx_file = None
    for path in possible_nginx_paths:
        if os.path.exists(path):
            nginx_file = path
            break

    if not nginx_file:
        print("Error: nginx.conf not found.")
        print("Checked paths:")
        for path in possible_nginx_paths:
            print(f"  - {path}")
        return False

    # Update nginx configuration
    success = update_nginx_with_nonce(nonce, nginx_file)

    # Also update security-headers.inc if it exists
    security_headers_updated = update_security_headers_with_nonce(nonce)

    if success or security_headers_updated:
        print("CSP nonce injection completed successfully!")
        print("You can now restart nginx to apply the changes.")
        return True
    else:
        print("Failed to update any nginx configuration files")
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
