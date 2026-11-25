import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CommitService, type FileStatus } from '@current-client-bindings/app'
import RiskReport from './RiskReport'

const staged = [{ path: 'app.go', indexAdded: 1, indexRemoved: 0 } as FileStatus]
const finding = { rule: 'secret', path: 'app.go', line: 3, oldLine: 0, message: 'Looks like github pat: ghp_…' }
const report = (findings: (typeof finding)[], leftOut = {}) => ({ findings, leftOut, commitsNotRead: false })

describe('RiskReport', () => {
  it('lists findings and opens a finding on its line', async () => {
    const user = userEvent.setup()
    vi.mocked(CommitService.GetRiskReport).mockResolvedValue(report([finding]))
    const onOpen = vi.fn()
    render(<RiskReport repoPath="/repo" staged={staged} rulesOff={[]} onOpen={onOpen} />)

    await user.click(await screen.findByRole('button', { name: /Looks like github pat/ }))

    expect(onOpen).toHaveBeenCalledWith('app.go', 3, 0)
  })

  it('opens a finding about a removed line at that line, and counts what it leaves out', async () => {
    const removed = { ...finding, rule: 'fewer-tests', line: 0, oldLine: 7, message: '1 test fewer.' }
    vi.mocked(CommitService.GetRiskReport).mockResolvedValue(report([removed], { 'fewer-tests': 4 }))
    const onOpen = vi.fn()
    render(<RiskReport repoPath="/repo" staged={staged} rulesOff={[]} onOpen={onOpen} />)

    expect(await screen.findByText('5 things to check before committing')).toBeInTheDocument()
    expect(screen.getByText('4 more not listed')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /1 test fewer/ }))
    expect(onOpen).toHaveBeenCalledWith('app.go', 0, 7)
  })

  it('shows nothing with no findings or nothing staged', async () => {
    vi.mocked(CommitService.GetRiskReport).mockResolvedValue(report([]))
    const { container, rerender } = render(
      <RiskReport repoPath="/repo" staged={staged} rulesOff={[]} onOpen={vi.fn()} />,
    )
    await vi.waitFor(() => expect(CommitService.GetRiskReport).toHaveBeenCalled())
    expect(container).toBeEmptyDOMElement()

    vi.mocked(CommitService.GetRiskReport).mockClear()
    rerender(<RiskReport repoPath="/other" staged={[]} rulesOff={[]} onOpen={vi.fn()} />)
    expect(CommitService.GetRiskReport).not.toHaveBeenCalled()
  })

  it('reads again when staged changes or rules change', async () => {
    vi.mocked(CommitService.GetRiskReport).mockResolvedValue(report([finding]))
    const { rerender } = render(<RiskReport repoPath="/repo" staged={staged} rulesOff={[]} onOpen={vi.fn()} />)
    await screen.findByRole('button', { name: /Looks like/ })

    rerender(<RiskReport repoPath="/repo" staged={staged} rulesOff={['secret']} onOpen={vi.fn()} />)
    await vi.waitFor(() => expect(CommitService.GetRiskReport).toHaveBeenCalledTimes(2))

    rerender(
      <RiskReport repoPath="/repo" staged={[{ ...staged[0], indexAdded: 2 }]} rulesOff={['secret']} onOpen={vi.fn()} />,
    )
    await vi.waitFor(() => expect(CommitService.GetRiskReport).toHaveBeenCalledTimes(3))
  })
})
