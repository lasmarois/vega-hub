import path from 'path'
import fs from 'fs'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Get vega-hub port from:
// 1. VEGA_HUB_PORT environment variable
// 2. .vega-hub.port file in parent directories
// 3. Default to 8080
function getVegaHubPort(): number {
  // Check environment variable first
  if (process.env.VEGA_HUB_PORT) {
    const port = parseInt(process.env.VEGA_HUB_PORT, 10)
    if (!isNaN(port)) return port
  }

  // Try to find .vega-hub.port file (walk up from web/ directory)
  let dir = __dirname
  for (let i = 0; i < 5; i++) {
    const portFile = path.join(dir, '..', '.vega-hub.port')
    try {
      const content = fs.readFileSync(portFile, 'utf-8').trim()
      const port = parseInt(content, 10)
      if (!isNaN(port)) return port
    } catch {
      // File doesn't exist, try parent
    }
    dir = path.dirname(dir)
  }

  // Default port
  return 8080
}

const vegaHubPort = getVegaHubPort()

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: true,
    port: 5173,
    proxy: {
      '/api': {
        target: `http://localhost:${vegaHubPort}`,
        changeOrigin: true,
        // SSE support - disable buffering
        configure: (proxy) => {
          proxy.on('proxyRes', (proxyRes) => {
            // Disable buffering for SSE
            if (proxyRes.headers['content-type']?.includes('text/event-stream')) {
              proxyRes.headers['Cache-Control'] = 'no-cache'
              proxyRes.headers['Connection'] = 'keep-alive'
            }
          })
        },
      },
    },
  },
})
