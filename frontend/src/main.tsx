import React from 'react'
import ReactDOM from 'react-dom/client'
import CurrentClientApp from './CurrentClientApp'
import './styles/standalone.scss'

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <CurrentClientApp />
  </React.StrictMode>,
)
