# Host integration

Current Client can show what a Git host knows about a repository: merge requests / pull requests, CI results and review threads, from GitLab (gitlab.com or Self-Managed) and GitHub (github.com or Enterprise Server). This guide says exactly when it talks to a host, what it asks for, what it keeps and how to turn it off or keep it off. It is being written as the feature is built for 0.4.0; the sections marked *to come* are filled in as each part lands.

The rules it works under are the [privacy rules](../CONTRIBUTING.md#privacy-rules): the only destination is the host the repository's remotes already point at, and nothing is contacted until someone turns integration on for that host.

## Off until turned on

Integration is per host and off by default. Turning it on is a deliberate choice in Settings → Integrations, or a first-use prompt that names the host and says what would be contacted. Until then no host UI appears and nothing is sent. Turning a host off clears everything cached from it and deletes the refs fetched for it (below).

The set of enabled hosts is decided by the app, not by `core/`. `core/` cannot read the settings file; it is handed the set by whichever app embeds it (Current Client passes what Settings says, other apps pass their own), and it refuses any host outside that set before a connection is opened. That is what makes "integration off means no request" a property of the library, not of each caller.

## Which host, and which API

A repository's host is worked out from its remote URLs alone: HTTPS, `ssh://` and scp-like `git@host:group/repo.git` forms, with ports and nested GitLab groups. An SSH alias from `~/.ssh/config` (`git@work:group/repo`) is resolved with `ssh -G`, which prints the configured hostname without connecting. Nothing probes the server.

github.com and gitlab.com are recognized by name. Any other host is taken for GitLab when its name contains "gitlab" or the project path has nested groups, and for GitHub otherwise. The type can be changed per host in Settings → Integrations, and the links the app builds for that host follow the choice.

The API address is derived from the type, and can be edited for unusual installs such as a custom port or GitLab served under a relative URL:

| Host | API address |
| --- | --- |
| gitlab.com, GitLab Self-Managed | `https://<host>/api/v4` |
| github.com | `https://api.github.com` |
| GitHub Enterprise Server | `https://<host>/api/v3` |

Settings and the host indicator show the address actually contacted, `api.github.com` rather than `github.com`.

## When requests happen

A **refresh** is the only thing that fetches from a host:

- The refresh button in a host panel.
- Opening a host panel for the first time in a session.
- A Git fetch or pull the user starts.
- A push refreshes only the pushed branch's MR/PR and its CI.

Nothing else does. Switching branches, scrolling history, drawing the sidebar or graph, hovering a marker or opening a diff reads what is cached and never makes a request. With background refresh on (off by default, like background fetch) the same refresh runs on a timer.

With integration off, the only host-related traffic is Git's own: a fetch the user starts also runs one `git ls-remote` against the remote, which lists the pull request / merge request heads the host publishes (`refs/pull/*` on GitHub, Gitea and Forgejo, `refs/merge-requests/*` on GitLab) and gives each branch its open change's number. Bitbucket publishes none, so it gives nothing. A change counts only while its head is still a branch tip, which drops every change whose branch was deleted on merge. GitHub also publishes `refs/pull/N/merge` only while a pull request is open, so there the number is an open one. GitLab keeps merge refs after closing, so Git alone cannot tell a closed merge request whose branch was kept from an open one: with integration off the branch shows the number without a state, and the state comes from the API once integration is on.

## What is cached

One cache per repository, keyed by host and repository, holding MRs/PRs, CI results, review threads and issue titles with the time each was fetched. It lives in memory only; nothing from a host is written to disk.

- Data stays visible after it goes stale, labeled with its age, until a refresh replaces it. Offline, the last answer shows.
- The cache is cleared when integration is turned off for the host or the repository is closed.

## Refs on disk

MR/PR heads are Git refs, so fetching one (to check it out, or to forecast conflicts) does land in the repository. They go under `refs/current-client/<remote>/mr/<number>`, never `refs/remotes/`, so they never appear as remote branches in the sidebar or graph. They are deleted when the host is turned off or the repository is removed from the app, and can be removed by hand with `git update-ref -d`.

## Endpoints and scopes

*To come with each provider.* Every endpoint called, the user action that triggers it and the token scope it needs, for GitLab and for GitHub. Read-only scope is enough for everything except creating a merge request / pull request.

## What works at each level

*To come.* Per feature: what works with integration off (links to the host, PR numbers found through `git ls-remote`, local detection of squash-merged branches), with a read-only token, and with write scope.

## Credentials

Current Client stores no credentials of its own. Two ways to a token, both through Git's credential helper:

- **Git's own sign-in.** With no token saved, Current Client asks Git's credential helper for the host the remote points at (`github.com`, not `api.github.com`), with `git credential fill`. Git Credential Manager often already holds a token there, so many people need no setup. This runs only on a user action, since a helper may show its own window, and once per host per session.
- **A personal access token**, for SSH-only remotes or when the helper has nothing. It is pasted once in Settings → Integrations and handed to Git with `git credential approve`, so the user's own helper keeps it (the OS keychain, Git Credential Manager, `store`…). **Remove** hands it to `git credential reject`. The settings file records only that a token exists, never the token.

The token is saved under its own key, `https://current-client-api@api.<host>`, not under the host itself. A fetch or push asks the helper for the host alone, and a helper answers with any entry it has for that host, so a read-only token stored under the host would be tried for a push, rejected, and erased by Git. No Git operation ever asks for `api.<host>`, so the two can't collide. The token reaches Git on standard input, never on a command line, and never appears in logs or Diagnostics.

**Scopes.** Read-only is enough for everything but creating a merge request / pull request: GitLab `read_api`; GitHub fine-grained read access to pull requests, checks, contents and issues. The per-endpoint table below names the scope each feature needs.

**Errors** are plain messages, never the host's answer: a rejected token (401) says "The token for `<host>` was rejected. Update it in Settings → Integrations."; a missing scope (403) names the scope the action needs. A rejected token is forgotten so the next request asks Git again, which is how a sign-in renewed in Git Credential Manager gets picked up.

## Certificates and proxies

Host API traffic trusts and routes exactly as Git does for the same URL. Before the first request to a host, Current Client reads `git config --get-urlmatch http <API address>` from your global and system Git configuration and applies:

- `http.sslCAInfo` and `http.sslCAPath`: your internal certificate authority, used instead of the system trust store, as Git uses it.
- `http.proxy`, and otherwise the `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` environment, as Git falls back to them.
- `http.sslVerify = false` (or `GIT_SSL_NO_VERIFY`): shown as a warning in Settings. Certificate checks are never turned off; configure the authority instead.

So a self-hosted GitLab behind a corporate proxy with a private CA needs no setup beyond what `git fetch` already needed. `GIT_SSL_CAINFO` and `GIT_SSL_CAPATH` in the environment override the configuration, as they do for Git.

Every request has a timeout and can be cancelled. When a host answers 429, or 403 with its rate limit exhausted, Current Client waits for the time the host named (`Retry-After`, GitHub's `X-RateLimit-Reset` or GitLab's `RateLimit-Reset`, at most an hour) and sends nothing to that host until then; the panel shows a quiet message. Paged answers are followed through the `Link` header up to a fixed number of pages, never off the API address, and the panel says when a list was cut. A host being unreachable never affects Git operations: it becomes a plain message in the panel and in Diagnostics' recent errors, without the response body.

## Building without host integration

*To come.* A build tag that compiles host integration out entirely.
