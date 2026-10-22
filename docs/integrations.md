# Host integration

Current Client can show what a Git host knows about a repository: merge requests / pull requests, CI results and review threads, from GitLab (gitlab.com or Self-Managed) and GitHub (github.com or Enterprise Server). This guide says exactly when it talks to a host, what it asks for, what it keeps and how to turn it off or keep it off. It is being written as the feature is built for 0.4.0; the sections marked *to come* are filled in as each part lands.

The rules it works under are the [privacy rules](../CONTRIBUTING.md#privacy-rules): the only destination is the host the repository's remotes already point at, and nothing is contacted until someone turns integration on for that host.

## Off until turned on

Integration is per host and off by default. Turning it on is a deliberate choice: the switch in Settings → Integrations, or the first-use prompt shown once when a repository on a supported host is opened, which names the host and says exactly what would be contacted ("Current Client calls its API when you open merge requests or refresh, and at no other time"). **Not now** is remembered per host. Showing the prompt reads the remote's address and makes no request. Until a host is on, no host UI appears and nothing is sent. Turning a host off clears everything cached from it and deletes the refs fetched for it (below).

Settings → Integrations lists every host found in the open repository's remotes plus any added by hand, each with its switch, provider type, API address, credential status, **Test connection** (the one action on the page that contacts the host, and only while it is on) and **Remove**. While integration is on for the current repository, the status bar shows the API host actually contacted (`api.github.com`, not `github.com`) and when it was last asked ("no requests yet", "last request 2m ago", "request in flight"), so the promise can be watched rather than taken on trust; clicking it opens Settings.

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

Nothing else does, with one bounded exception: CI results for commits that come into view in History are asked for as you scroll, in batches, each commit once until the next refresh. Switching branches, drawing the sidebar or graph, hovering a marker or opening a diff reads what is cached and never makes a request. There is no separate background refresh: a fetch is a refresh, so the background fetch setting (off by default) covers the host too.

With integration off, the only host-related traffic is Git's own: a fetch the user starts also runs one `git ls-remote` against the remote, which lists the pull request / merge request heads the host publishes (`refs/pull/*` on GitHub, Gitea and Forgejo, `refs/merge-requests/*` on GitLab) and gives each branch its open change's number. Bitbucket publishes none, so it gives nothing. A change counts only while its head is still a branch tip, which drops every change whose branch was deleted on merge. GitHub also publishes `refs/pull/N/merge` only while a pull request is open, so there the number is an open one. GitLab keeps merge refs after closing, so Git alone cannot tell a closed merge request whose branch was kept from an open one: with integration off the branch shows the number without a state, and the state comes from the API once integration is on.

## What is cached

One cache per repository, keyed by host and repository, holding MRs/PRs, CI results, review threads and issue titles with the time each was fetched. It lives in memory only; nothing from a host is written to disk.

- Data stays visible after it goes stale, labeled with its age, until a refresh replaces it. Offline, the last answer shows.
- The cache is cleared when integration is turned off for the host or the repository is closed.

## The merge request / pull request tab

While a host is on for the repository, a tab named **Merge requests** (GitLab) or **Pull requests** (GitHub) lists the open changes from the last refresh: number, title, author as an initials badge, source and target branch, draft, labels, CI result and when it was updated, with an **Authored by me** filter. Opening the tab for the first time in a session refreshes it; after that it reads the cache until **Refresh**, a fetch or a pull. The list stops at the page cap and says so.

Selecting a change shows its description, rendered through the one sanitizing renderer (images are links, nothing is fetched), its branches, labels, CI and conflict state, **Open on host** and **Check out locally**. Checking out fetches the change's head from the remote into `refs/current-client/<remote>/mr/<number>` (ordinary Git traffic, with Git's own credentials) and creates a local branch at it named after the source branch, or `<owner>/<branch>` for a change from a fork; a branch that already exists is checked out as it is. The diff, history and blame views then work on it as on any branch.

## The checked-out branch's merge request

While the host is on and the checked-out branch's upstream has a merge request / pull request in the cache, the top of the Branches sidebar shows it: number, title, state, draft, target branch, the CI result with the failed jobs named (a refresh fetches the jobs of a failing pipeline for this one branch, so no click is needed), unresolved threads, **Details** (the merge request tab) and **Open on host**. It reads the cache; checking out a branch makes no request. Review threads, the conflict forecast and the clean-up offer join it as they land.

## CI results

With the host on, commits in History and branch tips in the sidebar show a small square in the pipeline's color: passed, failed, canceled, or an outline while running or pending. For commits the result is asked for when the commit scrolls into view, in batches of up to 50 at a time and never twice for the same commit until the next refresh; a commit without a pipeline is remembered as such. Branch tips and open merge requests get theirs on refresh. Clicking a square lists the pipeline's jobs with stage, status and duration, fetched only then, with **Open on host**. HEAD's cyan is never used for CI state.

## Hover cards

Pausing the pointer on a merge request marker, a CI square or an issue or merge request reference, or giving one keyboard focus, shows a card with what the cache holds: for a change, its title, author, state, branches, CI result, unresolved threads and when it was updated; for a CI square, the pipeline state and, once the jobs were opened, the failed job names and total duration; for a reference, the title and state. Hovering never makes a request. Issue titles are asked for when the History page they appear on loads, at most 50 at a time and each once until the next refresh. With integration off, a card shows only what Git supplied: the number and the link, and says the rest needs the host.

## What a branch row shows

