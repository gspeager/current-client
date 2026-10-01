import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { bindingsAlias } from './vite.shared.ts'

export default defineConfig({
  plugins: [react()],
  resolve: { alias: bindingsAlias },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: true,
    mockReset: true,
  },
})
