# Add Content Security Policy header with secure configuration
# Nonce will be injected by build script - placeholder: jJv1dXcHOmvbRO2xN0o2uQ==
add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'nonce-jJv1dXcHOmvbRO2xN0o2uQ=='; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self' http://localhost:8080; media-src 'self' blob: data:; frame-ancestors 'self'; form-action 'self';" always;

# Add X-Frame-Options header for additional clickjacking protection
add_header X-Frame-Options "SAMEORIGIN" always;

add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;

# Cross-Origin headers for Spectre protection
add_header Cross-Origin-Resource-Policy "same-origin" always;
add_header Cross-Origin-Embedder-Policy "require-corp" always;
add_header Cross-Origin-Opener-Policy "same-origin" always;
