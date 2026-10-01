# Embedding Current Client

A minimal app that hosts Current Client next to a view of its own, in one window with a toggle. It's a starting point for a new host app.

- `main.go` keeps its settings in its own folder, then registers Current Client's services.
- `frontend/scripts/link-current-client.mjs` links `frontend/current-client` to Current Client's `frontend/src` at the version `go.mod` pins, and fails if `package.json` doesn't match Current Client's dependency versions. It runs before `dev`, `build` and `test`.
- `frontend/src/Host.tsx` mounts `CurrentClientApp`, keeps it mounted while the host view shows, puts the toggle in Current Client's header, and shares the open repository.

Here `go.mod` points at this repository with a `replace`; a real host requires a tagged Current Client version instead.

## Build

```bash
cd examples/embed
wails3 generate bindings -ts          # frontend/bindings, including Current Client's services
cd frontend && npm install && npm test && npm run build && cd ..
go build -o bin/ .
```

Run `bin/embed` (`bin\embed.exe` on Windows).
