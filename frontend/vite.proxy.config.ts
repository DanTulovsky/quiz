// Auto-generated proxy config from nginx.conf
export const proxyConfig = {
  // Proxy ^/v1/auth/(login|signup)$ to http://localhost:8080
  '^/v1/auth/(login|signup)$': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  // Proxy ^/v1/quiz/ to http://localhost:8080
  '^/v1/quiz/': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  // Proxy ^/v1/admin/backend/ to http://localhost:8080
  '^/v1/admin/backend/': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  // Proxy ^/v1/admin/worker/ to http://localhost:8181
  '^/v1/admin/worker/': {
    target: 'http://localhost:8181',
    changeOrigin: true,
  },
  // Proxy /v1/audio/ to http://localhost:5050
  '/v1/audio/': {
    target: 'http://localhost:5050',
    changeOrigin: true,
  },
  // Proxy ^/v1/voices(.*)$ to http://localhost:5050
  '^/v1/voices(.*)$': {
    target: 'http://localhost:5050',
    changeOrigin: true,
  },
  // Proxy /v1/translate to http://localhost:8080
  '/v1/translate': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  // Proxy /v1/verb-conjugations/ to http://localhost:8080
  '/v1/verb-conjugations/': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  // Proxy /v1/ to http://localhost:8080
  '/v1/': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  // Proxy /health to http://localhost:8080
  '/health': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
};
