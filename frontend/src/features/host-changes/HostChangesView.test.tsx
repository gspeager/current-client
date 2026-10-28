import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  ChangeList,
  CurrentChange,
  ForecastReport,
  HostService,
  PlatformService,
  RemoteService,
} from '@current-client-bindings/app'
import { Change, ChangeState, CIStatus, ReviewState } from '@current-client-bindings/core/hostapi'
import { Links } from '@current-client-bindings/core/hostlink'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import HostChangesView from './HostChangesView'

const REPO = '/repos/app'

function change(overrides: Partial<Change>): Change {
  return new Change({
    number: 138,
    title: 'Add the thing',
    state: ChangeState.ChangeOpen,
    sourceBranch: 'feature',
    targetBranch: 'main',
    author: { name: 'Ada Lovelace', username: 'ada' },
    labels: ['backend'],
    headSha: 'abc1234def',
    ci: CIStatus.CISuccess,
    updatedAt: new Date().toISOString(),
    webUrl: 'https://gitlab.example.com/g/r/-/merge_requests/138',
    ...overrides,
  })
}

function list(overrides: Partial<ChangeList> = {}): ChangeList {
  return new ChangeList({
    host: 'gitlab.example.com',
    provider: 'gitlab',
    project: 'g/r',
    enabled: true,
    prefix: '!',
    me: 'ada',
    changes: [
      change({}),
      change({ number: 140, title: 'Fix a typo', author: { name: 'Bob', username: 'bob' }, draft: true, labels: [] }),
    ],
    fetchedAt: new Date().toISOString(),
    ...overrides,
  })
}

function renderView({
  onBranchChanged = vi.fn(),
  onCreateChange = vi.fn(),
  onOpenSettings = vi.fn(),
  onMerged = vi.fn(),
} = {}) {
  vi.mocked(HostService.Forecast).mockResolvedValue(
    new ForecastReport({
      enabled: true,
      branch: 'main',
      conflicts: [
        { number: 138, prefix: '!', title: '', author: '', sourceBranch: 'feature', files: ['app/x.go'], webUrl: '' },
      ],
    }),
  )
  vi.mocked(HostService.CIStatus).mockResolvedValue({})
  vi.mocked(HostService.CurrentChange).mockResolvedValue(
    new CurrentChange({ branch: 'topic', upstream: 'origin/topic', enabled: true, change: null }),
  )
  render(
    <DialogProvider>
      <HostChangesView
        repoPath={REPO}
        label="Merge requests"
        onBranchChanged={onBranchChanged}
        onCreateChange={onCreateChange}
        onOpenSettings={onOpenSettings}
        onMerged={onMerged}
      />
    </DialogProvider>,
  )
  return { onBranchChanged, onCreateChange, onOpenSettings, onMerged }
}

