import { fileURLToPath } from 'node:url'

const here = (path: string) => fileURLToPath(new URL(path, import.meta.url))

// @current-client is Current Client's frontend source, linked by scripts/link-current-client.mjs. Its
// @current-client-bindings imports use this app's generated bindings, which include
// Current Client's services because main.go registers them.
export const currentClientResolve = {
  alias: {
    '@current-client': here('./current-client'),
    '@current-client-bindings': here('./bindings/github.com/gspeager/current-client'),
  },
  // Current Client's own imports (react, lucide-react, …) resolve from this app's
  // node_modules, not from where the link points.
  preserveSymlinks: true,
}
