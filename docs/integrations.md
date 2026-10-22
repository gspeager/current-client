# Host integration

Current Client can show what a Git host knows about a repository: merge requests / pull requests, CI results and review threads, from GitLab (gitlab.com or Self-Managed) and GitHub (github.com or Enterprise Server). This guide says exactly when it talks to a host, what it asks for, what it keeps and how to turn it off or keep it off. It is being written as the feature is built for 0.4.0; the sections marked *to come* are filled in as each part lands.

The rules it works under are the [privacy rules](../CONTRIBUTING.md#privacy-rules): the only destination is the host the repository's remotes already point at, and nothing is contacted until someone turns integration on for that host.

## Off until turned on

Integration is per host and off by default. Turning it on is a deliberate choice in Settings → Integrations, or a first-use prompt that names the host and says what would be contacted. Until then no host UI appears and nothing is sent. Turning a host off clears everything cached from it and deletes the refs fetched for it (below).

The set of enabled hosts is decided by the app, not by `core/`. `core/` cannot read the settings file; it is handed the set by whichever app embeds it (Current Client passes what Settings says, other apps pass their own), and it refuses any host outside that set before a connection is opened. That is what makes "integration off means no request" a property of the library, not of each caller.

## When requests happen

A **refresh** is the only thing that fetches from a host:

- The refresh button in a host panel.
- Opening a host panel for the first time in a session.
- A Git fetch or pull the user starts.
- A push refreshes only the pushed branch's MR/PR and its CI.

Nothing else does. Switching branches, scrolling history, drawing the sidebar or graph, hovering a marker or opening a diff reads what is cached and never makes a request. With background refresh on (off by default, like background fetch) the same refresh runs on a timer.

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

*To come.* Git's own credential helper is asked first; a pasted token is stored by Git's helper under a separate key, never by Current Client.

## Certificates and proxies

*To come.* Host API traffic trusts and routes exactly as Git does for the same URL, read from `git config`.

## Building without host integration

*To come.* A build tag that compiles host integration out entirely.
