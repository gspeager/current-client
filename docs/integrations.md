# Host integration

Current Client can show what a Git host knows about a repository: merge requests / pull requests, CI results and review threads, from GitLab (gitlab.com or Self-Managed) and GitHub (github.com or Enterprise Server). This guide says exactly when it talks to a host, what it asks for, what it keeps and how to turn it off or keep it off. It covers 0.4.0 in full: every endpoint, its trigger and its scope, the refresh and cache model, what works at each level of access, credentials, certificates and proxies, the managed policy, the build without the code, and what Diagnostics may contain.

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

Nothing else does, with bounded exceptions that each follow a click: CI results for commits that come into view in History are asked for as you scroll, in batches, each commit once until the next refresh; the merge request tab's **Closed** filter asks for one page of closed changes the first time it is chosen; and a change's **Checks**, **Threads**, **Files** and **Commits** tabs ask for what the refresh didn't bring, each once per change until the next refresh. Switching branches, drawing the sidebar or graph, hovering a marker or opening a diff reads what is cached and never makes a request. There is no separate background refresh: a fetch is a refresh, so the background fetch setting (off by default) covers the host too.

With integration off, the only host-related traffic is Git's own: a fetch the user starts also runs one `git ls-remote` against the remote, which lists the pull request / merge request heads the host publishes (`refs/pull/*` on GitHub, Gitea and Forgejo, `refs/merge-requests/*` on GitLab) and gives each branch its open change's number. Bitbucket publishes none, so it gives nothing. A change counts only while its head is still a branch tip, which drops every change whose branch was deleted on merge. GitHub also publishes `refs/pull/N/merge` only while a pull request is open, so there the number is an open one. GitLab keeps merge refs after closing, so Git alone cannot tell a closed merge request whose branch was kept from an open one: with integration off the branch shows the number without a state, and the state comes from the API once integration is on.

## What is cached

One cache per repository, keyed by host and repository, holding MRs/PRs with their reviews, CI results, review threads and issue titles with the time each was fetched. It lives in memory only; nothing from a host is written to disk.

- Data stays visible after it goes stale, labeled with its age, until a refresh replaces it. Offline, the last answer shows.
- The cache is cleared when integration is turned off for the host or the repository is closed.

## Open on host

With or without integration on, anything in the app can be opened on the host, built from the remote's address alone: a commit (its context menu) opens the commit page; a file in a commit or in Changes opens the file at that commit, or at HEAD; a branch opens its merge request / pull request when one is known, else its tree; a tag opens the tag or release page; **Open repository on host** in the command palette opens the repository's home page. The actions are hidden when no remote points at a hosted project, and none of them makes a request.

## Create a merge request / pull request, without the API

After a push that sets or updates a branch's upstream, and while the branch has no known merge request / pull request, a line above the views offers **Create merge request** (or pull request): the link the host printed in the push output when it did, else the host's pre-filled form built from the remote's address (GitLab's new-merge-request form with the source branch, GitHub's compare page with the default branch as base). The same action is in the branch's context menu and in the sidebar section for the checked-out branch. For a fork, a remote named `upstream` on the same host, GitHub's form lives in the upstream project with the fork's owner on the head; GitLab's form is the fork's own, where the target project is chosen. No request is made to show the link.

## Create a merge request / pull request in the app

