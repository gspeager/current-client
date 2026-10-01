import { forwardRef, useMemo, useState } from 'react'
import { System } from '@wailsio/runtime'
import App, { type AppProps, type CurrentClientAppHandle } from './App'
import { DialogProvider } from './components/chrome/DialogProvider'
import { CurrentClientRootContext } from './lib/currentClientRoot'
// Bundled locally so fonts never load from a CDN.
import '@fontsource/geist-sans/400.css'
import '@fontsource/geist-sans/500.css'
import '@fontsource/geist-sans/600.css'
import '@fontsource/geist-sans/700.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import '@fontsource/jetbrains-mono/600.css'
import '@fontsource/jetbrains-mono/700.css'
import './styles/styles.scss'

export type { CurrentClientAppHandle }

export interface CurrentClientAppProps extends Omit<AppProps, 'handle'> {
  // False while a host app shows another view: Current Client stays mounted but hidden.
  active?: boolean
}

const CurrentClientApp = forwardRef<CurrentClientAppHandle, CurrentClientAppProps>(function CurrentClientApp(
  { active = true, ...appProps },
  handle,
) {
  const [element, setElement] = useState<HTMLDivElement | null>(null)
  const root = useMemo(() => ({ element, active }), [element, active])

  // The macOS window buttons are drawn over the page (hidden inset title bar),
  // so styles need to know to leave room for them.
  return (
    <div
      ref={setElement}
      className="current-client"
      hidden={!active}
      data-platform={System.IsMac() ? 'mac' : undefined}
    >
      <CurrentClientRootContext.Provider value={root}>
        <DialogProvider>
          <App {...appProps} handle={handle} />
        </DialogProvider>
      </CurrentClientRootContext.Provider>
    </div>
  )
})

export default CurrentClientApp
