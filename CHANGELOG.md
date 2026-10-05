# Changelog

All notable changes to Current Client are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/) (pre-1.0: minor versions may include breaking changes).

## [Unreleased]

### Added

- Worktrees: a **Worktrees** section in the Nav Pane lists, adds, opens and removes worktrees, so several branches can be checked out at once in their own folders. The branch list marks branches checked out in another worktree. ([#42](https://github.com/gspeager/current-client/issues/42))
- Stage, unstage or discard single lines: click the line numbers of the changes you want in a hunk, then use its **Stage lines** button. ([#41](https://github.com/gspeager/current-client/issues/41))
- **Search in files** (⌘⇧F / Ctrl+Shift+F, or from Quick Switch) finds text across the working tree or any branch or tag, with match case, whole word and regular expression options. Clicking a match opens it in your editor. ([#40](https://github.com/gspeager/current-client/issues/40))
- **Set upstream…** and **Unset upstream** in a branch's right-click menu, to choose or remove the remote branch it tracks without pushing. ([#38](https://github.com/gspeager/current-client/issues/38))
- Stash just some files: select them in Working Copy and choose **Stash** from the right-click menu. A new stash can also **Keep staged changes**, stashing only what isn't staged. ([#37](https://github.com/gspeager/current-client/issues/37))
- **Check out** a commit from History to look at an old version. The header and status bar show `HEAD` and the commit while it's detached. ([#36](https://github.com/gspeager/current-client/issues/36))
- Changed images are shown as images instead of "Binary file changed": side by side, with a swipe slider, or as an onion skin. Works for PNG, JPEG, GIF, WebP, BMP, ICO, AVIF and SVG. ([#23](https://github.com/gspeager/current-client/issues/23))
- Two more ways to merge a branch from its right-click menu: **Merge into current (no fast-forward)** always makes a merge commit, and **Squash into current** stages the branch's changes as one change for you to commit, with the squashed commits listed in the commit message to start from. ([#22](https://github.com/gspeager/current-client/issues/22))
- Pull with rebase: shift-click the header's Pull button, or choose **Pull (rebase)** in Quick Switch. Plain Pull still follows your `pull.rebase` setting. ([#21](https://github.com/gspeager/current-client/issues/21))
- **Delete on remote** for a tag removes it from the remote it was pushed to, so it no longer comes back on the next fetch. ([#20](https://github.com/gspeager/current-client/issues/20))
- **Delete on remote** removes a branch from its remote, from a branch's right-click menu or from a remote branch in the branch dropdown. ([#19](https://github.com/gspeager/current-client/issues/19))
- **Show changes** on a stash lists the files it changed, with a diff for each, including untracked files it saved. ([#18](https://github.com/gspeager/current-client/issues/18))
- A **Fetch** button next to "fetched … ago" in the header, and a **Prune** checkbox beside it that also removes remote branches deleted on the remote when you fetch. The checkbox is remembered. ([#14](https://github.com/gspeager/current-client/issues/14))

### Fixed

- **Revert** and **Cherry-pick** work on merge commits, using the changes the merge brought in, so a merged pull request can be backed out. They used to fail with Git's "no -m option was given" error. ([#35](https://github.com/gspeager/current-client/issues/35))
- Checking out a remote branch from the branch dropdown or Quick Switch works when the remote's name contains a `/`, such as `team/origin`. ([#33](https://github.com/gspeager/current-client/issues/33))
- Force push now always updates the branch's own upstream. Before, a branch tracking a differently named remote branch (local `feature` tracking `origin/feat-x`) was force-pushed to `origin/feature` instead, and a remote with a `/` in its name wasn't found. ([#24](https://github.com/gspeager/current-client/issues/24))
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
