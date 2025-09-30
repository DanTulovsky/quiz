#!/usr/bin/env python3
"""
Self-host Google Fonts script for the quiz application.

This script:
1. Analyzes the codebase to find all Google Fonts usage
2. Downloads the font files locally
3. Generates CSS for self-hosted fonts
4. Updates the codebase to use self-hosted fonts
5. Removes Google Fonts dependencies
"""

import os
import re
import json
import urllib.request
import urllib.parse
from pathlib import Path
from typing import Dict, List, Set, Tuple
import subprocess
import sys

class FontSelfHoster:
    def __init__(self, project_root: str = "."):
        self.project_root = Path(project_root)
        self.frontend_dir = self.project_root / "frontend"
        self.fonts_dir = self.frontend_dir / "public" / "fonts"
        self.fonts_css_file = Path("src/fonts.css")

        # Font families and weights found in the codebase
        self.fonts_to_host = {
            "Inter": {
                "weights": [300, 400, 500, 600, 700],
                "styles": ["normal"],
                "display": "swap"
            }
        }

        # Google Fonts API base URLs
        self.google_fonts_api = "https://fonts.googleapis.com/css2"
        self.google_fonts_static = "https://fonts.gstatic.com/s"

    def analyze_codebase(self) -> Dict[str, Set[str]]:
        """Analyze the codebase to find all Google Fonts usage."""
        print("🔍 Analyzing codebase for Google Fonts usage...")

        found_fonts = {}

        # Search patterns for different types of font usage
        patterns = [
            r'fonts\.googleapis\.com/css2\?family=([^&]+)',
            r'@import\s+url\([\'"]?https://fonts\.googleapis\.com/css2\?family=([^&\'"]+)',
            r'font-family:\s*[\'"]([^\'"]+)[\'"]',
            r'fontFamily:\s*[\'"]([^\'"]+)[\'"]',
        ]

        # Search in common file types
        extensions = ['.html', '.css', '.tsx', '.ts', '.js', '.jsx']

        for ext in extensions:
            for file_path in self.project_root.rglob(f"*{ext}"):
                if file_path.is_file() and not any(part.startswith('.') for part in file_path.parts):
                    try:
                        content = file_path.read_text(encoding='utf-8')

                        for pattern in patterns:
                            matches = re.findall(pattern, content, re.IGNORECASE)
                            for match in matches:
                                # Clean up the font family name
                                font_family = match.split(':')[0].strip()
                                if font_family not in found_fonts:
                                    found_fonts[font_family] = set()

                                # Try to extract weights if present
                                if ':' in match:
                                    weights_str = match.split(':')[1]
                                    weights = re.findall(r'(\d+)', weights_str)
                                    found_fonts[font_family].update(weights)

                                print(f"  Found font '{font_family}' in {file_path.relative_to(self.project_root)}")
                    except Exception as e:
                        print(f"  Warning: Could not read {file_path}: {e}")

        return found_fonts

    def download_font_files(self, font_family: str, weights: List[int], styles: List[str]) -> Dict[str, str]:
        """Download font files for a given font family and weights."""
        print(f"📥 Downloading {font_family} font files...")

        # Create fonts directory
        self.fonts_dir.mkdir(parents=True, exist_ok=True)

        downloaded_files = {}

        for weight in weights:
            for style in styles:
                # Build the Google Fonts URL
                params = {
                    'family': f'{font_family}:wght@{weight}',
                    'display': 'swap'
                }
                url = f"{self.google_fonts_api}?{urllib.parse.urlencode(params)}"

                try:
                    # Get the CSS file with modern browser user agent
                    print(f"  Fetching CSS for {font_family} {weight}...")
                    headers = {
                        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
                    }
                    req = urllib.request.Request(url, headers=headers)
                    with urllib.request.urlopen(req) as response:
                        css_content = response.read().decode('utf-8')

                    # Extract font file URLs from CSS
                    font_urls = re.findall(r'src:\s*url\(([^)]+)\)', css_content)

                    for font_url in font_urls:
                        # Clean up the URL
                        font_url = font_url.strip('"\'')
                        if font_url.startswith('//'):
                            font_url = 'https:' + font_url

                        # Determine file extension and prefer WOFF2, then WOFF
                        if '.woff2' in font_url:
                            ext = 'woff2'
                        elif '.woff' in font_url:
                            ext = 'woff'
                        elif '.ttf' in font_url:
                            ext = 'ttf'
                        else:
                            continue

                        # Generate local filename
                        filename = f"{font_family.lower()}-{weight}-{style}.{ext}"
                        local_path = self.fonts_dir / filename

                        # Download the font file
                        print(f"    Downloading {filename}...")
                        with urllib.request.urlopen(font_url) as font_response:
                            with open(local_path, 'wb') as f:
                                f.write(font_response.read())

                        downloaded_files[f"{weight}_{style}_{ext}"] = str(local_path.relative_to(self.frontend_dir))

                except Exception as e:
                    print(f"  Error downloading {font_family} {weight}: {e}")

        return downloaded_files

    def generate_font_css(self, font_family: str, downloaded_files: Dict[str, str]) -> str:
        """Generate CSS for self-hosted fonts."""
        print(f"🎨 Generating CSS for {font_family}...")

        css_lines = [f"/* Self-hosted {font_family} font */"]

        # Group files by weight
        weight_files = {}
        for key, file_path in downloaded_files.items():
            weight, style, ext = key.split('_')
            weight = int(weight)
            if weight not in weight_files:
                weight_files[weight] = {}
            weight_files[weight][ext] = file_path

        # Generate @font-face declarations for each weight
        for weight in sorted(weight_files.keys()):
            files = weight_files[weight]

            # Build src with proper format fallbacks
            src_parts = []
            if 'woff2' in files:
                # Convert public/fonts/ to /fonts/ for web URLs
                web_path = files['woff2'].replace('public/fonts/', '/fonts/')
                src_parts.append(f"url('{web_path}') format('woff2')")
            if 'woff' in files:
                web_path = files['woff'].replace('public/fonts/', '/fonts/')
                src_parts.append(f"url('{web_path}') format('woff')")
            if 'ttf' in files:
                web_path = files['ttf'].replace('public/fonts/', '/fonts/')
                src_parts.append(f"url('{web_path}') format('truetype')")

            if src_parts:
                css_lines.extend([
                    "",
                    f"@font-face {{",
                    f"  font-family: '{font_family}';",
                    f"  font-style: normal;",
                    f"  font-weight: {weight};",
                    f"  font-display: swap;",
                    f"  src: {', '.join(src_parts)};",
                    f"}}"
                ])

        css_lines.append("")
        return "\n".join(css_lines)

    def update_codebase(self, downloaded_files: Dict[str, str]):
        """Update the codebase to use self-hosted fonts."""
        print("🔧 Updating codebase to use self-hosted fonts...")

        # 1. Create fonts.css file
        css_content = self.generate_font_css("Inter", downloaded_files)
        self.fonts_css_file.write_text(css_content)
        print(f"  Created {self.fonts_css_file}")

        # 2. Update index.css to import local fonts instead of Google Fonts
        index_css_path = Path("src/index.css")
        if index_css_path.exists():
            content = index_css_path.read_text()

            # Replace Google Fonts import with local fonts import
            content = re.sub(
                r'@import url\([\'"]?https://fonts\.googleapis\.com/css2\?family=Inter[^)]*[\'"]?\);',
                "/* Self-hosted fonts */\n@import './fonts.css';",
                content
            )

            index_css_path.write_text(content)
            print(f"  Updated {index_css_path}")

        # 3. Update index.html to remove Google Fonts links
        index_html_path = Path("index.html")
        if index_html_path.exists():
            content = index_html_path.read_text()

            # Remove Google Fonts preconnect and stylesheet links
            content = re.sub(
                r'<link rel="preconnect" href="https://fonts\.googleapis\.com">\s*<link rel="preconnect" href="https://fonts\.gstatic\.com"[^>]*>\s*<link href="https://fonts\.googleapis\.com/css2[^>]*>',
                '',
                content
            )

            index_html_path.write_text(content)
            print(f"  Updated {index_html_path}")

        # 4. Update nginx.conf to remove Google Fonts from CSP
        nginx_conf_path = Path("../nginx.conf")
        if nginx_conf_path.exists():
            content = nginx_conf_path.read_text()

            # Remove Google Fonts from style-src and font-src
            content = re.sub(
                r'style-src \'self\' \'nonce-[^\']+\' https://fonts\.googleapis\.com;',
                "style-src 'self' 'nonce-jJv1dXcHOmvbRO2xN0o2uQ==';",
                content
            )
            content = re.sub(
                r'font-src \'self\' https://fonts\.gstatic\.com;',
                "font-src 'self';",
                content
            )

            nginx_conf_path.write_text(content)
            print(f"  Updated {nginx_conf_path}")

    def run(self):
        """Run the complete font self-hosting process."""
        print("🚀 Starting Google Fonts self-hosting process...")

        # Step 1: Analyze codebase
        found_fonts = self.analyze_codebase()
        print(f"Found fonts: {found_fonts}")

        # Step 2: Download font files for each font family
        all_downloaded_files = {}
        for font_family, config in self.fonts_to_host.items():
            downloaded_files = self.download_font_files(
                font_family,
                config["weights"],
                config["styles"]
            )
            print(f"Downloaded {len(downloaded_files)} files for {font_family}")
            all_downloaded_files.update(downloaded_files)

        # Step 3: Update codebase
        self.update_codebase(all_downloaded_files)

        print("✅ Font self-hosting complete!")
        print(f"📁 Font files saved to: {self.fonts_dir.relative_to(self.project_root)}")
        print(f"🎨 CSS generated at: {self.fonts_css_file.relative_to(self.project_root)}")
        print("\nNext steps:")
        print("1. Rebuild the frontend: task build-frontend")
        print("2. Test the application to ensure fonts load correctly")
        print("3. Update your CSP headers if needed")

def main():
    """Main entry point."""
    if len(sys.argv) > 1:
        project_root = sys.argv[1]
    else:
        project_root = "."

    hoster = FontSelfHoster(project_root)
    hoster.run()

if __name__ == "__main__":
    main()
