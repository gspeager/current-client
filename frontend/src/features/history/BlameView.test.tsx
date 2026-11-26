import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BlameLineInfo, BlameService, IdentityService } from '@current-client-bindings/app'
import { OpenCommitContext } from '../../lib/openCommit'
import BlameView from './BlameView'

const REPO = '/repos/app'
const CHANGED = 'b'.repeat(40)
const ADDED = 'a'.repeat(40)

const line = (lineNo: number, content: string, sha: string, previous = '', previousPath = '') =>
  new BlameLineInfo({
    sha,
    lineNo,
    content,
    previous,
    previousPath,
    authorName: 'Ada',
    authorEmail: 'ada@example.com',
    summary: sha === ADDED ? 'Add the parser' : 'Reformat the parser',
    date: new Date().toISOString(),
  })

function show(onOpenCommit = vi.fn(), onClose = vi.fn()) {
  vi.mocked(IdentityService.GetCurrentUser).mockResolvedValue({ name: 'Me', email: 'me@example.com' } as never)
  vi.mocked(BlameService.GetBlame).mockResolvedValue([line(1, 'parse()', CHANGED, ADDED, 'old/parse.ts')])
  vi.mocked(BlameService.GetBlameAt).mockResolvedValue([line(1, 'parse( )', ADDED)])
  render(
    <OpenCommitContext.Provider value={onOpenCommit}>
      <BlameView repoPath={REPO} path="src/parse.ts" onClose={onClose} />
    </OpenCommitContext.Provider>,
  )
  return { onOpenCommit, onClose }
}

it('shows a focused line’s hash, and opens its commit in History, closing blame', async () => {
  const { onOpenCommit, onClose } = show()
  expect(screen.queryByText(CHANGED.slice(0, 7))).toBeNull()
  await userEvent.click(await screen.findByText('parse()'))
  expect(screen.getByText(CHANGED.slice(0, 7))).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Open in History' }))
  expect(onClose).toHaveBeenCalled()
  expect(onOpenCommit).toHaveBeenCalledWith(CHANGED)
})

it('copies the hash', async () => {
  const writeText = vi.fn().mockResolvedValue(undefined)
  Object.assign(navigator, { clipboard: { writeText } })
  show()
  await userEvent.click(await screen.findByText('parse()'))
  await userEvent.click(screen.getByRole('button', { name: 'Copy hash' }))
  expect(writeText).toHaveBeenCalledWith(CHANGED)
  expect(await screen.findByRole('button', { name: 'Copied' })).toBeInTheDocument()
})

it('blames the file before a line’s commit, back to the commit that added it', async () => {
  show()
  await userEvent.click(await screen.findByText('parse()'))
  await userEvent.click(screen.getByRole('button', { name: 'Blame before this commit' }))
  expect(BlameService.GetBlameAt).toHaveBeenCalledWith(REPO, 'old/parse.ts', ADDED)
  expect(await screen.findByRole('status')).toHaveTextContent(`As of ${ADDED.slice(0, 7)}`)
  expect(screen.getByRole('status')).toHaveTextContent('when it was old/parse.ts')

  await userEvent.click(await screen.findByText('parse( )'))
  expect(screen.getByText('The commit that added this file')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Blame before this commit' })).toBeNull()

  await userEvent.click(screen.getByRole('button', { name: 'Back' }))
  expect(await screen.findByText('parse()')).toBeInTheDocument()
  expect(screen.queryByRole('status')).toBeNull()
})
