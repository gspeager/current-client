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

**Add the changelog entry** to `CHANGELOG.md`: a `## [x.y.z] - YYYY-MM-DD` section in [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) form, written for users, plus the `[x.y.z]:` link at the bottom. Pull requests don't touch `CHANGELOG.md`, so they never conflict over it; the whole section is written here. Draft it from the Conventional Commit titles since the last tag with the Changelog tab, then rewrite it for someone upgrading: take the wording from the squash commits' bodies, leave out fixes to features new in this release, and fold them into those features' entries.

**Check the library API:** `wails3 task api:check` compares `core/` against the newest tag.

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

- the wiki, for anything the release changes for users
- the website's Current Client page, `current-client.html`: the version beside the name, on both **Download** buttons and in the footer; the release date under the hero; and **What's new**, which lists the release's main changes and links to the changelog
