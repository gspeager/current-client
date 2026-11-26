import type { ReactElement } from 'react'
import { render, type RenderOptions } from '@testing-library/react'
import { DialogProvider } from '../components/chrome/DialogProvider'

// Renders inside the DialogProvider the app wraps everything in, rerenders too.
export function renderWithDialogs(ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>) {
  return render(ui, { wrapper: DialogProvider, ...options })
}