describe('HostChangesView', () => {
  it('lists the open changes from the cache and filters to mine', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '<p>Body</p>',
      closes: [],
      environments: [],
    })
    renderView()

    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(within(rows).getByText('Add the thing')).toBeInTheDocument()
    expect(within(rows).getByText('Fix a typo')).toBeInTheDocument()
    expect(HostService.Refresh).not.toHaveBeenCalled()
    expect(screen.getByText(/gitlab\.example\.com · fetched/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Mine 1' }))
    expect(within(rows).queryByText('Fix a typo')).not.toBeInTheDocument()
    expect(HostService.ClosedChanges).not.toHaveBeenCalled()
  })

  it('shows where reviews stand and filters to the changes waiting on mine', async () => {
    const reviewer = (username: string, state: ReviewState) => ({ reviewer: { name: username, username }, state })
    vi.mocked(HostService.Changes).mockResolvedValue(
      list({
        changes: [
          change({
            reviews: [reviewer('bob', ReviewState.ReviewApproved), reviewer('cy', ReviewState.ReviewApproved)],
          }),
          change({
            number: 140,
            title: 'Fix a typo',
            author: { name: 'Bob', username: 'bob' },
            reviews: [reviewer('ada', ReviewState.ReviewRequested), reviewer('cy', ReviewState.ReviewChangesRequested)],
          }),
          change({
            number: 141,
            title: 'Tidy',
            author: { name: 'Bob', username: 'bob' },
            reviews: [reviewer('ada', ReviewState.ReviewRequested)],
          }),
        ],
      }),
    )
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    renderView()

    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(within(rows).getByText('2 approved')).toBeInTheDocument()
    expect(within(rows).getByText('Changes requested')).toBeInTheDocument()
    expect(within(rows).getByText('Review requested')).toBeInTheDocument()
    expect(screen.getByRole('article', { name: '!138' })).toHaveTextContent('ReviewsApproved: bob, cy')

    await userEvent.click(screen.getByRole('button', { name: 'To review 2' }))
    expect(within(rows).queryByText('Add the thing')).not.toBeInTheDocument()
    expect(within(rows).getByText('Fix a typo')).toBeInTheDocument()
    const detail = screen.getByRole('article', { name: '!140' })
    expect(within(detail).getByText('Changes requested: cy')).toBeInTheDocument()
    expect(within(detail).getByText('Requested: ada')).toBeInTheDocument()
  })

  it('asks for closed changes only when that filter is chosen', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    vi.mocked(HostService.ClosedChanges).mockResolvedValue(
      list({ changes: [change({ number: 99, title: 'Shipped it', state: ChangeState.ChangeMerged })] }),
    )
    renderView()

    await screen.findByRole('list', { name: 'Merge requests list' })
    await userEvent.click(screen.getByRole('button', { name: 'Closed' }))
    const rows = screen.getByRole('list', { name: 'Merge requests list' })
    expect(await within(rows).findByText('Shipped it')).toBeInTheDocument()
    expect(HostService.ClosedChanges).toHaveBeenCalledWith(REPO)
    expect(within(rows).queryByText('Add the thing')).not.toBeInTheDocument()
  })

  it('refreshes once when nothing was fetched yet, and on the button', async () => {
    vi.mocked(HostService.Changes)
      .mockResolvedValueOnce(list({ fetchedAt: null, changes: [] }))
      .mockResolvedValue(list())
    vi.mocked(HostService.Refresh).mockResolvedValue()
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    renderView()

    await waitFor(() => expect(HostService.Refresh).toHaveBeenCalledTimes(1))
    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(await within(rows).findByText('Add the thing')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(HostService.Refresh).toHaveBeenCalledTimes(2)
  })

  it('shows the selected change with its rendered description and opens it on the host', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '<p>See <a href="https://gitlab.example.com/g/r/-/issues/4">#4</a></p>',
      closes: [],
      environments: [],
    })
    vi.mocked(PlatformService.OpenWebPage).mockResolvedValue()
    renderView()

    const detail = await screen.findByRole('article', { name: '!138' })
    expect(await within(detail).findByText('#4')).toBeInTheDocument()
    expect(within(detail).getByText('feature → main')).toBeInTheDocument()
    expect(within(detail).getByText(/Would conflict with main in/)).toBeInTheDocument()
    expect(within(detail).getByText('CI passed')).toBeInTheDocument()
    await userEvent.click(within(detail).getByRole('button', { name: 'Open on gitlab.example.com' }))
    expect(PlatformService.OpenWebPage).toHaveBeenCalledWith('https://gitlab.example.com/g/r/-/merge_requests/138')
    expect(HostService.ChangeDetails).toHaveBeenCalledWith(REPO, 138)
  })

  it('fetches threads, files and commits only when their tab is opened', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    vi.mocked(HostService.ChangeFiles).mockResolvedValue([
      { path: 'app/x.go', oldPath: '', status: 'modified', additions: 3, deletions: 1 },
      { path: 'app/y.go', oldPath: 'app/old.go', status: 'renamed', additions: 0, deletions: 0 },
    ])
    vi.mocked(HostService.ChangeCommits).mockResolvedValue([
      { sha: 'abc1234def', subject: 'feat: x', author: 'Ada', date: new Date().toISOString() },
    ])
    vi.mocked(HostService.ChangeThreads).mockResolvedValue({
      enabled: true,
      number: 138,
      prefix: '!',
      total: 1,
      unresolved: { 'app/x.go': 1 },
      threads: [
        {
          id: 't1',
          path: 'app/x.go',
          line: 12,
          oldLine: 0,
          resolved: false,
          outdated: false,
          notes: [{ author: 'Bob', bodyHtml: '<p>Why?</p>', createdAt: new Date().toISOString() }],
        },
      ],
    })
    renderView()

    const detail = await screen.findByRole('article', { name: '!138' })
    expect(HostService.ChangeFiles).not.toHaveBeenCalled()

    await userEvent.click(within(detail).getByRole('button', { name: 'Files' }))
    const files = await within(detail).findByRole('list', { name: 'Files' })
    expect(within(files).getByText('app/x.go')).toBeInTheDocument()
    expect(within(files).getByText('from app/old.go')).toBeInTheDocument()
    expect(await within(detail).findByRole('button', { name: 'Files 2' })).toBeInTheDocument()

    vi.mocked(HostService.FetchCIStatus).mockResolvedValue({
      abc1234def: { status: 'failed', pipelineId: 7, webUrl: '' },
    })
    vi.mocked(HostService.PipelineJobs).mockResolvedValue([
      { id: 1, name: 'test', stage: 'ci', status: CIStatus.CIFailed, duration: 0, webUrl: '', retryable: false },
    ])
    await userEvent.click(within(detail).getByRole('button', { name: 'Checks' }))
    expect(await within(detail).findByText('test')).toBeInTheDocument()
    expect(HostService.FetchCIStatus).toHaveBeenCalledWith(REPO, ['abc1234def'])
    expect(await within(detail).findByRole('button', { name: 'Checks 1' })).toBeInTheDocument()

    await userEvent.click(within(detail).getByRole('button', { name: 'Commits' }))
    expect(await within(detail).findByText('feat: x')).toBeInTheDocument()
    expect(within(detail).getByText('abc1234')).toBeInTheDocument()

    await userEvent.click(within(detail).getByRole('button', { name: 'Threads' }))
    expect(await within(detail).findByText('Why?')).toBeInTheDocument()
    expect(within(detail).getByRole('group', { name: /app\/x\.go:12/ })).toBeInTheDocument()
    expect(HostService.ChangeThreads).toHaveBeenCalledWith(REPO, 138)
    expect(HostService.ChangeFiles).toHaveBeenCalledTimes(1)
  })

  it('starts a new change from the checked-out branch, unless it already has one', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    const { onCreateChange } = renderView()

    const create = await screen.findByRole('button', { name: 'New merge request' })
    await waitFor(() => expect(create).toBeEnabled())
    await userEvent.click(create)
    expect(onCreateChange).toHaveBeenCalledWith('topic')
  })

  it("names the host in the settings link's tooltip and opens the integration settings from it", async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    const { onOpenSettings } = renderView()

    const link = await screen.findByRole('button', { name: 'Integration settings' })
    expect(link).toHaveAttribute('title', expect.stringMatching(/^gitlab\.example\.com is contacted on Refresh/))
    await userEvent.click(link)
    expect(onOpenSettings).toHaveBeenCalled()
  })

  it('shows what the last refresh knows about merging, merges on the host, and reports a refusal', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    vi.mocked(HostService.MergeReadiness).mockResolvedValue({
      facts: [
        { label: 'Checks', value: 'passed', level: 'ok' },
        { label: 'Reviews', value: 'waiting on Bob', level: 'warn' },
        { label: 'Conflicts', value: 'not known until the change is opened on the host', level: 'unknown' },
      ],
      fetchedAt: new Date().toISOString(),
    } as never)
    vi.mocked(HostService.MergeOptions).mockResolvedValue({
      methods: ['squash', 'merge'],
      deleteBranch: true,
      deleteBranchFixed: false,
    } as never)
    vi.mocked(HostService.MergeChange).mockRejectedValueOnce(
      new Error('gitlab.example.com refused the merge: it conflicts with the target branch.'),
    )
    vi.mocked(HostService.MergeChange).mockResolvedValueOnce({
      number: 138,
      sourceBranch: 'feature',
      localBranch: 'feature',
    } as never)
    const onMerged = vi.fn()
    renderView({ onMerged })

    const detail = await screen.findByRole('article', { name: '!138' })
    // Readiness is the facts table itself: one row per fact, its icon by level, no restated block.
    await within(detail).findByText('waiting on Bob')
    expect(within(detail).getByText('waiting on Bob').closest('dd')).toHaveClass('host-change-fact-warn')
    expect(within(detail).getByText('passed').closest('dd')).toHaveClass('host-change-fact-ok')
    expect(within(detail).queryByText('Merge')).toBeNull()
    // Something stands in the way, so Merge… is there but not the primary.
    expect(within(detail).getByRole('button', { name: 'Merge…' })).toHaveClass('host-change-button')
    await userEvent.click(screen.getByRole('button', { name: 'Merge…' }))

    const dialog = await screen.findByRole('dialog', { name: 'Merge !138' })
    expect(HostService.MergeOptions).toHaveBeenCalledWith(REPO, 138)
    // Only the facts that stand in the way are repeated in the dialog.
    const warnings = await within(dialog).findByRole('list', { name: 'Before merging' })
    expect(within(warnings).queryByText(/Checks/)).toBeNull()
    expect(within(warnings).getByText(/Reviews: waiting on Bob/)).toBeInTheDocument()
    expect(within(dialog).getByRole('radio', { name: 'Squash and merge' })).toBeChecked()
    expect(within(dialog).getByRole('checkbox', { name: /Delete feature on gitlab.example.com/ })).toBeChecked()
    await userEvent.click(within(dialog).getByRole('radio', { name: 'Merge commit' }))
    await userEvent.click(within(dialog).getByRole('button', { name: 'Merge commit' }))
    expect(HostService.MergeChange).toHaveBeenCalledWith(REPO, 138, 'merge', true)
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      'refused the merge: it conflicts with the target branch',
    )
    expect(onMerged).not.toHaveBeenCalled()

    await userEvent.click(within(dialog).getByRole('button', { name: 'Merge commit' }))
    await waitFor(() =>
      expect(onMerged).toHaveBeenCalledWith({ number: 138, sourceBranch: 'feature', localBranch: 'feature' }),
    )
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(screen.getByRole('status')).toHaveTextContent('Merged !138. feature can be cleaned up.')
    expect(screen.queryByRole('button', { name: 'Merge…' })).toBeNull()
  })

  it('submits a review from the details when the host takes one', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    vi.mocked(HostService.ReviewCapabilities).mockResolvedValue({
      reply: true,
      resolve: true,
      lineComments: true,
      approve: true,
      requestChanges: false,
    } as never)
    vi.mocked(HostService.SubmitReview).mockResolvedValue({} as never)
    renderView()

    await userEvent.click(await screen.findByRole('button', { name: 'Review…' }))
    const dialog = await screen.findByRole('dialog', { name: 'Review !138' })
    await userEvent.click(await within(dialog).findByRole('button', { name: 'Submit review' }))
    await waitFor(() =>
      expect(HostService.SubmitReview).toHaveBeenCalledWith(REPO, 138, {
        verdict: 'approve',
        summary: '',
        comments: [],
      }),
    )
    expect(await screen.findByRole('status')).toHaveTextContent('Review submitted.')
    expect(HostService.Changes).toHaveBeenCalledTimes(2)
  })

  it('checks a change out locally and tells the app the branch changed', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    vi.mocked(HostService.CheckoutChange).mockResolvedValue({ branch: 'feature', existed: false })
    const { onBranchChanged } = renderView()

    await userEvent.click(await screen.findByRole('button', { name: 'Check out locally' }))

    expect(HostService.CheckoutChange).toHaveBeenCalledWith(REPO, 138, null)
    expect(await screen.findByRole('status')).toHaveTextContent('Checked out feature.')
    expect(onBranchChanged).toHaveBeenCalled()
  })

  it('renders nothing while the host is off', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(new ChangeList({ enabled: false }))
    renderView()
    await waitFor(() => expect(HostService.Changes).toHaveBeenCalled())
    expect(screen.queryByRole('region')).not.toBeInTheDocument()
  })
})

