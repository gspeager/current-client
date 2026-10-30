import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useEffect } from 'react'
import { HostService } from '@current-client-bindings/app'
import { DialogProvider } from '../../components/chrome/DialogProvider'
import { ReviewDraftProvider, reviewTarget, useReviewDraft } from '../../lib/reviewDraft'
import ReviewDialog from './ReviewDialog'

function Drafts({ target }: { target: string }) {
  const { add, comments } = useReviewDraft(target)
  useEffect(() => {
    if (comments.length === 0) add(target, { path: 'app/x.go', line: 2, oldLine: 0, body: 'Rename this.' })
  }, [add, comments.length, target])
  return null
}

function renderDialog({ drafted = false } = {}) {
  const onSubmitted = vi.fn()
  const onClose = vi.fn()
  render(
    <DialogProvider>
      <ReviewDraftProvider>
        {drafted && <Drafts target={reviewTarget('/repos/app', 7)} />}
        <ReviewDialog repoPath="/repos/app" number={7} prefix="!" onSubmitted={onSubmitted} onClose={onClose} />
      </ReviewDraftProvider>
    </DialogProvider>,
  )
  return { onSubmitted, onClose }
}

describe('ReviewDialog', () => {
  it('offers the verdicts the host allows and submits the review with the drafted comments', async () => {
    vi.mocked(HostService.ReviewCapabilities).mockResolvedValue({
      reply: true,
      resolve: true,
      lineComments: true,
      approve: true,
      requestChanges: false,
    } as never)
    vi.mocked(HostService.SubmitReview).mockResolvedValue({} as never)
    const { onSubmitted } = renderDialog({ drafted: true })

    const approve = await screen.findByRole('radio', { name: /Approve/ })
    expect(approve).toBeChecked()
    expect(screen.queryByRole('radio', { name: /Request changes/ })).toBeNull()
    expect(screen.getByText(/no request-changes verdict/)).toBeInTheDocument()
    expect(screen.getByText('1 drafted comment is sent with the review.')).toBeInTheDocument()
    expect(HostService.SubmitReview).not.toHaveBeenCalled()
    await userEvent.type(screen.getByRole('textbox', { name: 'Summary' }), 'Looks good.')
    await userEvent.click(screen.getByRole('button', { name: 'Submit review' }))
    await waitFor(() => expect(onSubmitted).toHaveBeenCalled())
    expect(HostService.SubmitReview).toHaveBeenCalledWith('/repos/app', 7, {
      verdict: 'approve',
      summary: 'Looks good.',
      comments: [{ path: 'app/x.go', line: 2, oldLine: 0, body: 'Rename this.' }],
    })
  })

  it('needs a summary or a drafted comment for a plain comment, and shows a refusal', async () => {
    vi.mocked(HostService.ReviewCapabilities).mockResolvedValue({
      reply: true,
      resolve: false,
      lineComments: true,
      approve: true,
      requestChanges: true,
    } as never)
    vi.mocked(HostService.SubmitReview).mockRejectedValue(
      new Error('The token for github.example.com lacks the repo scope this needs.'),
    )
    const { onSubmitted } = renderDialog()

    await userEvent.click(await screen.findByRole('radio', { name: /Comment/ }))
    expect(screen.getByRole('button', { name: 'Submit review' })).toBeDisabled()
    await userEvent.click(screen.getByRole('radio', { name: /Request changes/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Submit review' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('lacks the repo scope')
    expect(HostService.SubmitReview).toHaveBeenCalledWith('/repos/app', 7, {
      verdict: 'request-changes',
      summary: '',
      comments: [],
    })
    expect(onSubmitted).not.toHaveBeenCalled()
  })
})
