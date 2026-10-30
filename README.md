# Current Client

A privacy-first Git client for developers, on Windows, macOS and Linux.

Current Client runs entirely on your machine and connects only to the servers your repositories already use: no account, no telemetry, no automatic update checks. It works the same with GitHub, GitLab, Bitbucket, a self-hosted server or a plain SSH remote, including on air-gapped networks.

## Features

- Stage and unstage whole files, single hunks or single lines.
- Split and unified diffs, with word-level changes highlighted.
- A commit graph with search across the full history, blame and file history.
- Search across every file's contents, now or at any commit.
- Undo the last operation — a commit, amend, merge, pull, rebase, reset or branch switch — after a preview of what will move.
- An Inbox of what waits on you across every open repository: uncommitted work, stopped operations, unpushed, stale and merged branches, and with host integration on, reviews, failing CI, assigned issues and the host's to-do list, with a standup summary to copy.
- Conflict prediction: branches that would conflict with yours are marked, with the files, before you merge (Git 2.38 or newer).
- Code owners from the repository's `CODEOWNERS`, read locally: each changed file's owners in Changes, and the owners of a branch's changes suggested as reviewers when creating a merge request.
- Repository tabs, to keep several repositories open at once, and worktrees for several branches of one.
- An activity dashboard: commit calendar and velocity, contributors, file churn, languages, and stale branches or unpushed commits.
- A changelog writer that turns Conventional Commits into a Keep a Changelog `CHANGELOG.md`, or Markdown, HTML or PDF, and a Releases tab that keeps the next release's checklist current and takes it from tag to published release.
- Jira issue keys in commits and branch names link to your team's Jira site, and with that site's integration turned on, its issues show in hover cards, the issue panel, the Inbox and the new-branch picker.
- Optional host integration with GitLab and GitHub, off until you turn it on per host: merge requests and pull requests on branches and in a tab with their reviews and approvals, CI results, review threads beside the code with replies, line comments and reviews from the app, a conflict forecast, re-running failed CI jobs, where each environment is deployed and what has reached it, publishing a release from a tag, creating and merging a merge request from the app, starting a branch from an issue assigned to you, and reading, creating, commenting on and closing issues in the app. [docs/integrations.md](docs/integrations.md) lists every request it makes.

Current Client is pre-1.0 and has been tested most on Windows. It doesn't yet resolve conflicts inside the app or do a visual interactive rebase.

## Install

Download your platform's file from the [latest release](../../releases/latest). Current Client runs your installed Git, so you also need [Git](https://git-scm.com/downloads) 2.23 or newer. On a Mac, Git comes with Apple's Command Line Tools, and Current Client offers to install them if they're missing.

- **Windows:** run the `-setup.exe`. The installer isn't code-signed yet, so the first time you run it SmartScreen shows "Windows protected your PC". Choose **More info**, then **Run anyway**. It installs to `C:\Program Files\Current Client`.
- **macOS:** open the `.dmg` and drag Current Client to Applications. It's signed and notarized, so it opens without a warning.
- **Linux:** install the `.deb` (Debian, Ubuntu) or `.rpm` (Fedora, openSUSE) with your package manager, which also installs GTK 4 and WebKitGTK 6.0:

  ```bash
  sudo apt install ./current-client_*.deb
  sudo dnf install ./current-client-*.rpm
  ```

  The packages aren't signed yet. `apt` and `dnf` install a local file without asking, but a graphical software center may flag it as unverified, and a system set to accept only signed packages will refuse it.

## Privacy

- **Only destinations you chose.** Current Client connects to your repositories' Git remotes, the servers they're hosted on, and an issue tracker you configure by its address, such as your team's Jira site. Never to a vendor, CDN or other third party. If all your remotes and trackers are on an internal network, nothing leaves it.
- **Off until you turn it on.** Anything beyond the Git operations you start, such as background fetch, stays off until you enable it in Settings.
- **Nothing else.** No telemetry, analytics, automatic update checks or crash uploads. Settings has a **Check for updates** button that contacts github.com only when you click it, reads the newest release's version and date, and downloads nothing; the releases address is shown beside it for checking from another computer.
- **No accounts, no avatars.** People are shown as initials badges made on your machine.
- **No stored credentials.** Credentials stay with Git's credential helpers and your SSH agent; Current Client never stores or logs them.
- **Host integration is off until you turn it on, per host.** GitLab and GitHub features contact only the host your remote already points at, and only once you enable that host in Settings. [docs/integrations.md](docs/integrations.md) lists exactly what is contacted and when, and how to build without the code at all.

## Connecting to remotes

Current Client uses Git's own authentication and never shows Git's terminal prompts:

- **HTTPS:** when Git has no credentials for a remote, Current Client asks for a username and password or access token and hands them to Git for that operation. If it works, Git saves them with your credential helper (macOS Keychain, Git Credential Manager, `store`…) like any other sign-in; with no helper set up they're used once and forgotten. Git Credential Manager ships with Git for Windows and shows its own sign-in window instead.
- **SSH:** load your key into an SSH agent, and connect to a new host once from a terminal so it's in `known_hosts`.

Without those, a remote that needs an SSH passphrase or a host-key answer fails straight away with an explanation instead of waiting. Fetch, pull, push and clone can be cancelled while they run.

**Fetch** in the header fetches every remote. Tick **Prune** next to it to also remove remote branches that were deleted on the remote; the choice is remembered.

## Reporting a problem

Open an issue with the bug report template, and paste in the output of Settings → Diagnostics → **Copy diagnostics**: the app and Git versions, the OS and the last errors shown, with repository paths and your home folder hidden. Nothing is sent anywhere by the app.

Report security problems privately, as described in [SECURITY.md](SECURITY.md).

## Contributing

Current Client is a Go ([Wails v3](https://v3.wails.io/)) app with a React and TypeScript frontend. [CONTRIBUTING.md](CONTRIBUTING.md) covers building, testing and the rules every change follows. Changes are listed in [CHANGELOG.md](CHANGELOG.md).

## Using Current Client in another app

The Git layer is a Go library under `core/`, and the whole app can run inside another Wails app as one of its views; [examples/embed](examples/embed) shows how.

## License

[Apache License 2.0](LICENSE). Copyright 2026 Garrett Speager. Third-party components and their licenses are listed in [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES).

Contributions are accepted under the [Developer Certificate of Origin](https://developercertificate.org/): sign off each commit with `git commit -s`.
