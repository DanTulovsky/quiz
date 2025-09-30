#!/usr/bin/env python3
"""
Post-build script to add CSP nonces to the built frontend HTML.
This runs after Vite has built the application and adds nonces to script and style tags.
"""

import re
import os
import sys
import base64
import secrets

def generate_nonce():
    """Generate a cryptographically secure nonce."""
    return base64.b64encode(secrets.token_bytes(16)).decode('utf-8')

def add_nonces_to_html(html_file_path):
    """Add nonces to script and style tags in the built HTML."""
    try:
        with open(html_file_path, 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print(f"Error: {html_file_path} not found")
        return None

    # Generate a new nonce
    nonce = generate_nonce()

    # Add nonce to script tags
    content = re.sub(
        r'<script([^>]*)>',
        f'<script\\1 nonce="{nonce}">',
        content
    )

    # Add nonce to style tags (if any)
    content = re.sub(
        r'<style([^>]*)>',
        f'<style\\1 nonce="{nonce}">',
        content
    )

    # Add or update the CSP-NONCE comment
    if '<!-- CSP-NONCE:' in content:
        content = re.sub(
            r'<!-- CSP-NONCE: [^>]+ -->',
            f'<!-- CSP-NONCE: {nonce} -->',
            content
        )
    else:
        # Add the comment before the closing head tag
        content = content.replace(
            '</head>',
            f'    <!-- CSP-NONCE: {nonce} -->\n  </head>'
        )

    # Write the updated content
    with open(html_file_path, 'w') as f:
        f.write(content)

    print(f"Added nonce {nonce} to {html_file_path}")
    return nonce

def main():
    """Main function to add nonces to built HTML."""
    # Check multiple possible paths for the built HTML
    possible_paths = [
        'frontend/dist/index.html',  # Host context
        '/app/dist/index.html',      # Docker context
        'dist/index.html'            # Relative to current directory
    ]

    html_file = None
    for path in possible_paths:
        if os.path.exists(path):
            html_file = path
            break

    if not html_file:
        print("Error: Built HTML not found. Please run 'npm run build' in the frontend directory first.")
        print("Checked paths:")
        for path in possible_paths:
            print(f"  - {path}")
        return False

    nonce = add_nonces_to_html(html_file)
    if nonce:
        print("Nonces added successfully!")
        print(f"Nonce: {nonce}")
        return True
    else:
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