Every branch whose upstream has a merge request / pull request gets a square marker beside its pill, in the sidebar and on the graph's remote-tracking refs: the number, and when the host is on, the state (draft, open, merged, closed) and the CI result. Clicking it opens the change on the host. The marker is drawn from the cache: with the host off, the number comes from the `git ls-remote` a fetch runs (and on GitLab the state stays unknown, as Git alone can't tell a closed merge request with a kept branch from an open one); with the host on, a fetch or pull also refreshes the open changes. Drawing the sidebar or graph never makes a request.

## Refs on disk

MR/PR heads are Git refs, so fetching one (to check it out, or to forecast conflicts) does land in the repository. They go under `refs/current-client/<remote>/mr/<number>`, never `refs/remotes/`, so they never appear as remote branches in the sidebar or graph. They are deleted when the host is turned off or the repository is removed from the app, and can be removed by hand with `git update-ref -d`.

## Host content is untrusted

A merge request description or a review comment is text anyone with an account on the host may have written, and it is shown inside a page that has bindings to run Git. Two things keep it inert:

- **A Content-Security-Policy on the page**, set as a response header by the asset server on every build: scripts and fonts only from the app itself, images only from the app or `data:` URLs, connections only to the app's own bridge, no frames, objects or form targets. Inline event handlers, `javascript:` links and remote images are then stopped by the browser, before any renderer is involved. A dev build additionally allows Vite's inline hot-reload preamble and its socket; a release build does not.
- **One renderer for all host text** (`core/hostcontent`): Markdown in, HTML out. Raw HTML in the source is dropped, not escaped into view. Only `http(s)` links are kept; relative links are resolved against the project's web address, and any other scheme becomes plain text. Images are never fetched: they render as a link to their address. Links open in your own browser through the same opener as every other link in the app, which takes only `http(s)`.

The release check in RELEASING.md opens a description containing an image hosted elsewhere and confirms the firewall log shows no request to that host.

## Endpoints and scopes

Every endpoint a provider calls, what triggers it, and the token scope it needs. Read-only scope is enough for everything except creating a merge request / pull request. Nothing is called before integration is on for the host and one of the triggers happens.

### GitLab (gitlab.com and Self-Managed, 16.0 or newer)

All under the host's `/api/v4`. Project paths are URL-encoded (`group%2Fproject`).

| Endpoint | When | Scope |
| --- | --- | --- |
| `GET /user` | Test connection | `read_api` |
| `GET /projects/:path/merge_requests?state=opened` | Refresh: the open merge requests, paged up to the cap | `read_api` |
| `GET /projects/:path/merge_requests/:iid` | Refresh of one merge request (the pushed branch's, the checked-out branch's) | `read_api` |
| `GET /projects/:id` | Once per fork a merge request comes from, to name it | `read_api` |
| `GET /projects/:path/merge_requests/:iid/discussions` | Refresh, for the checked-out branch's merge request | `read_api` |
| `GET /projects/:path/pipelines?sha=` | Refresh, for open merge requests' heads; and as commits scroll into view, once per commit until the next refresh | `read_api` |
| `GET /projects/:path/pipelines/:id/jobs` | Clicking a CI mark; on refresh, for the checked-out branch's merge request when its pipeline failed | `read_api` |
| `GET /projects/:path/jobs/:id/trace` | Clicking a failed job | `read_api` |
| `GET /projects/:path/issues/:iid` | When a History page with `#n` references loads, each number once until the next refresh, at most 50 at a time | `read_api` |
| `POST /projects/:path/merge_requests` | Create merge request | `api` |

Avatar URLs in these answers are never decoded. GitHub's table arrives with its provider.

## What works at each level

| Feature | Integration off | Read-only token | Write scope |
| --- | --- | --- | --- |
| Links from issue and MR/PR references to the host | yes | yes | yes |
| Open on host: commit, file at line, branch, tag, repository | yes | yes | yes |
| Create MR/PR link after a push (the host's pre-filled form) | yes | yes | yes |
| PR numbers on branches, through `git ls-remote` with a fetch | yes (number only) | with state and CI | with state and CI |
| Clean up branches merged on the host | squash merges found locally | plus merged on the host | plus merged on the host |
| MR/PR list, details, local checkout | no | yes | yes |
| CI status, job list, failed job log | no | yes | yes |
| Review threads beside the code | no | yes | yes |
| Conflict forecast against open MRs/PRs | no | yes | yes |
| Hover cards with title and state | local facts only | yes | yes |
| Create MR/PR in the app | no | no, says which scope | yes |
| Warn before pushing to a protected branch | no | yes | yes |

"Off" means the host is never contacted for it. A read-only token is GitLab `read_api`, or GitHub fine-grained read access to pull requests, checks, contents and issues.

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

For a deployment that must not carry the code at all, build with the `nohost` tag:

```bash
wails3 build EXTRA_TAGS=nohost
```

That build has no host API client, no `HostService`, no Integrations settings, no first-use prompt and no host indicator; Diagnostics says "Host integration: not built in". A test (`TestNohostBuildLeavesOutHostIntegration`) checks that the tagged build does not depend on `core/hostapi` or `core/hostcontent`, and the test suite runs under the tag too.

## Diagnostics

Settings → Diagnostics → **Copy diagnostics** reports which hosts integration is on for and the last errors shown. Tokens (by shape, and anything after `token=`, `Bearer` or `PRIVATE-TOKEN:`) and host answers (JSON objects, HTML pages) are replaced before the report is made, however they got into a message, and **Leave out hostnames** replaces every known host and API host with `<host>`. Repository paths and the home folder were already left out.
