# Contributing to Current Client

Thanks for helping. This covers what belongs in Current Client, how to build and test it, and the rules every change has to follow.

## What belongs here

Current Client is a complete Git client for a developer working on their own machine. Changes that make that better — Git workflows, history, diffs, conflicts, performance, accessibility, platform support — are welcome. Team-collaboration features (shared reviews, comments, planning) are out of scope for this repository.

Before starting something large, open an issue to talk it through. Small fixes can go straight to a pull request.

## Privacy rules

These are not negotiable, and a change that breaks one won't be merged:

- No telemetry, analytics, tracking, advertising or remote logging.
- The only network traffic is Git talking to a repository's own remotes. Git traffic the user didn't start (such as background fetch) sits behind a setting that is off by default.
- No other network calls: no websites, CDNs, host APIs (GitHub/GitLab REST or GraphQL), AI services or other third parties. An optional integration must be off by default, opt-in, clearly visible when on, and work with self-hosted servers.
- No automatic update checks or crash-report uploads.
- No avatars or downloaded images; people are shown as local initials badges.
- No custom credential storage. Use Git's own mechanisms (credential helpers, SSH agent). Never log credentials, tokens or keys.

## Building and testing

You need Go 1.25+, Node.js 18+, the [Wails v3 CLI](https://v3.wails.io/quick-start/installation/) and Git 2.23+.

```bash
wails3 dev                                     # run with hot reload
go test . ./app/... ./core/... ./internal/... # Go tests
cd frontend && npm test                        # frontend tests
cd frontend && npm run lint && npx prettier --check src
wails3 build                                   # production build
```

Scope Go commands to `. ./app/... ./core/... ./internal/...`; `./...` also walks iOS/Android scaffolds Wails may generate under `build/`, which don't compile on their own.

For changes that could affect performance, run the opt-in large-repository timing check. It generates a 50,000-commit, 20,000-file repository and times the calls the UI makes:

```bash
LARGE_REPO_TIMING=1 go test -run TestLargeRepoTiming -v ./app
```

If you change dependencies, regenerate and check the third-party notices:

```bash
wails3 task notices
wails3 task notices:check
```

`core/` is a library other modules import, so its exported API changes carefully. Check it against the newest tag before opening a pull request; an incompatible change needs a reason in the PR:

```bash
wails3 task api:check
```

Add tests with your change: Go unit tests for parsing and logic, integration tests against real temporary Git repositories (never a hosted account), and component tests for UI behavior. Tests in `core/` share their Git helpers through `core/internal/gittest`.

## How the code is organized

- `core/` — the reusable library: runs Git, parses its output, diffs, the commit graph, identity badges, opening repositories, changelogs from Conventional Commits, undo, and conflict prediction. It must not import `internal/`, Wails or the settings file; pass it the values it needs.
- `internal/` — app-only code: settings, platform integration, file watching.
- `app/` — the Wails services the frontend calls. Keep them thin. `main.go` only creates the window and registers `app.Services()`.
- `frontend/` — React + TypeScript. The frontend never runs commands or touches the file system; everything goes through the Wails bindings. Other apps can mount it through `frontend/src/embed.ts`, so its styles stay scoped to the `.current-client` root and it imports bindings as `@current-client-bindings/…`, never by relative path.
  - `frontend/src/features/<area>/` — one folder per area of the app (`working-copy`, `diff`, `history`, `branches`, `remotes`, `activity`, `changelog`, `repositories`, `settings`), holding that area's view, components, hooks and helpers together.
  - `frontend/src/components/`, `lib/`, `styles/` — what more than one area uses: design-system controls and form fields, the app shell (header, navigation pane, status bar, dialogs), shared Git elements such as branch pills and identity badges, and general hooks and helpers. Code moves here once a second area needs it.
- `examples/embed/` — a host app that embeds Current Client. It has its own Go module; build and test it after changing `app/`, `CurrentClientApp` or anything it imports.

The settings file (`internal/config`) is versioned. If a change renames, removes or reshapes a field, bump `CurrentVersion` and add a migration step in `config.go`, so existing settings files keep working; adding a new optional field needs neither.

All Git commands go through `core/gitexec`. Arguments are always passed as a list, never built into a shell string.

## Code style

- Write the least code that does the job clearly. Don't duplicate logic; prefer a well-known package over custom code.
- Comments only where the code can't say it: an invariant, a non-obvious constraint, a reason. No comments that restate a name or signature.
- Go: `gofmt`. Frontend: ESLint and Prettier as configured.
- Styles are SCSS. Colors, spacing and sizes are CSS custom properties (`--token`), so light and dark themes can switch at runtime. Class names are lowercase with single hyphens.

## Design rules

- Geist for interface text; JetBrains Mono for data (paths, hashes, diffs, counts, branch names).
- The cyan accent means only three things: keyboard focus, `HEAD`, and the one primary action in a pane.
- Controls use the existing height tokens; don't add new sizes.
- Nothing animates longer than 140 ms, and nothing slides, scales, bounces or fades in on load.
- No illustrations, gradients or avatars. File paths truncate from the left so the file name stays visible.
- Labels are short and factual ("Unstage hunk", "Discard"), sentence case, no exclamation marks.

## Commits and pull requests

- Branch from `develop` and open pull requests against `develop`. `main` only receives releases, merged from `develop`.
- Pull requests are squash-merged, and the title becomes the commit on `develop`, so write the title as one short line in [Conventional Commits](https://www.conventionalcommits.org) form (`feat(history): filter commits by type`, `fix: keep lane color after rebase`), with `!` for breaking changes. No phase or ticket numbers. Current Client builds its changelog from these. Commits inside the pull request can be informal.
- Sign off every commit (`git commit -s`). This certifies the [Developer Certificate of Origin](https://developercertificate.org/): that you wrote the change or have the right to submit it under the project's license.
- By contributing you agree your work is licensed under the [Apache License 2.0](LICENSE).
- Keep pull requests focused on one change, and fill in the template.

## Reporting bugs and security issues

Use the bug report template and include the output of Settings → Diagnostics → **Copy diagnostics**. For security issues, follow [SECURITY.md](SECURITY.md) instead of opening a public issue.
