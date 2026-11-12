import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ command, mode }) => {
  // Fail production builds (e.g. on Netlify) when the API URL is missing,
  // instead of silently shipping a site that calls http://localhost:8080.
  if (command === 'build' && mode === 'production') {
    const env = loadEnv(mode, process.cwd(), 'VITE_')
    const url = process.env.VITE_API_URL ?? env.VITE_API_URL
    if (!url) {
      throw new Error('VITE_API_URL is not set. Set it in Netlify > Site configuration > Environment variables.')
    }
    if (/localhost|127\.0\.0\.1/.test(url) && process.env.NETLIFY) {
      throw new Error(`VITE_API_URL points to ${url} — use the deployed backend URL on Netlify.`)
    }
  }

  return { plugins: [react()] }
})
