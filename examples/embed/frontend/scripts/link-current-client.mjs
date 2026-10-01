// Links ./current-client to Current Client's frontend/src at the version the Go module pins, and
// checks this app's npm dependencies against Current Client's. Runs before dev, build and
// test (see package.json), so the frontend and Go sides never drift apart.
import { execFileSync } from 'node:child_process'
import { lstatSync, readFileSync, readlinkSync, symlinkSync, unlinkSync } from 'node:fs'
import { join, resolve } from 'node:path'

const frontend = resolve(import.meta.dirname, '..')
const currentClientDir = execFileSync('go', ['list', '-m', '-f', '{{.Dir}}', 'github.com/gspeager/current-client'], {
  cwd: resolve(frontend, '..'),
  encoding: 'utf8',
}).trim()
const target = join(currentClientDir, 'frontend', 'src')
const link = join(frontend, 'current-client')

let current = null
try {
  if (lstatSync(link).isSymbolicLink()) current = readlinkSync(link)
} catch {
  // No link yet.
}
if (current === null || resolve(current) !== resolve(target)) {
  if (current !== null) unlinkSync(link)
  symlinkSync(target, link, process.platform === 'win32' ? 'junction' : 'dir')
}

// Current Client's source compiles with this app's packages, so their versions must match.
const readPackage = (dir) => JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'))
const currentClient = readPackage(join(currentClientDir, 'frontend'))
const host = readPackage(frontend)
const required = { ...currentClient.dependencies, sass: currentClient.devDependencies.sass }
const installed = { ...host.dependencies, ...host.devDependencies }
const mismatches = Object.entries(required).filter(([name, version]) => installed[name] !== version)
if (mismatches.length > 0) {
  console.error(`package.json must match Current Client's (${currentClientDir}):`)
  for (const [name, version] of mismatches) {
    console.error(`  ${name}: ${version} (have ${installed[name] ?? 'nothing'})`)
  }
  process.exit(1)
}
