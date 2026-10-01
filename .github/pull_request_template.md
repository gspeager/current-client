## What this changes

<!-- One or two sentences. Link the issue if there is one. -->

## How it was tested

<!-- Tests added or run, and anything checked by hand in the running app. -->

## Checklist

- [ ] Targets `develop`, and the title is a Conventional Commit (`feat(scope): …`, `fix: …`) — it becomes the squashed commit
- [ ] Commits are signed off (`git commit -s`)
- [ ] `go test . ./app/... ./core/... ./internal/...` and `cd frontend && npm test` pass
- [ ] No new network calls beyond Git talking to a repository's remotes
- [ ] If dependencies changed: `wails3 task notices` re-run and `wails3 task notices:check` passes
