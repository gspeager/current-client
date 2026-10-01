// Generates THIRD_PARTY_NOTICES from the Go modules linked into the app (for
// every target OS) and the npm packages bundled into the frontend.
//   node scripts/third-party-notices.mjs          write THIRD_PARTY_NOTICES
//   node scripts/third-party-notices.mjs --check  fail on a license that can't ship under Apache-2.0
import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { basename, join } from 'node:path'

const root = join(import.meta.dirname, '..')
const goLicensesVersion = 'v1.6.0'
const targets = ['windows', 'darwin', 'linux']
// Copyleft or unknown terms that can't be bundled into an Apache-2.0 binary.
const disallowed = /GPL|SSPL|EUPL|CC-BY-NC|Unknown|UNKNOWN|UNLICENSED/

function run(cmd, args, options = {}) {
  return execFileSync(cmd, args, { cwd: root, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, ...options })
}

// Built for the host once, then pointed at each target with GOOS: `go run`
// would build the tool itself for the target and fail to start it.
function goLicensesBinary() {
  const bin = join(tmpdir(), `go-licenses-${goLicensesVersion}`)
  const exe = join(bin, process.platform === 'win32' ? 'go-licenses.exe' : 'go-licenses')
  if (!existsSync(exe)) {
    mkdirSync(bin, { recursive: true })
    run('go', ['install', `github.com/google/go-licenses@${goLicensesVersion}`], { env: { ...process.env, GOBIN: bin } })
  }
  return exe
}

function goModules() {
  const self = /^module\s+(\S+)/m.exec(readFileSync(join(root, 'go.mod'), 'utf8'))[1]
  const template = join(tmpdir(), 'go-licenses-notices.tpl')
  writeFileSync(template, '{{range .}}{{.Name}}\t{{.LicenseName}}\t{{.LicensePath}}\n{{end}}')
  const exe = goLicensesBinary()
  const byName = new Map()
  for (const goos of targets) {
    const report = run(exe, ['report', '.', '--template', template], {
      env: { ...process.env, GOOS: goos },
      stdio: ['ignore', 'pipe', 'ignore'],
    })
    for (const line of report.split('\n').filter(Boolean)) {
      const [name, license, licensePath] = line.split('\t')
      if (name === self) continue
      const entry = byName.get(name) ?? { name, license, licensePath, targets: [] }
      entry.targets.push(goos)
      byName.set(name, entry)
    }
  }
  // go-licenses leaves out the standard library, which is compiled in too.
  const goroot = run('go', ['env', 'GOROOT']).trim()
  byName.set('Go standard library', {
    name: 'Go standard library',
    license: 'BSD-3-Clause',
    licensePath: join(goroot, 'LICENSE'),
    targets,
  })
  return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name))
}

function npmPackages() {
  const frontend = join(root, 'frontend')
  const checker = join(frontend, 'node_modules', 'license-checker-rseidelsohn', 'bin', 'license-checker-rseidelsohn.js')
  const json = run(process.execPath, [checker, '--production', '--excludePrivatePackages', '--json'], { cwd: frontend })
  return Object.entries(JSON.parse(json))
    .map(([id, info]) => ({
      name: id,
      license: Array.isArray(info.licenses) ? info.licenses.join(' OR ') : info.licenses,
      // The checker falls back to a README when a package has no license file.
      licensePath: info.licenseFile && /licen[cs]e|copying/i.test(basename(info.licenseFile)) ? info.licenseFile : null,
      repository: info.repository,
    }))
    .sort((a, b) => a.name.localeCompare(b.name))
}

// Files a package ships under a license other than its own, with no license
// file of their own. The OFL text is taken from a package that ships it.
function bundledAssets() {
  const ofl = readFileSync(join(root, 'frontend', 'node_modules', '@fontsource', 'jetbrains-mono', 'LICENSE'), 'utf8')
  const oflBody = ofl.replace(/\r\n/g, '\n').split('\n').slice(1).join('\n')
  return [
    {
      name: 'Roboto 3.014 (fonts inside pdfmake, used for PDF export)',
      license: 'OFL-1.1',
      licenseText: 'Copyright 2011 The Roboto Project Authors (https://github.com/googlefonts/roboto-classic)\n' + oflBody,
    },
  ]
}

function section(title, entries) {
  const lines = [`${title}\n${'='.repeat(title.length)}\n`]
  for (const e of entries) {
    const scope = e.targets && e.targets.length < targets.length ? ` (${e.targets.join(', ')} builds)` : ''
    lines.push('-'.repeat(78), `${e.name}${scope}`, `License: ${e.license}`, '')
    lines.push(
      e.licenseText?.trim() ??
        (e.licensePath
          ? readFileSync(e.licensePath, 'utf8').replace(/\r\n/g, '\n').trim()
          : `License text not included in the package; see ${e.repository ?? 'the package page'}.`),
    )
    lines.push('')
  }
  return lines.join('\n')
}

const go = goModules()
const npm = npmPackages()
const rejected = [...go, ...npm].filter((e) => disallowed.test(e.license))

if (process.argv.includes('--check')) {
  if (rejected.length > 0) {
    console.error('Licenses that cannot ship under Apache-2.0:')
    for (const e of rejected) console.error(`  ${e.name}: ${e.license}`)
    process.exit(1)
  }
  console.log(`OK: ${go.length} Go and ${npm.length} npm dependencies checked.`)
} else {
  const header =
    'Third-party notices for Current Client\n\n' +
    'Current Client includes the following third-party software. Each is listed with its\n' +
    'license and the license text shipped with it. Generated by\n' +
    'scripts/third-party-notices.mjs; do not edit by hand.\n'
  writeFileSync(
    join(root, 'THIRD_PARTY_NOTICES'),
    [
      header,
      section('Go modules', go),
      section('Frontend packages', npm),
      section('Bundled assets', bundledAssets()),
    ].join('\n'),
  )
  console.log(`Wrote THIRD_PARTY_NOTICES: ${go.length} Go and ${npm.length} npm dependencies.`)
}
