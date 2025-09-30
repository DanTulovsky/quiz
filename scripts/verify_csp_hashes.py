#!/usr/bin/env python3
"""
Script to verify CSP hash lengths in nginx.conf
"""

import re

def verify_csp_hashes():
    """Verify all CSP hashes are the correct length"""
    try:
        with open('nginx.conf', 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print("Error: nginx.conf not found in current directory")
        return

    # Find the CSP line
    csp_pattern = r'add_header Content-Security-Policy "[^"]*" always;'
    csp_match = re.search(csp_pattern, content)

    if not csp_match:
        print("Error: Could not find Content-Security-Policy header in nginx.conf")
        return

    # Extract hashes
    hash_pattern = r"'sha256-([^']+)'"
    hashes = re.findall(hash_pattern, csp_match.group(0))

    print(f"Found {len(hashes)} CSP hashes:")

    all_valid = True
    for i, hash_val in enumerate(hashes, 1):
        clean_hash = hash_val.rstrip('=')
        length = len(clean_hash)
        status = "✓" if length == 44 else "✗"
        print(f"  {i}. {status} sha256-{hash_val} (length: {length})")

        if length != 44:
            all_valid = False

    if all_valid:
        print("\n✅ All CSP hashes are valid (44 characters)")
    else:
        print("\n❌ Some CSP hashes have incorrect length")

if __name__ == "__main__":
    verify_csp_hashes()
