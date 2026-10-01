import { changelogPdfBase64 } from './changelogPdf'

describe('changelogPdfBase64', () => {
  it('renders changelog HTML to a PDF', async () => {
    const base64 = await changelogPdfBase64(
      '<h2>[1.0.0] - 2026-09-28</h2><h3>Added</h3><ul><li><strong>history:</strong> filter</li></ul>',
    )

    expect(atob(base64).startsWith('%PDF-')).toBe(true)
  })
})
