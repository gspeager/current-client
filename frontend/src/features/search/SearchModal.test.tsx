import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  BranchService,
  PlatformService,
  SearchService,
  TagService,
  type SearchResult,
} from '@current-client-bindings/app'
import SearchModal from './SearchModal'

const REPO = '/repos/app'

// The bindings return cancellable promises; a new search cancels the last.
function cancellable<T>(promise: Promise<T>) {
  return Object.assign(promise, { cancel: vi.fn(() => Promise.resolve()) }) as never
}

function results(result: SearchResult) {
  vi.mocked(SearchService.SearchContents).mockImplementation(() => cancellable(Promise.resolve(result)))
}

function renderSearch(onOpenFileHistory = vi.fn()) {
  vi.mocked(BranchService.ListLocal).mockResolvedValue([
    { name: 'main', current: true, upstream: '', ahead: 0, behind: 0, lastCommitDate: '', worktreePath: '' },
  ])
  vi.mocked(BranchService.ListRemote).mockResolvedValue([])
  vi.mocked(TagService.ListTags).mockResolvedValue([])
  render(<SearchModal repoPath={REPO} onClose={vi.fn()} onOpenFileHistory={onOpenFileHistory} />)
  return onOpenFileHistory
}

describe('SearchModal', () => {
  it('searches the working tree as you type and groups matches by file', async () => {
    results({
      matches: [
        { path: 'src/a.ts', line: 3, text: 'const foo = 1' },
        { path: 'src/a.ts', line: 9, text: 'return foo' },
        { path: 'README.md', line: 1, text: 'Foo bar' },
      ],
      truncated: false,
    })
    vi.mocked(PlatformService.OpenFileInEditor).mockResolvedValue()
    renderSearch()

    await userEvent.type(screen.getByRole('textbox', { name: 'Search for' }), 'foo')

    expect(await screen.findByText('3 matches in 2 files')).toBeInTheDocument()
    expect(SearchService.SearchContents).toHaveBeenLastCalledWith(REPO, {
      pattern: 'foo',
      rev: '',
      ignoreCase: true,
      wholeWord: false,
      regexp: false,
    })
    const line = screen.getByRole('button', { name: /9\s*return foo/ })
    expect(within(line).getByText('foo', { selector: 'mark' })).toBeInTheDocument()

    await userEvent.click(line)
    expect(PlatformService.OpenFileInEditor).toHaveBeenCalledWith(REPO, 'src/a.ts', 9)
  })

  it('opens a match from a past commit in File history', async () => {
    results({
      matches: [{ path: 'src/a.ts', line: 3, text: 'const foo = 1' }],
      truncated: false,
    })
    const onOpenFileHistory = renderSearch()

    await userEvent.selectOptions(await screen.findByRole('combobox', { name: 'Search in' }), 'main')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Match case' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Search for' }), 'foo')
    await userEvent.click(await screen.findByRole('button', { name: /3\s*const foo/ }))

    expect(SearchService.SearchContents).toHaveBeenLastCalledWith(
      REPO,
      expect.objectContaining({ rev: 'main', ignoreCase: false }),
    )
    expect(onOpenFileHistory).toHaveBeenCalledWith('src/a.ts')
    expect(PlatformService.OpenFileInEditor).not.toHaveBeenCalled()
  })

  it('says when results were cut short, and shows search errors', async () => {
    results({
      matches: [{ path: 'a.ts', line: 1, text: 'x' }],
      truncated: true,
    })
    renderSearch()

    await userEvent.type(screen.getByRole('textbox', { name: 'Search for' }), 'x')
    expect(await screen.findByText('Showing the first 1 matches. Narrow the search to see more.')).toBeInTheDocument()

    vi.mocked(SearchService.SearchContents).mockImplementation(() =>
      cancellable(Promise.reject(new Error('Search failed: parentheses not balanced'))),
    )
    await userEvent.type(screen.getByRole('textbox', { name: 'Search for' }), '(')
    expect(await screen.findByText('Search failed: parentheses not balanced')).toBeInTheDocument()
  })
})
