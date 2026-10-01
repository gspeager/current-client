// Fails when an exported API in core/ changed incompatibly since a tag, so a
// break Current would hit shows up here first.
//   node scripts/api-check.mjs [tag]   defaults to the newest tag reachable from HEAD
import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const root = join(import.meta.dirname, '..')
const apidiffVersion = 'v0.0.0-20260908205506-85c1c2202aba'

function run(cmd, args, options = {}) {
  return execFileSync(cmd, args, { cwd: root, encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, ...options })
}

function apidiffBinary() {
  const bin = join(tmpdir(), `apidiff-${apidiffVersion}`)
  const exe = join(bin, process.platform === 'win32' ? 'apidiff.exe' : 'apidiff')
  if (!existsSync(exe)) {
    mkdirSync(bin, { recursive: true })
    run('go', ['install', `golang.org/x/exp/cmd/apidiff@${apidiffVersion}`], { env: { ...process.env, GOBIN: bin } })
  }
  return exe
}

const base = process.argv[2] ?? run('git', ['describe', '--tags', '--abbrev=0']).trim()
const apidiff = apidiffBinary()
const work = mkdtempSync(join(tmpdir(), 'current-client-api-'))
const baseTree = join(work, 'base')
run('git', ['worktree', 'add', '--detach', baseTree, base], { stdio: 'ignore' })

function compare(modulePath) {
  const broken = []
  // Internal packages can't be imported from outside core/, so they may change freely.
  const packages = run('go', ['list', './core/...'])
    .split('\n')
    .filter((pkg) => pkg && !pkg.includes('/internal/'))
  for (const pkg of packages) {
    // A package added since the tag has nothing to break.
    if (!existsSync(join(baseTree, pkg.slice(modulePath.length + 1)))) continue
    const exportFile = join(work, pkg.replaceAll('/', '_'))
    run(apidiff, ['-w', exportFile, pkg], { cwd: baseTree })
    const report = run(apidiff, ['-incompatible', exportFile, pkg]).trim()
    if (report) broken.push(`${pkg}\n${report}`)
  }
  if (broken.length) {
    console.error(`Incompatible core/ API changes since ${base}:\n\n${broken.join('\n\n')}`)
    process.exitCode = 1
  } else {
    console.log(`core/ API is compatible with ${base}.`)
  }
}

try {
  const modulePath = run('go', ['list', '-m']).trim()
  const baseModulePath = run('go', ['list', '-m'], { cwd: baseTree }).trim()
  if (baseModulePath === modulePath) {
    compare(modulePath)
  } else {
    console.log(`Module path changed since ${base} (was ${baseModulePath}); nothing to compare.`)
  }
} finally {
  run('git', ['worktree', 'remove', '--force', baseTree], { stdio: 'ignore' })
  rmSync(work, { recursive: true, force: true })
}