With the host on, **Create merge request** opens a form in the app instead of the host's page: the target branch (the default branch, changeable), a title from the branch's commits (one commit: its subject; several: the oldest's), a description from the commit list placed inside the repository's template when it has one (`.gitlab/merge_request_templates/Default.md` or the first template, `.github/pull_request_template.md` and GitHub's other places), and a draft toggle. A branch that isn't pushed yet is pushed first, with Git's credentials. This is the one request that needs write scope (GitLab `api`; GitHub write access to pull requests): a token without it gets a message naming the scope, and the host's own form stays one click away. The created change shows on the branch at once. For a change from a fork, the host's form is used.

## Clean up merged branches

**Clean up merged branches**, in the Branches sidebar's header, lists the local branches whose work is already on the default branch and offers to delete them. Git alone finds them, so this works with integration off: a branch with an upstream is merged when merging it into the default branch's upstream (`origin/main`, or the local branch without a remote) would change nothing, which catches squash and rebase merges as well as merge commits. With the host on, a branch whose merge request / pull request the cache knows as merged is listed with its number as the reason; the host is asked about a change once, on the refresh after it leaves the open list or while Git still publishes its head, and never again. A branch with commits past what was merged is marked **Newer commits** and left unselected. Building the list makes no request. Deleting goes through the same **Undo** as a single deletion; when the checked-out branch is among them, the default branch is checked out and pulled first.

## Protected branches

With the host on, a push, force-push, push that sets an upstream, or deletion on the remote of a branch the host protects is confirmed first, naming the branch, the rule when it is a wildcard (`release/*`, matched in the app) and what the host forbids: direct pushes, pushes except by some roles, force-pushes, deleting it. The action is allowed after the warning; the host enforces. The rules come from a refresh and the cache: nothing is asked at push time, and with no rules cached, or the host unreachable, the push runs as it does without the host. On GitLab the endpoint needs the Maintainer role, and a 403 means no warning rather than an error.

## The merge request / pull request tab

While a host is on for the repository, a tab named **Merge requests** (GitLab) or **Pull requests** (GitHub) lists the open changes from the last refresh as cards: number, draft, CI result, where its reviews stand (changes requested, approvals, or review requested), title, source and target branch and when it was updated. The header names the host and the time of the last refresh, filters the list to **Open**, **Mine**, **To review** (open changes that ask for your review) or **Closed**, and holds **Refresh** and **New merge request** (the in-app form, from the checked-out branch). Opening the tab for the first time in a session refreshes it; after that it reads the cache until **Refresh**, a fetch or a pull. **Closed** asks the host for one page of merged and closed changes the first time it is chosen, then reads the cache. The list stops at the page cap and says so, and a line at its foot says when the host is contacted and links to the integration settings.

Selecting a change shows its author, branches, each reviewer by state, checks, conflict state and labels, **Check out locally** and **Open on <host>**, then five tabs: **Overview** (the description, rendered through the one sanitizing renderer: images are links, nothing is fetched), **Checks** (the head pipeline's jobs), **Threads** (its review threads), **Files** and **Commits**. Each of the last four asks the host once per change, the first time it is opened, until the next refresh. Checking out fetches the change's head from the remote into `refs/current-client/<remote>/mr/<number>` (ordinary Git traffic, with Git's own credentials) and creates a local branch at it named after the source branch, or `<owner>/<branch>` for a change from a fork; a branch that already exists is checked out as it is. The diff, history and blame views then work on it as on any branch.

## The checked-out branch's merge request

While the host is on and the checked-out branch's upstream has a merge request / pull request in the cache, the top of the Branches sidebar shows it: number, title, state, draft, target branch, the CI result with the failed jobs named (a refresh fetches the jobs of a failing pipeline for this one branch, so no click is needed), unresolved threads, **Details** (the merge request tab) and **Open on host**. It reads the cache; checking out a branch makes no request. Review threads, the conflict forecast and the clean-up offer join it as they land.

## What a refresh found

After a refresh (the tab's button, a fetch or a pull), Current Client compares the new answer for the checked-out branch's merge request / pull request with the last one and, when something changed, shows one line above the views: "2 new comments on !138", "Pipeline failed: test on !138", "!138 was merged". Each goes to the relevant place: the file with the new thread in Changes, or the merge request's details. An unchanged answer shows nothing, nothing is shown for other branches, and nothing is modal. The comparison uses only what the refresh already fetched.

## Conflict forecast

On every refresh, with the host on, Current Client takes the open merge requests / pull requests that target the same base as the checked-out branch (its own merge request's target, else the default branch), newest first and at most 20, fetches their heads into `refs/current-client/` in one `git fetch` (ordinary Git traffic to the remote, with Git's credentials) and merges each with the checked-out branch in memory with `git merge-tree`. The result appears at the top of the Branches sidebar: "Conflicts with !138 by Ada in app/x.go", with "20 of 34 checked" when the cap cut the list, or that none of the checked ones conflict. Opening a merge request in its tab shows the same answer from the other side. Nothing about local work is sent to the host; the only API call is the list of open changes the refresh already made. The merges are remembered per pair of commits, so a refresh whose heads haven't moved runs none.

## Review threads

For the checked-out branch's merge request / pull request, a refresh also fetches its review threads. In the Changes diff and in any file diff opened from the History or Compare views, each thread appears under the line it was left on, read-only: who wrote it, when, and the comment rendered through the one sanitizing renderer (images are links). Resolved threads start collapsed to one line. Threads on lines the diff doesn't show, and outdated ones whose code has changed since, are listed under the diff with their original line. File lists show each file's unresolved count, and the sidebar's count opens the first file that has one. Posting, resolving and approving stay on the host.

## CI results

With the host on, commits in History and branch tips in the sidebar show a small square in the pipeline's color: passed, failed, canceled, or an outline while running or pending. For commits the result is asked for when the commit scrolls into view, in batches of up to 50 at a time and never twice for the same commit until the next refresh; a commit without a pipeline is remembered as such. Branch tips and open merge requests get theirs on refresh. Clicking a square lists the pipeline's jobs with stage, status and duration, fetched only then, with **Open on host**. Clicking a failed job fetches the last 50 lines of its log, with terminal colors and escape sequences removed; **Show full log** fetches the whole log up to a cap and says when it was cut, and **Open on host** goes to the job. No log is fetched for a passing job or before a click. HEAD's cyan is never used for CI state.

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
| `GET /projects/:path/merge_requests/:iid` | Refresh of one merge request (the pushed branch's, the checked-out branch's); once per local branch whose merge request left the open list, to learn whether it was merged | `read_api` |
| `GET /projects/:id` | Once per fork a merge request comes from, to name it | `read_api` |
| `GET /projects/:path/merge_requests?state=merged` and `?state=closed` | The tab's **Closed** filter, once until the next refresh; one page of 25 each | `read_api` |
| `GET /projects/:path/merge_requests/:iid/reviewers` and `…/approvals` | Refresh, for each open merge request, at most 50; the approvals too, since anyone may approve | `read_api` |
| `GET /projects/:path/merge_requests/:iid/discussions` | Refresh, for the checked-out branch's merge request; a change's **Threads** tab, once per change until the next refresh | `read_api` |
| `GET /projects/:path/merge_requests/:iid/diffs` and `…/commits` | A change's **Files** or **Commits** tab, once per change until the next refresh | `read_api` |
| `GET /projects/:path/pipelines?sha=` | Refresh, for open merge requests' heads; as commits scroll into view, once per commit until the next refresh; a change's **Checks** tab when the cache has none for its head | `read_api` |
| `GET /projects/:path/pipelines/:id/jobs` | Clicking a CI mark or a change's **Checks** tab; on refresh, for the checked-out branch's merge request when its pipeline failed | `read_api` |
| `GET /projects/:path/jobs/:id/trace` | Clicking a failed job | `read_api` |
| `GET /projects/:path/issues/:iid` | When a History page with `#n` references loads, each number once until the next refresh, at most 50 at a time | `read_api` |
| `GET /projects/:path/protected_branches` | Refresh; a 403 (not a Maintainer) means no warnings | `read_api` |
| `POST /projects/:path/merge_requests` | Create merge request | `api` |

### GitHub (github.com and Enterprise Server 3.11 or newer)

Under `https://api.github.com` or the server's `/api/v3`. A classic token needs the `repo` scope; a fine-grained one, read access to pull requests, checks, contents and issues, and write access to pull requests for creating one. The two differ only in address; Enterprise Server before 3.11 has no repository rulesets, so it gets everything but the protected-branch warning.

| Endpoint | When | Access |
| --- | --- | --- |
| `GET /user` | Test connection | read |
| `GET /repos/:owner/:repo/pulls?state=open` | Refresh: the open pull requests, paged up to the cap | pull requests: read |
| `GET /repos/:owner/:repo/pulls/:number` | Refresh of one pull request; once per local branch whose pull request left the open list | pull requests: read |
| `GET /repos/:owner/:repo/pulls?state=closed` | The tab's **Closed** filter, once until the next refresh; one page of 50 | pull requests: read |
| `GET /repos/:owner/:repo/pulls/:number/reviews` | Refresh, for each open pull request, at most 50; requested reviewers come with the open list | pull requests: read |
| `GET /repos/:owner/:repo/pulls/:number/comments` | Refresh, for the checked-out branch's pull request; a change's **Threads** tab, once per change until the next refresh (review comments; the REST API doesn't say whether a thread is resolved, so none shows as resolved) | pull requests: read |
| `GET /repos/:owner/:repo/pulls/:number/files` and `…/commits` | A change's **Files** or **Commits** tab, once per change until the next refresh | pull requests: read |
| `GET /repos/:owner/:repo/commits/:sha/check-runs` and `…/status` | Refresh, for open pull requests' heads; as commits scroll into view, once per commit until the next refresh; clicking a CI mark or a change's **Checks** tab for the job list. Check runs and commit statuses are read as one result | checks, commit statuses: read |
| `GET /repos/:owner/:repo/actions/jobs/:id/logs` | Clicking a failed job; the host redirects to the log's storage, which is followed without the token | actions: read |
| `GET /repos/:owner/:repo/issues/:number` | When a History page with `#n` references loads, each number once until the next refresh, at most 50 at a time | issues: read |
| `GET /repos/:owner/:repo/rules/branches/:branch` | Refresh, once per local branch's upstream, at most 25; a 404 (no rulesets) or 403 means no warnings | metadata: read |
| `POST /repos/:owner/:repo/pulls` | Create pull request | pull requests: write |

Avatar URLs in these answers are never decoded.

## What works at each level

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
| Reviewers, approvals and the **To review** filter | no | yes | yes |
| Review threads beside the code | no | yes | yes |
| Conflict forecast against open MRs/PRs | no | yes | yes |
| Hover cards with title and state | local facts only | yes | yes |
| Create MR/PR in the app | no | no, says which scope | yes |
| Warn before pushing to a protected branch | no | yes | yes |
| Managed policy: lock a host off or force it on | applies | applies | applies |

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

## Managed policy

An organization can fix the networked switches for every user of a machine with one file, read once at start-up and never contacted for anything:

| Platform | Path |
| --- | --- |
| Windows | `%ProgramData%\Current Client\policy.json` |
| macOS | `/Library/Application Support/Current Client/policy.json` |
| Linux | `/etc/current-client/policy.json` |

```json
{
  "hosts": {
    "gitlab.example.com": { "enabled": true },
    "github.com": { "enabled": false }
  },
  "autoFetch": { "enabled": true, "intervalMinutes": 15 }
}
```

A host with `enabled: false` is locked off: its switch in Settings is disabled and labeled **Managed by your organization**, the prompt to turn it on never shows, and the HTTP client refuses it whatever the user's settings say. With `enabled: true` it is forced on, with the same label. A host the file doesn't name is the user's to decide, as is everything with no file at all. `autoFetch` works the same for background fetch; forced on without an interval fetches every five minutes. A file that can't be read or parsed is ignored, as if absent. The Windows registry and macOS configuration profiles are for a later release.

## Building without host integration

For a deployment that must not carry the code at all, build with the `nohost` tag:

```bash
wails3 build EXTRA_TAGS=nohost
```

That build has no host API client, no `HostService`, no Integrations settings, no first-use prompt and no host indicator; Diagnostics says "Host integration: not built in". A test (`TestNohostBuildLeavesOutHostIntegration`) checks that the tagged build does not depend on `core/hostapi` or `core/hostcontent`, and the test suite runs under the tag too.

## Diagnostics

Settings → Diagnostics → **Copy diagnostics** reports which hosts integration is on for and the last errors shown. Tokens (by shape, and anything after `token=`, `Bearer` or `PRIVATE-TOKEN:`) and host answers (JSON objects, HTML pages) are replaced before the report is made, however they got into a message, and **Leave out hostnames** replaces every known host and API host with `<host>`. Repository paths and the home folder were already left out.
