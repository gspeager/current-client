import { useState } from 'react'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  ChangelogService,
  PlatformService,
  type ChangelogEntry,
  type ChangelogReleaseSection,
} from '@current-client-bindings/app'
import type { ChangelogPrefs } from '@current-client-bindings/internal/config'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import TabBar from '../repositories/TabBar'
import { changelogPdfBase64 } from './changelogPdf'
import { giveElementsLayout } from '../../test/bindings'
import ChangelogView from './ChangelogView'

vi.mock('./changelogPdf', () => ({ changelogPdfBase64: vi.fn() }))

const REPO = '/repos/app'

function entry(
  sha: string,
  type: string,
  section: string,
  description: string,
  scope = '',
  author = 'Ada Lovelace',
): ChangelogEntry {
  return { sha: sha.padEnd(40, '0'), type, scope, description, section, author, coAuthors: [], date: '2026-09-20' }
}

const typed = [
  entry('a1', 'feat', 'Added', 'filter by type', 'history'),
  entry('b2', 'fix', 'Fixed', 'keep lane color', '', 'Grace Hopper'),
  entry('c3', 'docs', 'Documentation', 'update readme'),
  entry('d4', '', 'Other', 'Plain subject'),
]

function release(entries: ChangelogEntry[], tag = '', version = '', date = ''): ChangelogReleaseSection {
  const sections = [...new Set(entries.map((e) => e.section))]
  return { tag, version, date, suggestedVersion: tag ? '' : '0.2.0', entries, sections }
}

const DEFAULT_PREFS = {
  types: ['breaking', 'feat', 'fix', 'perf', 'revert'],
  authorNames: false,
  dates: false,
  preview: false,
  splitByRelease: false,
}
const onPrefsChange = vi.fn()

// Holds the preferences in state the way App's settings do.
function Harness({ onOpenCommit, initial }: { onOpenCommit: (sha: string) => void; initial: ChangelogPrefs }) {
  const [prefs, setPrefs] = useState<ChangelogPrefs>(initial)
  return (
    <ChangelogView
      repoPath={REPO}
      repoVersion={0}
      prefs={prefs}
      onPrefsChange={(next) => {
        onPrefsChange(next)
        setPrefs(next)
      }}
      onOpenCommit={onOpenCommit}
    />
  )
}

function renderView(
  entries: ChangelogEntry[],
  onOpenCommit = vi.fn(),
  releases = [release(entries)],
  prefs: ChangelogPrefs = DEFAULT_PREFS,
) {
  vi.mocked(ChangelogService.LatestTag).mockResolvedValue('v0.1.0')
  vi.mocked(ChangelogService.Build).mockResolvedValue(releases)
  vi.mocked(ChangelogService.Markdown).mockImplementation(
    (sections, options) =>
      Promise.resolve(
        sections
          .map(
            (s) =>
              `## [${s.version}]\n` +
              s.entries.map((e) => `- ${e.description}${options.authors ? ` (${e.author})` : ''}`).join('\n'),
          )
          .join('\n'),
      ) as ReturnType<typeof ChangelogService.Markdown>,
  )
  render(
    <DialogProvider>
      <Harness onOpenCommit={onOpenCommit} initial={prefs} />
    </DialogProvider>,
  )
}

const markdown = () => screen.getByRole('textbox', { name: 'Changelog Markdown' })

beforeEach(() => giveElementsLayout())

