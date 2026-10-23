import { System } from '@wailsio/runtime'
import { fileManagerName, revealInFileManagerLabel } from './fileManager'

vi.mock('@wailsio/runtime', () => ({ System: { IsMac: vi.fn(), IsWindows: vi.fn() } }))

describe('revealInFileManagerLabel', () => {
  it('names each system’s file manager', () => {
    vi.mocked(System.IsMac).mockReturnValue(true)
    expect(revealInFileManagerLabel()).toBe('Reveal in Finder')

    vi.mocked(System.IsMac).mockReturnValue(false)
    vi.mocked(System.IsWindows).mockReturnValue(true)
    expect(revealInFileManagerLabel()).toBe('Reveal in File Explorer')

    vi.mocked(System.IsWindows).mockReturnValue(false)
    expect(revealInFileManagerLabel()).toBe('Open in File Manager')
  })
})

describe('fileManagerName', () => {
  it('is the short name used in the sidebar row', () => {
    vi.mocked(System.IsMac).mockReturnValue(true)
    expect(fileManagerName()).toBe('Finder')
    vi.mocked(System.IsMac).mockReturnValue(false)
    vi.mocked(System.IsWindows).mockReturnValue(true)
    expect(fileManagerName()).toBe('File Explorer')
  })
})
