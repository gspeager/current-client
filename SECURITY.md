# Security policy

## Reporting a vulnerability

Please report security problems privately, not in a public issue: on GitHub, open the repository's **Security** tab and choose **Report a vulnerability**.

Include what you found, how to reproduce it, and the version (Settings → Diagnostics → **Copy diagnostics** gives the details). You'll get an acknowledgement within a week. Please keep the issue private until a fix is released.

## Supported versions

Current Client is pre-1.0. Only the latest release receives security fixes.

## Scope

Current Client runs locally and only connects to destinations you configured: your repositories' Git remotes and the servers hosting them. Especially relevant reports include:

- Any network connection to a destination the user didn't configure, or networked behavior that runs without being turned on.
- Credentials, tokens or keys being logged, stored or exposed.
- Command injection through repository content, branch names, file names or remote URLs.
- Opening a repository causing code to run without the user's action.
