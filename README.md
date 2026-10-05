# Current Client

A privacy-first Git client for developers, on Windows, macOS and Linux.

Current Client runs entirely on your machine and talks only to your repositories' own Git remotes: no account, no telemetry, no update checks. It works the same with GitHub, GitLab, Bitbucket, a self-hosted server or a plain SSH remote, including on air-gapped networks.

## Features

- Stage and unstage whole files, single hunks or single lines.
- Split and unified diffs, with word-level changes highlighted.
- A commit graph with search across the full history, blame and file history.
- Search across every file's contents, now or at any commit.
- Undo the last operation — a commit, amend, merge, pull, rebase, reset or branch switch — after a preview of what will move.
- Conflict prediction: branches that would conflict with yours are marked, with the files, before you merge (Git 2.38 or newer).
- Repository tabs, to keep several repositories open at once, and worktrees for several branches of one.
- An activity dashboard: commit calendar and velocity, contributors, file churn, languages, and stale branches or unpushed commits.
- A changelog writer that turns Conventional Commits into a Keep a Changelog `CHANGELOG.md`, or Markdown, HTML or PDF.

Current Client is pre-1.0 and has been tested most on Windows. It doesn't yet resolve conflicts inside the app, do a visual interactive rebase, or manage submodules or Git LFS.

## Install

Download your platform's file from the [latest release](../../releases/latest). Current Client runs your installed Git, so you also need [Git](https://git-scm.com/downloads) 2.23 or newer.

- **Windows:** run the `-setup.exe`. The installer isn't code-signed yet, so the first time you run it SmartScreen shows "Windows protected your PC". Choose **More info**, then **Run anyway**. It installs to `C:\Program Files\Current Client`.
- **macOS:** open the `.dmg` and drag Current Client to Applications. It's signed and notarized, so it opens without a warning.
- **Linux:** install the `.deb` (Debian, Ubuntu) or `.rpm` (Fedora, openSUSE) with your package manager, which also installs GTK 4 and WebKitGTK 6.0:

  ```bash
  sudo apt install ./current-client_*.deb
  sudo dnf install ./current-client-*.rpm
  ```

  The packages aren't signed yet. `apt` and `dnf` install a local file without asking, but a graphical software center may flag it as unverified, and a system set to accept only signed packages will refuse it.

## Privacy

- **Only your Git remotes.** The only network traffic is Git talking to a repository's configured remotes. Background fetch is off until you turn it on in Settings.
- **Nothing else.** No telemetry, analytics, update checks, crash uploads or host APIs.
- **No accounts, no avatars.** People are shown as initials badges made on your machine.
- **No stored credentials.** Credentials stay with Git's credential helpers and your SSH agent; Current Client never stores or logs them.

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
