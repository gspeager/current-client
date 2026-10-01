import { createContext, useContext } from 'react'

export interface ConfirmOptions {
  title: string
  message: string
  confirmLabel: string
  destructive?: boolean
}

export interface PromptOptions {
  title: string
  label: string
  confirmLabel: string
  initialValue?: string
}

export interface Dialogs {
  confirm: (options: ConfirmOptions) => Promise<boolean>
  // Resolves with the trimmed value, or null when cancelled.
  prompt: (options: PromptOptions) => Promise<string | null>
}

export const DialogContext = createContext<Dialogs | null>(null)

export function useDialogs(): Dialogs {
  const dialogs = useContext(DialogContext)
  if (!dialogs) throw new Error('useDialogs must be used inside DialogProvider')
  return dialogs
}
