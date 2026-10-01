import { fileURLToPath } from 'node:url'

// Current Client's source imports its Wails bindings as @current-client-bindings/…; an app
// embedding Current Client points the alias at its own generated bindings instead.
export const bindingsAlias = {
  '@current-client-bindings': fileURLToPath(new URL('./bindings/github.com/gspeager/current-client', import.meta.url)),
}
