import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import wails from '@wailsio/runtime/plugins/vite'
import { currentClientResolve } from './vite.shared.ts'

export default defineConfig({
  resolve: currentClientResolve,
  plugins: [react(), wails('./bindings')],
})
