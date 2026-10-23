# Changelog

All notable changes to Current Client are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/) (pre-1.0: minor versions may include breaking changes).

## [Unreleased]

### Added

- A machine-wide `policy.json` lets an organization lock a host's integration off or force it on, and fix background fetch, for every user of the machine. A managed setting shows disabled in Settings, labeled **Managed by your organization**. The file's place on each platform is in [docs/integrations.md](docs/integrations.md); with none, nothing changes. ([#145](https://github.com/gspeager/current-client/issues/145))
- Host integration works with GitHub, github.com and GitHub Enterprise Server 3.11 or newer, with the same features as GitLab: pull requests, including ones from forks, review comments beside the code, check runs and commit statuses as one CI result, Actions job logs, issue titles, creating a pull request, protected-branch warnings. [docs/integrations.md](docs/integrations.md) lists every endpoint and the access each needs. ([#152](https://github.com/gspeager/current-client/issues/152))
- With host integration on, pushing, force-pushing or deleting on the remote a branch the host protects asks first, naming the branch and what the host forbids, instead of the push being rejected afterwards. The rules come from the last refresh; nothing is asked at push time. ([#163](https://github.com/gspeager/current-client/issues/163))
- **Clean up merged branches**, in the Branches sidebar's header, lists local branches whose work is already on the default branch, squash and rebase merges included, and deletes the ones you pick, with **Undo**. With host integration on, a branch whose merge request or pull request was merged on the host is listed with its number. A branch with commits newer than what was merged is marked and left unselected. When the checked-out branch is among them, the default branch is checked out and pulled first. ([#159](https://github.com/gspeager/current-client/issues/159))

## [0.3.0] - 2026-10-17

### Added

- Conflicted files can be resolved without leaving the app. From a file's menu or the buttons above its diff, **Keep main's version** or **Keep feature-x's version** resolves the whole file with one side, named by branch even during a rebase, and asks first. When one side deleted the file, it offers **Keep it deleted, as in main**. **Open in editor** opens the file at its first conflict marker, and **Mark resolved** asks first if markers remain. ([#113](https://github.com/gspeager/current-client/issues/113))
- On a Mac without Apple's Command Line Tools, the app says so and offers **Install Command Line Tools**, instead of Apple's install window appearing as soon as the app opens. **Check again** is offered whenever Git isn't found. ([#107](https://github.com/gspeager/current-client/issues/107))

### Changed

- Repositories open on **Working Copy**, now the first tab, with the first changed file already open: a conflicted file first, then unstaged, then staged. When the open file is staged, discarded or committed, the next one opens. The tabs are Working Copy, Branch Graph & History, Activity and Changelog. ([#114](https://github.com/gspeager/current-client/issues/114), [#118](https://github.com/gspeager/current-client/issues/118))
- Branch Graph & History shows all branches by default, so it draws a graph straight away. The branch filter you choose is remembered; a branch that no longer exists falls back to all branches. ([#119](https://github.com/gspeager/current-client/issues/119))
- The sidebar's Workspace actions are one row of icon buttons, named on hover, so Branches starts near the top. ([#115](https://github.com/gspeager/current-client/issues/115))
- Activity's Repository health no longer counts uncommitted changes as an issue; it still shows them, and still counts conflicts. The sprint-burst percentage under Commit velocity and the streaks under the activity calendar are gone. ([#116](https://github.com/gspeager/current-client/issues/116))
- Opening a repository runs about a third as many Git commands (22 instead of 70 in a repository with 16 branches), and each action refreshes once instead of twice. Answers that only change when a branch, tag or the config does are read once and reused, and tabs that aren't in view take one Git command each. This matters most on Windows, where each Git command costs more. ([#112](https://github.com/gspeager/current-client/issues/112))
- On Windows, the app runs Git for Windows' own `git.exe` directly instead of through the launcher on the `PATH`, which started a second process for every command. A Git executable chosen in Settings is used as it is. ([#106](https://github.com/gspeager/current-client/issues/106))
- Predicting which branches would conflict with yours runs one Git command for all of them instead of one per branch. Git 2.38 and 2.39 still check one branch at a time. ([#111](https://github.com/gspeager/current-client/issues/111))

### Fixed

- Pull and push progress and errors show only in the tab of the repository they're for. Switching tabs no longer leaves another repository's "Could not pull" in view, and its **Pull (merge)** and **Pull (rebase)** pull the repository that diverged. A pull or push keeps running in a background tab. ([#122](https://github.com/gspeager/current-client/issues/122))
- On macOS, the app goes idle again while files have uncommitted changes. Reading a modified file to show its status counted as a change to it, so the app refreshed about twice a second. ([#109](https://github.com/gspeager/current-client/issues/109))
- Conflict marks on branches no longer disappear when the repository has a branch with unrelated history, such as an orphan `gh-pages`.
- In the branch graph, a branch tip's lane starts at its commit instead of a short line above it that led nowhere. ([#108](https://github.com/gspeager/current-client/issues/108))
- Activity's **Repository age** counts from the repository's oldest commit instead of its newest, so an active repository no longer looks a few days old. ([#110](https://github.com/gspeager/current-client/issues/110))
- The status bar keeps the repository path on one line, shortened from the start so the repository's folder stays visible. The home folder shows as `~` on macOS and Linux, and clicking the path copies it in full. ([#120](https://github.com/gspeager/current-client/issues/120))
- **Reveal in File Explorer** says **Reveal in Finder** on macOS and **Open in File Manager** on Linux. ([#117](https://github.com/gspeager/current-client/issues/117))

## [0.2.1] - 2026-10-08

### Added

- Branch, file and commit rows have a **More actions** button, and Shift+F10 opens a row's menu from the keyboard, so their actions no longer need a right-click. The menu moves with the arrow keys and hands focus back when it closes. Background repository tabs and Quick Switch rows can be reached and are named for screen readers. ([#83](https://github.com/gspeager/current-client/issues/83))
- **Pull (merge)** and **Pull (rebase)** in a menu on the header's Pull button and in Quick Switch. ([#72](https://github.com/gspeager/current-client/issues/72))

### Changed

- Ahead and behind counts are shown on the header's Pull and Push buttons only. The banners that repeated them above every view are gone. ([#82](https://github.com/gspeager/current-client/issues/82))

### Fixed

- Pulling a branch that has diverged from its upstream says so and offers Pull (merge) or Pull (rebase), instead of "Git command failed." ([#72](https://github.com/gspeager/current-client/issues/72))
- When Git fails for a reason the app doesn't recognise, the message and **Copy diagnostics** give Git's own reason instead of only "Git command failed." ([#74](https://github.com/gspeager/current-client/issues/74))
- Branches, tags, fetches and stashes made outside the app, such as in a terminal, now refresh the sidebar, including in worktrees. ([#81](https://github.com/gspeager/current-client/issues/81))
- Settings saved close together (open tabs, pane widths, toggles) no longer overwrite each other, and a crash while saving can't lose them. An unreadable settings file is set aside as `config.json.broken`, with a notice, instead of breaking every later save. ([#73](https://github.com/gspeager/current-client/issues/73))
- A merge commit's details list the files and line counts it brought in, against its first parent, instead of "No file changes." ([#78](https://github.com/gspeager/current-client/issues/78))
- Clicking an untracked file shows its contents, and a new folder lists each file in it instead of one row. ([#77](https://github.com/gspeager/current-client/issues/77))
- Unstaging a new file's only hunk unstages the file, and staging a deleted file's only hunk stages the deletion. Both used to stage an empty file. ([#76](https://github.com/gspeager/current-client/issues/76))
- **Import Patch** reads plain diffs, such as those from Working Copy's **Export patch**, as well as format-patch files. ([#75](https://github.com/gspeager/current-client/issues/75))
- On macOS, opening the app from Finder no longer hides tools installed with Homebrew, such as Git LFS and credential helpers, or VS Code's `code` command. When no editor is found, **Open in Editor** says so instead of showing a raw error. ([#79](https://github.com/gspeager/current-client/issues/79))
- Activity no longer counts stashes as commits in its activity, velocity, streak and file churn figures. ([#80](https://github.com/gspeager/current-client/issues/80))
- Reflog dates each entry by when it happened, not by the date of the commit it points to. ([#71](https://github.com/gspeager/current-client/pull/71))
- Relative times such as "5 minutes ago" keep updating, and "fetched … ago" updates after a pull. ([#84](https://github.com/gspeager/current-client/issues/84))
- Very large files in staged, commit, compare and stash diffs wait for **Load anyway**, as unstaged ones already did. A staged rename diffs as an edit of the file. ([#84](https://github.com/gspeager/current-client/issues/84))
- Remote URLs containing spaces are read correctly, and a file dialog that fails says so instead of acting as if it were cancelled. ([#84](https://github.com/gspeager/current-client/issues/84))
- In a repository with no commits yet, Changelog, Activity and Branches say there are no commits yet, instead of "HEAD is not a commit", an age of Today and "No branches." Worktrees no longer lists the repository as its own only worktree. ([#85](https://github.com/gspeager/current-client/issues/85))
- Activity's stat strip wraps onto more rows in a narrow window instead of running past its card, and a long branch name is shortened, with the full name on hover. ([#101](https://github.com/gspeager/current-client/issues/101))

## [0.2.0] - 2026-10-06

### Added

- **Search in files** (⌘⇧F / Ctrl+Shift+F, or from Quick Switch) finds text across the working tree or any branch or tag, with match case, whole word and regular expression options. Clicking a match opens it in your editor. ([#40](https://github.com/gspeager/current-client/issues/40))
- Stage, unstage or discard single lines: click the line numbers of the changes you want in a hunk, then use its **Stage lines** button. ([#41](https://github.com/gspeager/current-client/issues/41))
- Worktrees: a **Worktrees** section in the Nav Pane lists, adds, opens and removes worktrees, so several branches can be checked out at once in their own folders. The branch list marks branches checked out in another worktree. ([#42](https://github.com/gspeager/current-client/issues/42))
- Submodules and Git LFS: a **Submodules** section (in repositories that have them) shows each one's state and can update or open them, and a changed submodule shows the commits it moved between instead of a commit hash. Files stored in Git LFS are labelled, show their size change instead of pointer text, and say when Git LFS isn't installed. ([#43](https://github.com/gspeager/current-client/issues/43))
- Changed images are shown as images instead of "Binary file changed": side by side, with a swipe slider, or as an onion skin. Works for PNG, JPEG, GIF, WebP, BMP, ICO, AVIF and SVG. ([#23](https://github.com/gspeager/current-client/issues/23))
- Clicking a changed file in a commit's details opens its diff in a large window, with the commit's other files beside it, instead of a cramped diff in the narrow pane. ([#58](https://github.com/gspeager/current-client/issues/58))
- **Expand diff** in Working Copy hides the file list so the diff runs across to the sidebar, for reviewing long or wide changes. ([#56](https://github.com/gspeager/current-client/issues/56))
- **Wrap lines** in the diff toolbar turns line wrapping off, so long lines stay on one row and the diff scrolls sideways. In a split diff each side keeps half the width with its own scroll bars, and scrolling one side scrolls the other. Remembered across the app. ([#57](https://github.com/gspeager/current-client/issues/57))
- **Show changes** on a stash lists the files it changed, with a diff for each, including untracked files it saved. ([#18](https://github.com/gspeager/current-client/issues/18))
- Stash just some files: select them in Working Copy, across Staged and Unstaged (shift-click for a range, ⌘- or Ctrl-click for single files), and choose **Stash** from the right-click menu. Other staged files stay out of the stash. A new stash can also **Keep staged changes**, stashing only what isn't staged. ([#37](https://github.com/gspeager/current-client/issues/37))
- Two more ways to merge a branch from its right-click menu: **Merge into current (no fast-forward)** always makes a merge commit, and **Squash into current** stages the branch's changes as one change for you to commit, with the squashed commits listed in the commit message to start from. ([#22](https://github.com/gspeager/current-client/issues/22))
- Pull with rebase: shift-click the header's Pull button, or choose **Pull (rebase)** in Quick Switch. Plain Pull still follows your `pull.rebase` setting. ([#21](https://github.com/gspeager/current-client/issues/21))
- **Check out** a commit from History to look at an old version. The header and status bar show `HEAD` and the commit while it's detached. ([#36](https://github.com/gspeager/current-client/issues/36))
- **Set upstream…** and **Unset upstream** in a branch's right-click menu, to choose or remove the remote branch it tracks without pushing. ([#38](https://github.com/gspeager/current-client/issues/38))
- **Delete on remote** removes a branch from its remote, from a branch's right-click menu or from a remote branch in the branch dropdown. ([#19](https://github.com/gspeager/current-client/issues/19))
- **Delete on remote** for a tag removes it from the remote it was pushed to, so it no longer comes back on the next fetch. ([#20](https://github.com/gspeager/current-client/issues/20))
- A **Fetch** button next to "fetched … ago" in the header, and a **Prune** checkbox beside it that also removes remote branches deleted on the remote whenever you fetch all remotes. The checkbox is remembered. ([#14](https://github.com/gspeager/current-client/issues/14))

### Fixed

- Force push now always updates the branch's own upstream. Before, a branch tracking a differently named remote branch (local `feature` tracking `origin/feat-x`) was force-pushed to `origin/feature` instead, and a remote with a `/` in its name wasn't found. ([#24](https://github.com/gspeager/current-client/issues/24))
- **Revert** and **Cherry-pick** work on merge commits, using the changes the merge brought in, so a merged pull request can be backed out. They used to fail with Git's "no -m option was given" error. ([#35](https://github.com/gspeager/current-client/issues/35))
- A commit with a long description no longer pushes its changed files out of reach in the Commit detail pane; the pane scrolls instead. ([#58](https://github.com/gspeager/current-client/issues/58))
- Switching branches, merging, pulling or cherry-picking over uncommitted or untracked changes now says so and suggests committing or stashing first, instead of "Git command failed."
- Merging a branch that's already merged says "Already up to date. Nothing to merge." instead of doing nothing.
- In Branch Graph & History, long branch and tag names no longer run over the author column. They shorten to fit, with the full name on hover, and the commit message stays visible.
- Checking out a remote branch from the branch dropdown or Quick Switch works when the remote's name contains a `/`, such as `team/origin`. ([#33](https://github.com/gspeager/current-client/issues/33))
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

[0.3.0]: https://github.com/gspeager/current-client/releases/tag/v0.3.0
[0.2.1]: https://github.com/gspeager/current-client/releases/tag/v0.2.1
[0.2.0]: https://github.com/gspeager/current-client/releases/tag/v0.2.0
[0.1.1]: https://github.com/gspeager/current-client/releases/tag/v0.1.1
[0.1.0]: https://github.com/gspeager/current-client/releases/tag/v0.1.0
