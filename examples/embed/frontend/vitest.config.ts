import { configDefaults, defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { currentClientResolve } from './vite.shared.ts'

export default defineConfig({
  plugins: [react()],
  resolve: currentClientResolve,
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.ts'],
    // Current Client's own tests run in Current Client's repository.
    exclude: [...configDefaults.exclude, 'current-client/**'],
    css: true,
    mockReset: true,
  },
})
