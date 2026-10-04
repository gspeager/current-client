# Changelog

All notable changes to Current Client are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/) (pre-1.0: minor versions may include breaking changes).

## [Unreleased]

### Added

- **Delete on remote** removes a branch from its remote, from a branch's right-click menu or from a remote branch in the branch dropdown. ([#19](https://github.com/gspeager/current-client/issues/19))
- A **Fetch** button next to "fetched … ago" in the header, and a **Prune** checkbox beside it that also removes remote branches deleted on the remote when you fetch. The checkbox is remembered. ([#14](https://github.com/gspeager/current-client/issues/14))

### Fixed

- On Windows the installer puts Current Client in `C:\Program Files\Current Client`, rather than in a folder named after the publisher. Installing over 0.1.1 or earlier removes the old `C:\Program Files\Garrett Speager\Current Client` folder. ([#15](https://github.com/gspeager/current-client/issues/15))

## [0.1.1] - 2026-10-01

### Added

- Signing in to HTTPS remotes from the app. When Git has no credentials for a remote, or the remote turns them down, clone, fetch, pull and push ask for a username and password or access token and try again. Git saves the sign-in with your credential helper, such as the macOS Keychain or Git Credential Manager; Current Client stores nothing. ([#2](https://github.com/gspeager/current-client/issues/2))

### Fixed

- Cloning now creates a folder named after the repository inside the destination you choose, as `git clone` does. The folder name can be changed, and the full path is shown before cloning. ([#1](https://github.com/gspeager/current-client/issues/1))
- Activity, Branch Graph & History and Changelog no longer reload and redraw when you return to them; they refresh only when commits or branches have changed. ([#3](https://github.com/gspeager/current-client/issues/3))
- Spaces and other characters Git doesn't allow in branch names are turned into dashes, and the name is shown before the branch is created. ([#4](https://github.com/gspeager/current-client/issues/4))
- On macOS the app is named Current Client in Finder and Applications, rather than current-client. If you installed 0.1.0, delete the old `current-client` app from Applications after installing this one.

## [0.1.0] - 2026-09-30

First public version.

### Added

- Staging and unstaging by file or hunk, and a commit composer with amend and co-authors.
- Split and unified diffs with word-level changes.
- Commit graph with full-history search, blame and file history.
- Undo for the last commit, merge, pull, rebase, reset or branch switch.
- Conflict prediction for branches before merging, and continue or abort for merges and rebases in progress.
- Branches, remotes, tags and stashes, with cancellable fetch, pull and push.
- Repository tabs and an activity dashboard.
- A Changelog tab that writes Conventional Commits into `CHANGELOG.md`, or saves them as Markdown, HTML or PDF.
- The Git layer as a Go library under `core/`, and an example of embedding the app in another Wails app.

[0.1.1]: https://github.com/gspeager/current-client/releases/tag/v0.1.1
[0.1.0]: https://github.com/gspeager/current-client/releases/tag/v0.1.0
