# Add Content Security Policy header with secure configuration
# Nonce will be injected by build script - placeholder: rQv5+fXKpPEe2rqY+4/HDw==
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'nonce-rQv5+fXKpPEe2rqY+4/HDw=='; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self' http://localhost:8080 https://translation.googleapis.com; media-src 'self' blob: data:; frame-ancestors 'self'; form-action 'self';" always;

# add x-frame-options header for additional clickjacking protection
add_header x-frame-options "sameorigin" always;

add_header x-content-type-options "nosniff" always;
add_header x-xss-protection "1; mode=block" always;
add_header referrer-policy "strict-origin-when-cross-origin" always;

# cross-origin headers for spectre protection
add_header cross-origin-resource-policy "same-origin" always;
add_header cross-origin-embedder-policy "unsafe-none" always;
add_header cross-origin-opener-policy "same-origin" always;

# cache control - 4 hour max age for static assets
add_header cache-control "public, max-age=14400" always;
