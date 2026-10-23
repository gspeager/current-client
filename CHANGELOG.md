# Changelog

All notable changes to Current Client are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/) (pre-1.0: minor versions may include breaking changes).

## [Unreleased]

Host integration arrives in this release, for GitLab (gitlab.com and Self-Managed 16.0 or newer) and GitHub (github.com and Enterprise Server 3.11 or newer). It is **off until you turn it on, per host**: nothing contacts a host's API before then, and [docs/integrations.md](docs/integrations.md) lists every request it makes, when, and the token scope it needs.

### Added

- **Host integration, per host and off by default.** Settings → Integrations lists every host your remotes point at, with a switch, the provider type and API address worked out from the remote (editable), a token kept by Git's own credential helper (or Git's existing sign-in, with no setup), **Test connection** and **Remove**. A one-time prompt offers to turn a host on when you open a repository there; **Not now** is remembered. The status bar shows the API host actually contacted, when it was last asked, and the request budget the host last reported. ([#146](https://github.com/gspeager/current-client/issues/146), [#147](https://github.com/gspeager/current-client/issues/147), [#148](https://github.com/gspeager/current-client/issues/148), [#149](https://github.com/gspeager/current-client/issues/149))
- **A refresh is the only thing that fetches from a host**: the tab's button, opening a host panel for the first time, a fetch or a pull. Everything else reads what was cached, in memory only. ([#170](https://github.com/gspeager/current-client/issues/170))
- **Merge requests / pull requests on branches**, as a square marker beside the branch in the sidebar and on the graph: the number alone with integration off (from the refs the host publishes, read by the fetch you run), and the state and CI result with it on. Hover cards show the title, author, state and CI; issue and merge request references in commit messages link to the host and show their title. ([#154](https://github.com/gspeager/current-client/issues/154), [#161](https://github.com/gspeager/current-client/issues/161), [#168](https://github.com/gspeager/current-client/issues/168), [#176](https://github.com/gspeager/current-client/issues/176))
- **A Merge requests / Pull requests tab**: the open changes as cards with number, CI, title and branches, filtered to **Open**, **Mine** or **Closed** (closed ones are asked for only when that filter is chosen), with **Refresh** and **New merge request** in the header. Selecting one shows its author, branches, checks, conflicts and labels, **Check out locally**, which fetches the change's head and creates a branch named after it, **Open on host**, and tabs for the description (rendered safely: images are links, nothing is fetched), the pipeline's jobs, review threads, files and commits, each fetched the first time it is opened. A line under the list says when the host is contacted. ([#153](https://github.com/gspeager/current-client/issues/153), [#175](https://github.com/gspeager/current-client/issues/175), [#215](https://github.com/gspeager/current-client/issues/215))
- **The checked-out branch's merge request** at the top of the Branches sidebar, with its state, CI result and the failed jobs named, unresolved threads, and **Create merge request** when it has none. After a refresh, one line says what changed: new comments, a failed pipeline, the change merged. ([#177](https://github.com/gspeager/current-client/issues/177), [#178](https://github.com/gspeager/current-client/issues/178))
- **CI results** on commits in History and on branch tips, asked for as commits scroll into view; clicking one lists the jobs, and a failed job shows the end of its log in the app. ([#155](https://github.com/gspeager/current-client/issues/155), [#156](https://github.com/gspeager/current-client/issues/156))
- **Review threads beside the code**, read-only, under the line they were left on in Working Copy and in any diff, with unresolved counts in the file list. ([#157](https://github.com/gspeager/current-client/issues/157))
- **A conflict forecast**: on each refresh, the open merge requests that target the same base as your branch are merged with it in memory, and the ones that would conflict are named with the files. ([#160](https://github.com/gspeager/current-client/issues/160))
- **Open on host**: a commit, a file at a line, a branch, a tag or the repository, from their context menus, built from the remote's address with no request. ([#179](https://github.com/gspeager/current-client/issues/179))
- **Create a merge request / pull request**: after a push, a line offers the host's pre-filled form, with or without integration; with it on, a form in the app, prefilled from the branch's commits and the repository's template, with a draft toggle. This is the one action that needs write scope, and the app names the scope when the token lacks it. ([#158](https://github.com/gspeager/current-client/issues/158), [#180](https://github.com/gspeager/current-client/issues/180))
- **Clean up merged branches**, in the Branches sidebar's header, lists local branches whose work is already on the default branch, squash and rebase merges included, and deletes the ones you pick, with **Undo**. With host integration on, a branch whose merge request or pull request was merged on the host is listed with its number. A branch with commits newer than what was merged is marked and left unselected. When the checked-out branch is among them, the default branch is checked out and pulled first. ([#159](https://github.com/gspeager/current-client/issues/159))
- With host integration on, pushing, force-pushing or deleting on the remote a branch the host protects asks first, naming the branch and what the host forbids, instead of the push being rejected afterwards. The rules come from the last refresh; nothing is asked at push time. ([#163](https://github.com/gspeager/current-client/issues/163))
- Host integration works with GitHub, github.com and GitHub Enterprise Server 3.11 or newer, with the same features as GitLab: pull requests, including ones from forks, review comments beside the code, check runs and commit statuses as one CI result, Actions job logs, issue titles, creating a pull request, protected-branch warnings. ([#151](https://github.com/gspeager/current-client/issues/151), [#152](https://github.com/gspeager/current-client/issues/152))
- A machine-wide `policy.json` lets an organization lock a host's integration off or force it on, and fix background fetch, for every user of the machine. A managed setting shows disabled in Settings, labeled **Managed by your organization**. The file's place on each platform is in [docs/integrations.md](docs/integrations.md); with none, nothing changes. ([#145](https://github.com/gspeager/current-client/issues/145))
- A build without any of the host integration code, `wails3 build EXTRA_TAGS=nohost`, for deployments that must not carry it. **Copy diagnostics** never includes tokens or a host's answers, and can leave out hostnames. ([#150](https://github.com/gspeager/current-client/issues/150))
- The command palette gained Undo, Reflog, Bisect and Import patch. ([#139](https://github.com/gspeager/current-client/issues/139))

### Changed

- **Opening a repository is one dialog with Open, Clone and New tabs.** Open is a filter box, Browse and the recent cards (open repositories badged); Clone and New keep their fields on their own tabs; the footer shows the default location from Settings. The repository name in the header drops a list of recent repositories to switch to, with **Open another…** for the rest, instead of a second copy of the dialog; the branch list beside it groups branches as Checked out, Local and Remote like the sidebar. ([#222](https://github.com/gspeager/current-client/issues/222))
- **The conflict diff names its sides.** A conflicted file shows **Side by side** (your branch's version and the incoming branch's, each column named after its branch) or **Result** (the file as it is, markers included, with one line-number column and no hunk header), and a footer walks the conflict blocks. While a merge, rebase, cherry-pick or revert is in progress, checking out, new branch, Undo, Bisect and the header's Fetch / Pull / Push are dimmed with the reason. ([#221](https://github.com/gspeager/current-client/issues/221))
- **A merge, rebase, cherry-pick or revert in progress is a mode.** The commit composer becomes the operation's commit, prefilled with the message Git prepared, and its button is **Continue merge** (disabled, saying how many files are still conflicted, until all are resolved); the banner says what is merging into what and keeps **Abort**. Changed files group as **Conflicted** and **Resolved**, the Changes tab counts conflicted files, the repository chip and the checked-out branch read **merging**, the other branch reads **incoming**, and **Mark resolved** is the primary action above a conflicted diff. ([#220](https://github.com/gspeager/current-client/issues/220))
- **One sync cluster in the header.** **Fetch**, **Pull** and **Push** each have a menu: Fetch and prune, Fetch all remotes; Pull (merge), Pull (rebase); Push tags, Set upstream, Force-push (with lease). Fetch fetches `origin` (or the first remote); the Prune checkbox left the header for Settings → Network as **Prune on fetch**; "fetched … ago" is the Fetch button's tooltip and a status bar line. The status bar shows the repository's name, with the path on hover and copied on click. ([#219](https://github.com/gspeager/current-client/issues/219))
- **One list of commit actions in History.** The right-click menu, the row's button and the inspector offer the same actions in the same order: Check out, Branch here, Cherry-pick, Revert, **Reset to here…**, Export patch, Copy SHA, and Open on host. Reset is one item that opens a dialog naming the three modes, what each does and the `git reset` it is the same as, with hard marked destructive. The inspector is a fixed width, says how many files changed and which branches hold the commit, and labels the badge colour control; the toolbar keeps its search field at every width by folding time, type, author and Hide merges into a **Filters** menu with a count. Stashes are left out of the all-branches graph and the list footer says so. ([#218](https://github.com/gspeager/current-client/issues/218))
- **The sidebar is restructured.** HEAD is one line at the top, with your branch's pull request under it; every section folds from its header and keeps its count and add button while closed, empty sections start closed, and what you open or close is remembered per repository (the changed-files groups in Changes fold the same way). Branches come in **Checked out**, **Local** and **Remote** groups with a filter, a **conflicts** badge in place of the warning glyph, and the time on hover. The icon row became **Open in** (Terminal, file manager, your editor, the host) and **Tools** (Undo, Reflog, Bisect, Import patch, Maintenance) rows at the bottom; messages there can be dismissed. Remotes on disk show the end of their path rather than overflowing. ([#217](https://github.com/gspeager/current-client/issues/217))
- One name per concept across the app: the tabs read **Changes** (with the count of changed files), **History**, **Pull requests** / **Merge requests**, **Activity**, **Changelog**; branch menus say **Check out** like the buttons; the command palette is **Go to anything** and at rest lists local branches, recent repositories and actions rather than everything, with settings appearing once typed for; the sidebar's merge request section and the create dialog use the host's own word; the diff's side labels are **HEAD** and **Staged**. ([#216](https://github.com/gspeager/current-client/issues/216))

### What works at each level of access

| Feature | Integration off | Read-only token | Write scope |
| --- | --- | --- | --- |
| Links from issue and MR/PR references to the host | yes | yes | yes |
| Open on host: commit, file at line, branch, tag, repository | yes | yes | yes |
| Create MR/PR link after a push (the host's pre-filled form) | yes | yes | yes |
| PR numbers on branches, through `git ls-remote` with a fetch | yes (number only) | with state and CI | with state and CI |
| Clean up branches merged on the host | squash merges found locally | plus merged on the host | plus merged on the host |
| MR/PR list, details, local checkout | no | yes | yes |
| The checked-out branch's MR/PR in the sidebar, and what a refresh changed | no | yes | yes |
| CI status, job list, failed job log | no | yes | yes |
| Review threads beside the code | no | yes | yes |
| Conflict forecast against open MRs/PRs | no | yes | yes |
| Hover cards with title and state | local facts only | yes | yes |
| Create MR/PR in the app | no | no, says which scope | yes |
| Warn before pushing to a protected branch | no | yes | yes |

"Off" means the host is never contacted for it. A read-only token is GitLab `read_api`, or GitHub fine-grained read access to pull requests, checks, contents and issues.

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
