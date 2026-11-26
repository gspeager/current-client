import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Report } from '@current-client-bindings/core/rework'
import Rework from './Rework'

const report = (over: Partial<Report> = {}) =>
  new Report({
    written: 120,
    rewritten: 30,
    commits: [
      { sha: 'a'.repeat(40), subject: 'write', written: 100, rewritten: 0 },
      { sha: 'b'.repeat(40), subject: 'rewrite', written: 20, rewritten: 30 },
    ],
    files: [
      { path: 'src/a.ts', written: 80, rewritten: 25, commits: 2 },
      { path: 'src/gone.ts', written: 5, rewritten: 5, commits: 2 },
    ],
    ...over,
  })

const onOpen = vi.fn()
const show = (r: Report | null, uncommitted = false) =>
  render(
    <Rework
      report={r}
      error={null}
      uncommitted={uncommitted}
      listed={(path) => path !== 'src/gone.ts'}
      onOpen={onOpen}
    />,
  )

it('lists files by lines rewritten and opens one', async () => {
  show(report())
  expect(screen.getByRole('list', { name: 'Lines rewritten by each of 2 commits' }).children).toHaveLength(2)
  const file = screen.getByRole('button', { name: 'src/a.ts: 25 of 80 lines written rewritten, across 2 commits' })
  await userEvent.click(file)
  expect(onOpen).toHaveBeenCalledWith('src/a.ts')
  expect(screen.getByRole('button', { name: /^src\/gone\.ts:/ })).toBeDisabled()
})

it('shows nothing when nothing was rewritten, or before the report is read', () => {
  const { container } = show(report({ rewritten: 0, files: [] }))
  expect(container).toBeEmptyDOMElement()
  show(null)
  expect(screen.queryByRole('region', { name: 'Rewritten along the way' })).toBeNull()
})

it('says when uncommitted changes and older commits were left out', () => {
  show(report({ capped: true }), true)
  expect(screen.getByText('Only the newest 2 commits were read.')).toBeInTheDocument()
  expect(screen.getByText("Uncommitted changes aren't counted.")).toBeInTheDocument()
})
