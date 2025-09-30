import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import {randomBytes} from 'crypto'

// Custom plugin to generate CSP nonces
function cspNoncePlugin() {
  return {
    name: 'csp-nonce',
    transformIndexHtml: {
      order: 'pre' as const,
      handler(html: string) {
        // Only process in production or when CSP is enabled
        if (process.env.NODE_ENV === 'production') {
          const nonce = randomBytes(16).toString('base64')

          // Add nonce to script tags
          html = html.replace(
            /<script([^>]*)>/g,
            `<script$1 nonce="${nonce}">`
          )

          // Add nonce to style tags (if any)
          html = html.replace(
            /<style([^>]*)>/g,
            `<style$1 nonce="${nonce}">`
          )

          // Add a comment with the nonce for nginx to use
          html = html.replace(
            '</head>',
            `    <!-- CSP-NONCE: ${nonce} -->\n  </head>`
          )
        }
        return html
      }
    }
  }
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), cspNoncePlugin()],
  define: {
    // Use env variables for version info (set via Docker build args)
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(process.env.VITE_APP_VERSION || 'dev'),
    'import.meta.env.VITE_BUILD_TIME': JSON.stringify(process.env.VITE_BUILD_TIME || new Date().toISOString()),
    'import.meta.env.VITE_COMMIT_HASH': JSON.stringify(process.env.VITE_COMMIT_HASH || 'dev'),
  },
  esbuild: {
    // Use esbuild for faster TypeScript compilation
    loader: 'tsx',
    include: ['src/**/*.ts', 'src/**/*.tsx'],
    // Optimize for production builds
    target: 'es2020',
    minify: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  optimizeDeps: {
    include: ['react', 'react-dom', '@mantine/core', '@mantine/hooks'],
    // Force pre-bundling for better caching
    force: false,
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
    // Optimize build performance
    minify: 'esbuild',
    target: 'es2020',
    // Reduce chunk size for better caching
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          ui: ['@mantine/core', '@mantine/hooks'],
          // Add more chunks for better caching
          utils: ['axios', 'react-router-dom'],
        },
        // Optimize chunk naming for better caching
        chunkFileNames: 'assets/[name]-[hash].js',
        entryFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
      },
    },
  },
})
