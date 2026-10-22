## What this changes

<!-- One or two sentences. Link the issue if there is one. -->

## How it was tested

<!-- Tests added or run, and anything checked by hand in the running app. -->

## Checklist

- [ ] Targets `develop`, and the title is a Conventional Commit (`feat(scope): …`, `fix: …`) — it becomes the squashed commit
- [ ] Commits are signed off (`git commit -s`)
- [ ] `go test . ./app/... ./core/... ./internal/...` and `cd frontend && npm test` pass
- [ ] Any new network connection goes only to a destination the user configured, and is off by default unless it's a Git operation the user starts
- [ ] If dependencies changed: `wails3 task notices` re-run and `wails3 task notices:check` passes
