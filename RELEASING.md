# Releasing

How a Current Client release is versioned, built, tagged and published. This is for maintainers; contributors only need [CONTRIBUTING.md](CONTRIBUTING.md).

macOS and Windows packages are built on a Mac. Linux packages are built on an x86_64 Linux machine in Docker. Every package is built from a checkout of the release tag.

## 1. Prepare the release on `develop`

Branch `release/x.y.z` from `develop`, then:

**Bump the version** in every file the build carries it in. `wails3 task common:update:build-assets` can regenerate these from `build/config.yml`, but it also overwrites hand-made fixes, so bump them by hand:

| File | What it sets |
|---|---|
| `build/config.yml` | `info.version`: the version embedded in the binary, shown in the status bar and Diagnostics |
| `build/darwin/Info.plist`, `build/darwin/Info.dev.plist` | `CFBundleShortVersionString`, `CFBundleVersion` |
| `build/windows/info.json` | `.exe` file and product version |
| `build/windows/wails.exe.manifest` | assembly version |
| `build/windows/nsis/wails_tools.nsh` | `INFO_PRODUCTVERSION`, the installer's version |
| `build/windows/msix/app_manifest.xml`, `build/windows/msix/template.xml` | MSIX version, as `x.y.z.0` |
| `build/linux/nfpm/nfpm.yaml` | `.deb`/`.rpm` package version |

Afterwards, `grep -rn "<old version>" build` should find nothing.

**Finish the changelog entry** in `CHANGELOG.md`: pull requests add their user-facing changes under `## [Unreleased]` as they merge, so rename that heading to `## [x.y.z] - YYYY-MM-DD` in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) form and add the `[x.y.z]:` link at the bottom. Read it as someone upgrading would: fold fixes to features new in this release into those features' entries, and check the Conventional Commit titles since the last tag (the Changelog tab lists them) for anything a pull request left out.

**Check the library API:** `wails3 task api:check` compares `core/` against the newest tag.

**Check readiness in the app:** Tags → **Release readiness**, with the candidate `HEAD` of the release branch, the version, and target `main`. Under **Version files**, enter the files in the table above, one per line, once per clone. It reports any file that lacks the new version or still has the old one, whether `CHANGELOG.md` has the version's section yet, the commits since the last tag, and, with host integration on, CI on the candidate and protection on `main` from the last refresh. It doesn't run the API check.

Open the pull request into `develop` as `chore: release x.y.z` and squash-merge it.

## 2. Merge to `main` and tag

1. Open a pull request from `develop` into `main`, and merge it with **Create a merge commit**, not a squash. A squash gives `main` a commit `develop` doesn't have, and the next release's merge then conflicts.
2. Tag the merge commit on `main`:
   ```
   git fetch origin
   git switch main && git merge --ff-only origin/main
   git tag -a vX.Y.Z -m "Current Client X.Y.Z"
   git push origin vX.Y.Z
   ```
3. Fast-forward `develop` to `main`, so the tag is in `develop`'s history. The Changelog tab finds the latest tag with `git describe`, which only sees tags on the current branch.
   ```
   git switch develop && git merge --ff-only main
   git push origin develop
   ```

If the release date has moved since the changelog entry was written, fix it on `develop` before opening the pull request into `main`.

## 3. Build the packages

Build from the tag (`git switch --detach vX.Y.Z`). Run `git describe --tags` first: it must print exactly `vX.Y.Z`.

Start with an empty `bin/`. A universal macOS build leaves a fat binary at `bin/current-client`, and a later `wails3 dev` or single-architecture build fails on it with "already exists and is not an object file".

The version shown in the app comes from `build/config.yml`. `VERSION=x.y.z` on a task's command line overrides it, but the package metadata files above don't follow that override.

### macOS (universal, signed and notarized)

Signing needs a Developer ID certificate and a notarytool keychain profile, set up once with `wails3 setup signing`.

```
wails3 task darwin:package:universal
wails3 tool sign --input "bin/Current Client.app" --notarize
rm -f "bin/Current Client.app.zip"
wails3 task darwin:create:dmg
```

`darwin:sign:notarize` isn't used, because it depends on the single-architecture `package` task and would replace the universal build. The DMG is created after notarizing, so it holds the stapled app.

Check: `spctl -a -vv -t exec "bin/Current Client.app"` reports `source=Notarized Developer ID`.

The bundle is `bin/Current Client.app` (the name Finder shows), and the DMG is `bin/current-client.dmg`.

### Windows (cross-built on the Mac)

