import os
import re
import hashlib
import base64
from pathlib import Path

FRONTEND_DIST = Path('frontend/dist')
NGINX_CONF = Path('nginx.conf')

STYLE_TAG_RE = re.compile(r'<style[^>]*>([\s\S]*?)<\/style>', re.IGNORECASE)
STYLE_ATTR_RE = re.compile(r'style\s*=\s*"([^"]+)"|style\s*=\s*\'([^\']+)\'', re.IGNORECASE)

HASH_PREFIX = "'sha256-"
HASH_SUFFIX = "='"


def compute_csp_hash(style_content: str) -> str:
    # CSP requires the hash of the raw bytes of the style content
    digest = hashlib.sha256(style_content.encode('utf-8')).digest()
    b64 = base64.b64encode(digest).decode('utf-8')
    return f"'sha256-{b64}='"


def extract_inline_styles(file_path: Path):
    hashes = set()
    with open(file_path, encoding='utf-8', errors='ignore') as f:
        content = f.read()
        # <style>...</style>
        for match in STYLE_TAG_RE.finditer(content):
            style = match.group(1).strip()
            if style:
                hashes.add(compute_csp_hash(style))
        # style="..." or style='...'
        for match in STYLE_ATTR_RE.finditer(content):
            style = match.group(1) or match.group(2)
            if style:
                hashes.add(compute_csp_hash(style.strip()))
    return hashes


def find_all_hashes():
    all_hashes = set()
    for root, _, files in os.walk(FRONTEND_DIST):
        for file in files:
            if file.endswith(('.html', '.js', '.mjs', '.cjs', '.ts', '.tsx')):
                file_path = Path(root) / file
                all_hashes.update(extract_inline_styles(file_path))
    return all_hashes


def update_nginx_conf(new_hashes):
    with open(NGINX_CONF, 'r', encoding='utf-8') as f:
        conf = f.read()
    # Find the style-src line
    style_src_re = re.compile(r'(style-src [^;]+)', re.IGNORECASE)
    match = style_src_re.search(conf)
    if not match:
        print('No style-src directive found in nginx.conf!')
        return
    style_src = match.group(1)
    # Extract existing hashes
    existing_hashes = set(re.findall(r"'sha256-[^']+'", style_src))
    # Add new hashes
    updated_hashes = existing_hashes.union(new_hashes)
    # Rebuild the style-src directive
    style_src_new = re.sub(r"(style-src [^;]+)",
        lambda m: 'style-src ' + ' '.join(sorted(updated_hashes)) + ''.join([s for s in m.group(1).split(' ') if not s.startswith("'sha256-") and not s == 'style-src']),
        style_src)
    # Replace in conf
    conf_new = conf.replace(style_src, style_src_new)
    with open(NGINX_CONF, 'w', encoding='utf-8') as f:
        f.write(conf_new)
    print(f'Updated nginx.conf with {len(new_hashes)} new hashes.')


def main():
    print('Scanning for inline styles in frontend/dist...')
    hashes = find_all_hashes()
    if not hashes:
        print('No inline styles found.')
        return
    print(f'Found {len(hashes)} unique inline style hashes.')
    update_nginx_conf(hashes)

if __name__ == '__main__':
    main()