describe('HostChangesView primary', () => {
  it('makes Merge the primary once nothing stands in the way, and Review when yours is asked for', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(
      list({
        changes: [
          change({
            author: { name: 'Bob', username: 'bob' },
            reviews: [{ reviewer: { name: 'ada', username: 'ada' }, state: ReviewState.ReviewRequested }],
          }),
          change({
            number: 140,
            title: 'Mine, waiting',
            reviews: [{ reviewer: { name: 'Cy', username: 'cy' }, state: ReviewState.ReviewRequested }],
          }),
        ],
      }),
    )
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [],
    })
    vi.mocked(HostService.ReviewCapabilities).mockResolvedValue({
      reply: true,
      resolve: true,
      lineComments: true,
      approve: true,
      requestChanges: false,
    } as never)
    vi.mocked(HostService.MergeReadiness).mockResolvedValue({
      facts: [{ label: 'Checks', value: 'passed', level: 'ok' }],
      fetchedAt: new Date().toISOString(),
    } as never)
    renderView()

    const rows = await screen.findByRole('list', { name: 'Merge requests list' })
    expect(within(rows).getByText('Waiting on Cy')).toBeInTheDocument()
    const detail = await screen.findByRole('article', { name: '!138' })
    expect(await within(detail).findByRole('button', { name: 'Review…' })).toHaveClass('host-changes-new')
    expect(within(detail).getByRole('button', { name: 'Merge…' })).toHaveClass('host-change-button')

    await userEvent.click(within(rows).getByRole('button', { name: /Mine, waiting/ }))
    const mine = await screen.findByRole('article', { name: '!140' })
    await waitFor(() => expect(within(mine).getByRole('button', { name: 'Merge…' })).toHaveClass('host-changes-new'))
  })
})