The NSIS installer needs `makensis` (`brew install makensis`). Each build writes `bin/current-client.exe`, so rename it before building the next architecture:

```
for arch in amd64 arm64; do
  wails3 task windows:package ARCH=$arch
  mv bin/current-client.exe bin/current-client-$arch.exe
done
```

This gives `bin/current-client-amd64-installer.exe` and `bin/current-client-arm64-installer.exe`. The installers aren't code-signed, so Windows SmartScreen warns on first run.

### Linux (x86_64 hardware, in Docker)

Building the AppImage under emulation fails (QEMU rejects the AppImage tooling's ELF header with "exec format error"), so build on real x86_64 Linux. `build/docker/Dockerfile.linux-native` has the toolchain. Run it from a clone of this repository with the tag checked out:

```
git fetch --tags origin && git switch --detach vX.Y.Z
docker build -t current-client-linux-native -f build/docker/Dockerfile.linux-native .
docker run --rm --user "$(id -u):$(id -g)" -v "$(pwd):/app" current-client-linux-native
```

This gives `bin/current-client-x86_64.AppImage`, `bin/current-client.deb` and `bin/current-client.rpm`. Check the package version with `dpkg-deb -f bin/current-client.deb Version`.

## 3b. Check that nothing leaves the network

Run this on each platform's built package, before upload, so the [privacy model](README.md#privacy) is shown rather than assumed: with only local remotes, nothing Current Client does leaves the network, and with host integration on for one local host, the only API traffic goes to that host. Record the result in the table at the end; the release is held until all three platforms pass.

**A local server.** A bare repository served by `git daemon` is enough for the integration-off case:

```
git init --bare /tmp/remote/app.git
git daemon --base-path=/tmp/remote --export-all --enable=receive-pack --reuseaddr --listen=127.0.0.1 --port=9418
```

and the remote is `git://127.0.0.1/app.git`. The integration-on case needs a host with an API: a GitLab CE container, `docker run --rm -p 8929:8929 -p 2224:22 -e GITLAB_OMNIBUS_CONFIG="external_url 'http://127.0.0.1:8929'" gitlab/gitlab-ce` (several minutes to start; the root password is in `/etc/gitlab/initial_root_password` inside the container), a project pushed to it, one merge request with a description that embeds an image hosted elsewhere (`![](https://example.com/image.png)`), and a `read_api` token. Add `127.0.0.1:8929` in Settings → Integrations as a GitLab host and turn it on. Docker's own traffic is on the host side of the firewall and shows as `docker` or `com.docker`, not as the app.

**What to exercise**, in the packaged app and not `wails3 dev`:

1. Integration off, every remote local: open the repository, fetch, pull, commit, push, browse History and diffs, open Settings and Diagnostics, leave the app idle for ten minutes with background fetch off and then ten with it on. Type a misspelled word in the commit message field, since the embedded browser may try to fetch a spelling dictionary.
2. Integration on for the local host: open the Merge requests tab, **Refresh**, hover a marker, open the merge request whose description embeds the remote image, check out a merge request, open Settings → Integrations and **Test connection**.

Pass only if the log shows no connection to anything but the local server's address (the Git port, and the API port in case 2), and in case 2 nothing to the image's host.

**Connections the operating system makes for itself** are not the app's, but they appear in the same log, so recognise them rather than fail on them: macOS Gatekeeper and notarization checks (`api.apple-cloudkit.com`, `ocsp.apple.com`, `*.apple.com` from `trustd`, `syspolicyd`, `nsurlsessiond`), Windows certificate revocation and update traffic (`ctldl.windowsupdate.com`, `*.microsoft.com` from `svchost`), and anything from `docker`. A connection from the app's own process to any of those is a failure.

### macOS

`pf` logs what it blocks. Allow loopback and the local server, block and log everything else from the app's process by its user is not possible, so block all outbound and read the process name from the log:

```
cat > /tmp/current-client.pf <<'RULES'
set skip on lo0
block out log all
pass out quick proto tcp to 127.0.0.1 port { 9418, 8929, 2224 }
RULES
sudo pfctl -F all -f /tmp/current-client.pf
sudo pfctl -e
sudo tcpdump -n -e -ttt -i pflog0 'action block'
```

Open the app from `bin/Current Client.app`, work through the list, then `sudo pfctl -d` and `sudo pfctl -F all` to restore. Every blocked packet prints with its destination; `sudo lsof -i -nP | grep -i "current client"` while the app runs shows what it has open and confirms the process names in the log. Little Snitch's connection list for the app, with its rules set to ask, is an equivalent check.

### Windows

The firewall log records dropped connections with the process path. In an administrator PowerShell:

```
netsh advfirewall set allprofiles state on
netsh advfirewall set allprofiles firewallpolicy blockinbound,blockoutbound
netsh advfirewall set allprofiles logging droppedconnections enable
netsh advfirewall set allprofiles logging filename %SystemRoot%\system32\LogFiles\Firewall\pfirewall.log
netsh advfirewall firewall add rule name="Current Client check: local git" dir=out action=allow protocol=TCP remoteip=127.0.0.1 remoteport=9418,8929,2224
```

Install from `current-client-X.Y.Z-windows-x64-setup.exe`, work through the list, then read `C:\Windows\System32\LogFiles\Firewall\pfirewall.log`: every `DROP` line names the destination and, in the last column, the process. Restore with `netsh advfirewall set allprofiles firewallpolicy blockinbound,allowoutbound` and delete the rule. On the WebView2 side, watch for `edge.microsoft.com`, `msedge.b.tlu.dl.delivery.mp.microsoft.com` and `*.smartscreen.microsoft.com` from `msedgewebview2.exe`: the app turns SmartScreen off, so a hit there is a regression.

### Linux

A network namespace holds the app and the local server together with nothing else, and nftables logs what the app tries:

```
sudo unshare --net --mount-proc bash -c '
  ip link set lo up
  nft add table inet guard
  nft add chain inet guard out "{ type filter hook output priority 0; policy accept; }"
  nft add rule inet guard out ip daddr != 127.0.0.0/8 log prefix \"current-client: \" drop
  nft add rule inet guard out ip6 daddr != ::1 log prefix \"current-client: \" drop
  git daemon --base-path=/tmp/remote --export-all --enable=receive-pack --listen=127.0.0.1 --port=9418 &
  sudo -u "$SUDO_USER" ./current-client-X.Y.Z-linux-x86_64.AppImage
'
sudo journalctl -k -f | grep current-client:
```

The namespace has no route out, so any attempt is logged and fails at once; the app must show no error for it either. For case 2 run the GitLab container's port forward inside the same namespace, or point the app at a GitLab on the machine's own interface and allow that address in the rule.

### Results

Paste this into the release pull request, filled in:

| Platform | Package | Integration off | Integration on | Notes |
|---|---|---|---|---|
| macOS | `current-client-X.Y.Z-macos-universal.dmg` | pass / fail | pass / fail | OS traffic seen, if any |
| Windows x64 | `current-client-X.Y.Z-windows-x64-setup.exe` | | | |
| Linux | `current-client-X.Y.Z-linux-x86_64.AppImage` | | | |

## 4. Publish

Copy the packages into one folder under their release names:

| Build output | Release asset |
|---|---|
| `current-client.dmg` | `current-client-X.Y.Z-macos-universal.dmg` |
| `current-client-amd64-installer.exe` | `current-client-X.Y.Z-windows-x64-setup.exe` |
| `current-client-arm64-installer.exe` | `current-client-X.Y.Z-windows-arm64-setup.exe` |
| `current-client-x86_64.AppImage` | `current-client-X.Y.Z-linux-x86_64.AppImage` |
| `current-client.deb` | `current-client_X.Y.Z_amd64.deb` |
| `current-client.rpm` | `current-client-X.Y.Z.x86_64.rpm` |

Write `SHA256SUMS` over the renamed files, so it matches what users download, and check it:

```
shasum -a 256 current-client-X.Y.Z-linux-x86_64.AppImage current-client-X.Y.Z-macos-universal.dmg \
  current-client-X.Y.Z-windows-arm64-setup.exe current-client-X.Y.Z-windows-x64-setup.exe \
  current-client_X.Y.Z_amd64.deb current-client-X.Y.Z.x86_64.rpm > SHA256SUMS
shasum -a 256 -c SHA256SUMS
```

(`wails3 task release:checksums` hashes the unrenamed files in `bin/`, so its output doesn't match the release asset names.)

Create the GitHub release for the tag as a **draft**, paste the version's `CHANGELOG.md` section as the notes, attach the six packages and `SHA256SUMS`, and publish once everything is attached. Don't attach the bare binaries (`current-client`, `current-client-*.exe`).

Finally, update what users read about the release:

- the wiki: merge its version branch (`0.4.0`, kept current as PRs merged) into `master`, which is what GitHub renders, and check it against the changelog for anything missed
- the website's Current Client page, `current-client.html`: the version beside the name, on both **Download** buttons and in the footer; the release date under the hero; and **What's new**, which lists the release's main changes and links to the changelog
