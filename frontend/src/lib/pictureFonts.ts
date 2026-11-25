import type { PictureFont } from './picture'

type Loaded = Promise<{ default: string }>

const font = (
  face: PictureFont['face'],
  weight: number,
  load: () => Loaded,
  subset?: { needs: RegExp; range: string },
) => ({
  face,
  weight,
  ...subset,
  load: async () => (await load()).default,
})

// As the app bundles them, by the letters each part holds.
const LATIN_EXT = { needs: /[Ā-ʺḀ-ỿ]/, range: 'U+0100-02BA,U+1E00-1EFF' }
const CYRILLIC = { needs: /[Ѐ-ӿ]/, range: 'U+0400-04FF' }
const GREEK = { needs: /[Ͱ-Ͽ]/, range: 'U+0370-03FF' }

// Loaded when a picture is saved, so they're no part of starting the app.
export const PICTURE_FONTS: PictureFont[] = [
  font('data', 400, () => import('@fontsource/jetbrains-mono/files/jetbrains-mono-latin-400-normal.woff2?inline')),
  font('data', 600, () => import('@fontsource/jetbrains-mono/files/jetbrains-mono-latin-600-normal.woff2?inline')),
  font(
    'data',
    400,
    () => import('@fontsource/jetbrains-mono/files/jetbrains-mono-latin-ext-400-normal.woff2?inline'),
    LATIN_EXT,
  ),
  font(
    'data',
    400,
    () => import('@fontsource/jetbrains-mono/files/jetbrains-mono-cyrillic-400-normal.woff2?inline'),
    CYRILLIC,
  ),
  font(
    'data',
    400,
    () => import('@fontsource/jetbrains-mono/files/jetbrains-mono-greek-400-normal.woff2?inline'),
    GREEK,
  ),
  font('ui', 400, () => import('@fontsource/geist-sans/files/geist-sans-latin-400-normal.woff2?inline')),
  font('ui', 600, () => import('@fontsource/geist-sans/files/geist-sans-latin-600-normal.woff2?inline')),
]
