import { imageMimeType } from './imageFiles'

describe('imageMimeType', () => {
  it('knows common image types, whatever the case', () => {
    expect(imageMimeType('assets/logo.PNG')).toBe('image/png')
    expect(imageMimeType('photo.jpeg')).toBe('image/jpeg')
    expect(imageMimeType('icon.svg')).toBe('image/svg+xml')
  })

  it('returns null for anything else', () => {
    expect(imageMimeType('main.go')).toBeNull()
    expect(imageMimeType('png')).toBeNull()
    expect(imageMimeType('archive.png.zip')).toBeNull()
  })
})
