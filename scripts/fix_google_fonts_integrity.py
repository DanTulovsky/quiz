#!/usr/bin/env python3
"""
Script to fix Google Fonts integrity issue in frontend/index.html
Removes the integrity attribute since Google Fonts content can change
"""

import re
import sys

def fix_google_fonts_integrity():
    """Remove integrity attribute from Google Fonts link"""
    try:
        with open('frontend/index.html', 'r') as f:
            content = f.read()
    except FileNotFoundError:
        print("Error: frontend/index.html not found")
        sys.exit(1)

    # Pattern to match the Google Fonts link with integrity
    google_fonts_pattern = r'(<link href="https://fonts\.googleapis\.com/css2\?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet")\s+integrity="[^"]*"\s+crossorigin="anonymous">'

    # Check if the pattern exists
    match = re.search(google_fonts_pattern, content)
    if not match:
        print("Google Fonts link with integrity not found")
        return

    # Replace with the link without integrity
    new_link = '<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap" rel="stylesheet">'

    # Replace the entire line
    new_content = re.sub(google_fonts_pattern, new_link, content)

    # Write backup and new file
    with open('frontend/index.html.backup', 'w') as f:
        f.write(content)

    with open('frontend/index.html', 'w') as f:
        f.write(new_content)

    print("Fixed Google Fonts integrity issue:")
    print("  - Removed integrity attribute from Google Fonts link")
    print("  - Backup saved to frontend/index.html.backup")

if __name__ == "__main__":
    fix_google_fonts_integrity()
