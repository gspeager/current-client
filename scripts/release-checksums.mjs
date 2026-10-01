// Writes bin/SHA256SUMS over whatever release artifacts are present in bin/,
// so a download can be verified with `shasum -a 256 -c` (macOS) or
// `sha256sum -c` (Linux). Each platform builds its own subset of these files,
// so this only hashes the ones it finds, not a fixed list that must all exist.
//   node scripts/release-checksums.mjs
import { createHash } from 'node:crypto'
import { createReadStream, existsSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const root = join(import.meta.dirname, '..')
const binDir = join(root, 'bin')

const artifacts = [
  'current-client.dmg',
  'current-client-amd64-installer.exe',
  'current-client-arm64-installer.exe',
  'current-client.deb',
  'current-client.rpm',
  'current-client-x86_64.AppImage',
]

function sha256(path) {
  return new Promise((resolve, reject) => {
    const hash = createHash('sha256')
    createReadStream(path)
      .on('data', (chunk) => hash.update(chunk))
      .on('end', () => resolve(hash.digest('hex')))
      .on('error', reject)
  })
}

const found = artifacts.filter((name) => existsSync(join(binDir, name)))
if (found.length === 0) {
  console.error(`No release artifacts found in ${binDir}`)
  process.exit(1)
}

const lines = []
for (const name of found) {
  const digest = await sha256(join(binDir, name))
  lines.push(`${digest}  ${name}`)
}

writeFileSync(join(binDir, 'SHA256SUMS'), lines.join('\n') + '\n')
console.log(`Wrote bin/SHA256SUMS for ${found.length} artifact(s):`)
for (const name of found) console.log(`  ${name}`)

const missing = artifacts.filter((name) => !found.includes(name))
if (missing.length > 0) {
  console.log(`Not present (built on a different machine): ${missing.join(', ')}`)
}
