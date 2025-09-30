#!/usr/bin/env python3
"""
Script to generate CSP nonces for inline styles and scripts.
This is a more efficient alternative to using many hashes.
"""

import secrets
import base64
import re

def generate_nonce():
    """Generate a cryptographically secure nonce for CSP."""
    # Generate 16 random bytes and encode as base64
    random_bytes = secrets.token_bytes(16)
    nonce = base64.b64encode(random_bytes).decode('utf-8')
    return nonce

def update_nginx_with_nonce():
    """Update nginx.conf to use nonces instead of hashes."""
    try:
        with open('nginx.conf', 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print("Error: nginx.conf not found")
        return

    # Generate a new nonce
    nonce = generate_nonce()

    # Replace the style-src directive to use nonce instead of hashes
    # Keep 'self' and external sources, replace hashes with nonce
    style_src_pattern = r"(style-src 'self')([^;]+)(https://fonts\.googleapis\.com[^;]*;)"

    def replace_style_src(match):
        prefix = match.group(1)
        # Remove all the hash entries
        suffix = match.group(3)
        return f"{prefix} 'nonce-{nonce}' {suffix}"

    new_content = re.sub(style_src_pattern, replace_style_src, content)

    # Also add script-src nonce if needed
    script_src_pattern = r"(script-src 'self')([^;]*;)"

    def replace_script_src(match):
        prefix = match.group(1)
        suffix = match.group(2)
        return f"{prefix} 'nonce-{nonce}' {suffix}"

    new_content = re.sub(script_src_pattern, replace_script_src, new_content)

    # Write the updated configuration
    with open('nginx.conf', 'w') as f:
        f.write(new_content)

    print(f"Updated nginx.conf with nonce: {nonce}")
    print("You'll need to add this nonce to your inline styles/scripts:")
    print(f"<style nonce=\"{nonce}\">...</style>")
    print(f"<script nonce=\"{nonce}\">...</script>")

if __name__ == "__main__":
    update_nginx_with_nonce()
