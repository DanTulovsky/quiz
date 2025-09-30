#!/usr/bin/env python3
"""
Script to update nginx.conf to use CSP nonces for frontend instead of hashes.
This will significantly reduce the CSP header size and improve security.
"""

import re
import sys

def update_nginx_for_frontend_nonces():
    """Update nginx.conf to use nonces for frontend CSP."""
    try:
        with open('nginx.conf', 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print("Error: nginx.conf not found")
        return False

    # Replace the style-src directive to use nonce instead of hashes
    # Keep 'self' and external sources, replace hashes with nonce
    style_src_pattern = r"(style-src 'self')([^;]+)(https://fonts\.googleapis\.com[^;]*;)"

    def replace_style_src(match):
        prefix = match.group(1)
        # Remove all the hash entries and replace with nonce
        suffix = match.group(3)
        return f"{prefix} 'nonce-{{CSP_NONCE}}' {suffix}"

    new_content = re.sub(style_src_pattern, replace_style_src, content)

    # Also add script-src nonce if needed
    script_src_pattern = r"(script-src 'self')([^;]*;)"

    def replace_script_src(match):
        prefix = match.group(1)
        suffix = match.group(2)
        return f"{prefix} 'nonce-{{CSP_NONCE}}' {suffix}"

    new_content = re.sub(script_src_pattern, replace_script_src, new_content)

    # Write the updated configuration
    with open('nginx.conf', 'w') as f:
        f.write(new_content)

    print("Updated nginx.conf to use nonces for frontend CSP")
    print("The nginx configuration now uses 'nonce-{CSP_NONCE}' placeholders")
    print("You'll need to configure nginx to extract the nonce from HTML comments")
    print("and replace {CSP_NONCE} with the actual nonce value.")

    return True

if __name__ == "__main__":
    success = update_nginx_for_frontend_nonces()
    sys.exit(0 if success else 1)