describe('HostChangesView closes', () => {
  it('lists the issues a change closes as references', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: ['#12', '#7'],
      environments: [],
    })
    vi.mocked(RemoteService.HostLinks).mockResolvedValue(
      new Links({
        root: 'https://gitlab.example.com/',
        issue: 'https://gitlab.example.com/g/r/-/issues/',
        issuePath: '/-/issues/',
      }),
    )
    renderView()
    await userEvent.click(await screen.findByRole('button', { name: /Add the thing/ }))
    const closes = await screen.findByRole('link', { name: '#12' })
    expect(closes).toHaveAttribute('href', 'https://gitlab.example.com/g/r/-/issues/12')
    expect(screen.getByRole('link', { name: '#7' })).toBeInTheDocument()
  })

  it('says where a merged change has reached', async () => {
    vi.mocked(HostService.Changes).mockResolvedValue(list())
    vi.mocked(HostService.ChangeDetails).mockResolvedValue({
      change: change({}),
      descriptionHtml: '',
      closes: [],
      environments: [
        { environment: 'staging', state: 'in', production: false },
        { environment: 'production', state: 'out', production: true },
      ],
    })
    renderView()
    await userEvent.click(await screen.findByRole('button', { name: /Add the thing/ }))
    const environments = await screen.findByRole('list', { name: 'Environments' })
    expect(environments).toHaveTextContent('stagingReached')
    expect(environments).toHaveTextContent('productionNot yet')
  })
})
