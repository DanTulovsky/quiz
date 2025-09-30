#!/usr/bin/env python3
"""
Conservative script to fix CSP hash length warnings in nginx.conf
Only removes hashes that are clearly malformed (42 chars)
Keeps 43-char hashes as they might be working in practice
"""

import re
import sys

def extract_hashes_from_csp(csp_line):
    """Extract all sha256 hashes from CSP line"""
    # Pattern to match sha256 hashes
    hash_pattern = r"'sha256-([^']+)'"
    matches = re.findall(hash_pattern, csp_line)
    return matches

def analyze_hashes(hashes):
    """Analyze hash lengths and identify clearly problematic ones"""
    clearly_problematic = []  # 42 chars or less
    potentially_working = []  # 43 chars (might work despite ZAP warning)
    valid = []  # 44 chars

    for hash_value in hashes:
        # Remove any padding characters and count actual hash length
        clean_hash = hash_value.rstrip('=')
        length = len(clean_hash)

        if length <= 42:
            clearly_problematic.append(hash_value)
        elif length == 43:
            potentially_working.append(hash_value)
        else:
            valid.append(hash_value)

    return clearly_problematic, potentially_working, valid

def fix_csp_line(csp_line):
    """Remove only clearly problematic hashes from CSP line"""
    # Extract all hashes
    hashes = extract_hashes_from_csp(csp_line)
    clearly_problematic, potentially_working, valid = analyze_hashes(hashes)

    print(f"Found {len(hashes)} total hashes")
    print(f"Valid hashes (44 chars): {len(valid)}")
    print(f"Potentially working hashes (43 chars): {len(potentially_working)}")
    print(f"Clearly problematic hashes (≤42 chars): {len(clearly_problematic)}")

    if clearly_problematic:
        print("\nClearly problematic hashes to remove:")
        for hash_val in clearly_problematic:
            print(f"  'sha256-{hash_val}'")

    if potentially_working:
        print("\nPotentially working hashes (keeping despite ZAP warning):")
        for hash_val in potentially_working:
            print(f"  'sha256-{hash_val}'")

    # Create a new CSP line by removing only clearly problematic hashes
    new_csp = csp_line

    # Remove each clearly problematic hash
    for hash_val in clearly_problematic:
        hash_pattern = f"'sha256-{hash_val}'"
        new_csp = new_csp.replace(hash_pattern, '')

    # Clean up any double spaces that might result
    new_csp = re.sub(r'\s+', ' ', new_csp)

    return new_csp

def main():
    # Read nginx.conf
    try:
        with open('nginx.conf', 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print("Error: nginx.conf not found in current directory")
        sys.exit(1)

    # Find the CSP line
    csp_pattern = r'add_header Content-Security-Policy "[^"]*" always;'
    csp_match = re.search(csp_pattern, content)

    if not csp_match:
        print("Error: Could not find Content-Security-Policy header in nginx.conf")
        sys.exit(1)

    original_csp = csp_match.group(0)
    print("Original CSP line found:")
    print(original_csp)
    print()

    # Extract just the CSP value
    csp_value_match = re.search(r'add_header Content-Security-Policy "([^"]*)" always;', original_csp)
    csp_value = csp_value_match.group(1)

    # Fix the CSP value
    fixed_csp_value = fix_csp_line(csp_value)

    # Create new CSP line
    new_csp_line = f'add_header Content-Security-Policy "{fixed_csp_value}" always;'

    print(f"\nFixed CSP line:")
    print(new_csp_line)

    # Replace in content
    new_content = content.replace(original_csp, new_csp_line)

    # Write backup and new file
    with open('nginx.conf.backup2', 'w') as f:
        f.write(content)

    with open('nginx.conf', 'w') as f:
        f.write(new_content)

    print(f"\nBackup saved to nginx.conf.backup2")
    print(f"Fixed nginx.conf written")

if __name__ == "__main__":
    main()