describe('ChangelogView', () => {
  it('says there are no commits yet in a new repository', async () => {
    renderView([], vi.fn(), [])

    expect(await screen.findByText('No commits yet.')).toBeInTheDocument()
  })

  it('builds from the latest tag and groups commits by section', async () => {
    const onOpenCommit = vi.fn()
    renderView(typed, onOpenCommit)

    await screen.findByRole('heading', { name: 'Added' })
    expect(ChangelogService.Build).toHaveBeenCalledWith(REPO, 'v0.1.0', 'HEAD', '', false)
    // Headings and commits in list order: each commit sits under its section.
    const list = document.querySelector('.changelog-entry-list')!
    expect([...list.querySelectorAll('h3, .changelog-entry-text')].map((el) => el.textContent)).toEqual([
      'Added',
      'history: filter by type',
      'Fixed',
      'keep lane color',
    ])
    await waitFor(() => expect(markdown()).toHaveValue('## [0.2.0]\n- filter by type\n- keep lane color'))

    await userEvent.click(screen.getByRole('button', { name: 'a100000' }))
    expect(onOpenCommit).toHaveBeenCalledWith(typed[0].sha)
  })

  it('filters by type, showing only the release types at first', async () => {
    renderView(typed)
    await screen.findByRole('heading', { name: 'Added' })

    expect(screen.getByText('2 of 4')).toBeInTheDocument()
    await userEvent.click(screen.getByText('Types'))
    expect(screen.getByRole('checkbox', { name: 'feat' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'docs' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'other' })).not.toBeChecked()
    expect(screen.queryByRole('heading', { name: 'Documentation' })).toBeNull()

    await userEvent.click(screen.getByRole('checkbox', { name: 'docs' }))
    expect(screen.getByRole('heading', { name: 'Documentation' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Include update readme' })).toBeInTheDocument()
    await waitFor(() =>
      expect(markdown()).toHaveValue('## [0.2.0]\n- filter by type\n- keep lane color\n- update readme'),
    )

    await userEvent.click(screen.getByRole('checkbox', { name: 'feat' }))
    expect(screen.queryByRole('heading', { name: 'Added' })).toBeNull()
  })

  it('filters by author', async () => {
    renderView(typed)
    await screen.findByRole('heading', { name: 'Fixed' })

    await userEvent.click(screen.getByText('Authors'))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Grace Hopper' }))

    expect(screen.queryByRole('heading', { name: 'Fixed' })).toBeNull()
    expect(screen.getByText('1 of 2')).toBeInTheDocument()
    await waitFor(() => expect(markdown()).toHaveValue('## [0.2.0]\n- filter by type'))
  })

  it('leaves an unchecked commit out of the Markdown', async () => {
    renderView(typed)

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Include keep lane color' }))

    await waitFor(() => expect(markdown()).toHaveValue('## [0.2.0]\n- filter by type'))
  })

  it('adds authors to the Markdown when asked', async () => {
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.click(screen.getByRole('checkbox', { name: 'Author names' }))

    await waitFor(() => expect((markdown() as HTMLTextAreaElement).value).toContain('- keep lane color (Grace Hopper)'))
    expect(ChangelogService.Markdown).toHaveBeenLastCalledWith(expect.any(Array), {
      authors: true,
      dates: false,
    })
  })

  it('asks before regenerating over hand edits', async () => {
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.type(markdown(), ' edited')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Include keep lane color' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))

    expect((markdown() as HTMLTextAreaElement).value).toContain('edited')
    expect(screen.getByRole('checkbox', { name: 'Include keep lane color' })).toBeChecked()
  })

  it('writes a new section to CHANGELOG.md', async () => {
    vi.mocked(ChangelogService.ExistingVersions).mockResolvedValue([])
    vi.mocked(ChangelogService.Write).mockResolvedValue(undefined)
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.click(screen.getByRole('button', { name: 'Write to CHANGELOG.md' }))

    expect(ChangelogService.Write).toHaveBeenCalledWith(REPO, expect.stringContaining('## [0.2.0]'), false)
    expect(await screen.findByText(/Written to CHANGELOG.md/)).toBeInTheDocument()
  })

  it('replaces an existing section only after confirming', async () => {
    vi.mocked(ChangelogService.ExistingVersions).mockResolvedValue(['0.2.0'])
    vi.mocked(ChangelogService.Write).mockResolvedValue(undefined)
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.click(screen.getByRole('button', { name: 'Write to CHANGELOG.md' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Replace' }))

    expect(ChangelogService.Write).toHaveBeenCalledWith(REPO, expect.any(String), true)
  })

  it('previews the Markdown as HTML', async () => {
    vi.mocked(ChangelogService.RenderHTML).mockResolvedValue('<h3>Added</h3><ul><li>filter by type</li></ul>')
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.click(screen.getByRole('button', { name: 'Preview' }))

    const preview = await screen.findByLabelText('Changelog preview')
    await waitFor(() => expect(within(preview).getByRole('heading', { name: 'Added', level: 3 })).toBeInTheDocument())
    expect(screen.queryByRole('textbox', { name: 'Changelog Markdown' })).toBeNull()
    expect(ChangelogService.RenderHTML).toHaveBeenCalledWith(expect.stringContaining('- keep lane color'))
  })

  it('copies the Markdown from an icon button', async () => {
    const user = userEvent.setup()
    const writeText = vi.spyOn(navigator.clipboard, 'writeText')
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await user.click(screen.getByRole('button', { name: 'Copy Markdown' }))

    expect(writeText).toHaveBeenCalledWith(expect.stringContaining('- keep lane color'))
  })

  it('saves as Markdown, HTML or PDF from the Write menu', async () => {
    vi.mocked(ChangelogService.SaveMarkdown).mockResolvedValue(false)
    vi.mocked(ChangelogService.SaveHTML).mockResolvedValue(true)
    vi.mocked(ChangelogService.RenderHTML).mockResolvedValue('<p>x</p>')
    vi.mocked(ChangelogService.SavePDF).mockResolvedValue(true)
    vi.mocked(changelogPdfBase64).mockResolvedValue('JVBERi0=')
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)
    const save = async (option: string) => {
      await userEvent.click(screen.getByTitle('More save options'))
      await userEvent.click(screen.getByRole('button', { name: option }))
    }

    await save('Save Markdown…')
    expect(ChangelogService.SaveMarkdown).toHaveBeenCalledWith(
      expect.stringContaining('## [0.2.0]'),
      'CHANGELOG-0.2.0.md',
    )
    expect(screen.queryByText('Saved.')).toBeNull()

    await save('Save as HTML…')
    expect(ChangelogService.SaveHTML).toHaveBeenCalledWith(
      expect.stringContaining('## [0.2.0]'),
      'CHANGELOG-0.2.0.html',
    )
    expect(await screen.findByText('Saved.')).toBeInTheDocument()

    await save('Save as PDF…')
    await waitFor(() => expect(ChangelogService.SavePDF).toHaveBeenCalledWith('JVBERi0=', 'CHANGELOG-0.2.0.pdf'))
  })

  it('opens the Conventional Commits guide from the help button', async () => {
    renderView(typed)
    await screen.findByRole('heading', { name: 'Added' })

    await userEvent.click(screen.getByRole('button', { name: 'Conventional Commits guide' }))

    const dialog = await screen.findByRole('dialog', { name: 'Conventional Commits' })
    expect(within(dialog).getByText('type(scope)!: description')).toBeInTheDocument()
    expect(within(dialog).getByRole('cell', { name: 'perf, revert' })).toBeInTheDocument()
    expect(within(dialog).getByRole('cell', { name: 'Other' })).toBeInTheDocument()

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('explains the format when no commit in the range follows it', async () => {
    vi.mocked(PlatformService.OpenConventionalCommitsPage).mockResolvedValue(undefined)
    renderView([entry('d4', '', 'Other', 'Plain subject'), entry('e5', '', 'Other', 'Another')])

    expect(await screen.findByText('No Conventional Commits between v0.1.0 and HEAD')).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Changelog Markdown' })).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'conventionalcommits.org' }))
    expect(PlatformService.OpenConventionalCommitsPage).toHaveBeenCalled()
  })

  it('shows the normal panes once one commit follows the format', async () => {
    renderView([entry('d4', '', 'Other', 'Plain subject'), entry('a1', 'fix', 'Fixed', 'one')])

    expect(await screen.findByRole('heading', { name: 'Fixed' })).toBeInTheDocument()
    expect(screen.queryByText(/No Conventional Commits/)).toBeNull()
  })

  it('builds from a time preset instead of a ref', async () => {
    renderView([])
    await screen.findByText('No commits between v0.1.0 and HEAD.')

    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Changelog from' }), 'Past 30 days')

    expect(await screen.findByText('No commits in the past 30 days.')).toBeInTheDocument()
    const [, from, to, since] = vi.mocked(ChangelogService.Build).mock.lastCall!
    expect([from, to]).toEqual(['', 'HEAD'])
    expect(Date.now() - new Date(since).getTime()).toBeGreaterThan(29 * 24 * 60 * 60 * 1000)
  })

  it('splits the range into one section per release', async () => {
    const releases = [
      { ...release([typed[0]]), suggestedVersion: '0.3.0' },
      release([typed[1]], 'v0.2.0', '0.2.0', '2026-09-20'),
      release([entry('e5', 'feat', 'Added', 'first release')], 'v0.1.0', '0.1.0', '2026-08-02'),
    ]
    renderView(typed, vi.fn(), releases)
    await screen.findByRole('heading', { name: /^0\.2\.0/ })

    await userEvent.click(screen.getByRole('checkbox', { name: 'Split by release' }))
    expect(ChangelogService.Build).toHaveBeenLastCalledWith(REPO, 'v0.1.0', 'HEAD', '', true)

    const headings = screen.getAllByRole('heading').map((h) => h.textContent)
    expect(headings).toEqual(['0.3.0', 'Added', '0.2.02026-09-20', 'Fixed', '0.1.02026-08-02', 'Added'])
    await waitFor(() =>
      expect(markdown()).toHaveValue(
        '## [0.3.0]\n- filter by type\n## [0.2.0]\n- keep lane color\n## [0.1.0]\n- first release',
      ),
    )
    const [sections] = vi.mocked(ChangelogService.Markdown).mock.lastCall!
    expect(sections.map((s) => s.date)).toEqual([
      expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/),
      '2026-09-20',
      '2026-08-02',
    ])
  })

  it('hides the version field when every section is a tagged release', async () => {
    renderView(typed, vi.fn(), [release([typed[1]], 'v0.2.0', '0.2.0', '2026-09-20')])

    await screen.findByRole('heading', { name: 'Fixed' })
    expect(screen.queryByLabelText('Version')).toBeNull()
    await waitFor(() => expect(markdown()).toHaveValue('## [0.2.0]\n- keep lane color'))
  })

  it('asks once before replacing several existing sections', async () => {
    vi.mocked(ChangelogService.ExistingVersions).mockResolvedValue(['0.1.0', '0.2.0'])
    vi.mocked(ChangelogService.Write).mockResolvedValue(undefined)
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.click(screen.getByRole('button', { name: 'Write to CHANGELOG.md' }))
    expect(
      await screen.findByText('CHANGELOG.md already has sections for 0.1.0, 0.2.0. Replace them?'),
    ).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Replace' }))

    expect(ChangelogService.Write).toHaveBeenCalledTimes(1)
    expect(ChangelogService.Write).toHaveBeenCalledWith(REPO, expect.any(String), true)
  })

  it('starts from saved preferences', async () => {
    vi.mocked(ChangelogService.RenderHTML).mockResolvedValue('<p>rendered</p>')
    renderView(typed, vi.fn(), [release(typed)], {
      types: ['docs'],
      authorNames: true,
      dates: false,
      preview: true,
      splitByRelease: true,
    })

    expect(await screen.findByRole('heading', { name: 'Documentation' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Added' })).toBeNull()
    expect(screen.getByRole('checkbox', { name: 'Author names' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Split by release' })).toBeChecked()
    expect(ChangelogService.Build).toHaveBeenCalledWith(REPO, 'v0.1.0', 'HEAD', '', true)
    expect(await screen.findByLabelText('Changelog preview')).toHaveTextContent('rendered')
  })

  it('saves changed preferences, but not hidden authors', async () => {
    renderView(typed)
    await screen.findByRole('heading', { name: 'Added' })

    await userEvent.click(screen.getByRole('checkbox', { name: 'Dates' }))
    expect(onPrefsChange).toHaveBeenLastCalledWith(expect.objectContaining({ dates: true }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'docs' }))
    expect(onPrefsChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ types: expect.arrayContaining(['docs', 'feat']) }),
    )

    const saves = onPrefsChange.mock.calls.length
    await userEvent.click(screen.getByRole('checkbox', { name: 'Grace Hopper' }))
    expect(onPrefsChange).toHaveBeenCalledTimes(saves)
  })

  it('blocks copying and writing while the Markdown regenerates', async () => {
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)
    let finish: (text: string) => void = () => {}
    vi.mocked(ChangelogService.Markdown).mockImplementationOnce(
      () =>
        new Promise<string>((resolve) => {
          finish = resolve
        }) as ReturnType<typeof ChangelogService.Markdown>,
    )

    await userEvent.click(screen.getByRole('checkbox', { name: 'Include keep lane color' }))

    expect(screen.getByRole('status')).toHaveTextContent('Updating…')
    expect(screen.getByRole('button', { name: 'Write to CHANGELOG.md' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Copy Markdown' })).toBeDisabled()
    expect(markdown()).toHaveAttribute('readonly')

    finish('## [0.2.0]\n- filter by type')
    await waitFor(() => expect(screen.getByRole('button', { name: 'Write to CHANGELOG.md' })).toBeEnabled())
    expect(screen.queryByRole('status')).toBeNull()
    expect(markdown()).toHaveValue('## [0.2.0]\n- filter by type')
  })

  it('shows progress and blocks other saves while a PDF is prepared', async () => {
    let finish: (pdf: string) => void = () => {}
    vi.mocked(changelogPdfBase64).mockImplementation(
      () =>
        new Promise<string>((resolve) => {
          finish = resolve
        }),
    )
    vi.mocked(ChangelogService.RenderHTML).mockResolvedValue('<p>x</p>')
    vi.mocked(ChangelogService.SavePDF).mockResolvedValue(true)
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)

    await userEvent.click(screen.getByTitle('More save options'))
    await userEvent.click(screen.getByRole('button', { name: 'Save as PDF…' }))

    expect(await screen.findByRole('status')).toHaveTextContent('Preparing PDF…')
    expect(screen.getByRole('button', { name: 'Write to CHANGELOG.md' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Save as HTML…' })).toBeDisabled()

    finish('JVBERi0=')
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Write to CHANGELOG.md' })).toBeEnabled()
  })

  it('shows why the Markdown could not be rendered', async () => {
    renderView(typed)
    await screen.findByDisplayValue(/keep lane color/)
    vi.mocked(ChangelogService.Markdown).mockRejectedValueOnce(new Error('bridge closed'))

    await userEvent.click(screen.getByRole('checkbox', { name: 'Dates' }))

    expect(await screen.findByText('Could not render the Markdown: bridge closed')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Write to CHANGELOG.md' })).toBeDisabled()
  })

  it('renders only the visible part of a long range', async () => {
    const many = Array.from({ length: 5000 }, (_, i) => entry(`f${i}`, 'feat', 'Added', `change ${i}`))
    renderView(many)

    await screen.findByRole('heading', { name: 'Added' })
    const rendered = document.querySelectorAll('.changelog-entry').length
    expect(rendered).toBeGreaterThan(0)
    expect(rendered).toBeLessThan(200)
  })

  it('says when the range is empty', async () => {
    renderView([])

    expect(await screen.findByText('No commits between v0.1.0 and HEAD.')).toBeInTheDocument()
  })
})

describe('TabBar', () => {
  it('hides the Changelog tab when Conventional Commits are turned off', () => {
    const { rerender } = render(<TabBar active="activity" onChange={vi.fn()} showChangelog />)
    expect(screen.getByRole('button', { name: 'Changelog' })).toBeInTheDocument()

    rerender(<TabBar active="activity" onChange={vi.fn()} showChangelog={false} />)
    expect(screen.queryByRole('button', { name: 'Changelog' })).toBeNull()
  })
})
