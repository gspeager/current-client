import htmlToPdfmake from 'html-to-pdfmake'

// pdfmake is a UMD bundle: depending on the bundler its API arrives as the
// module itself or as its default export.
const unwrap = <T>(mod: T): T => (mod as { default?: T }).default ?? mod

// pdfmake and its bundled fonts are about 1 MB, so they load only when a PDF is saved.
export async function changelogPdfBase64(html: string): Promise<string> {
  const [pdfMakeModule, vfs] = await Promise.all([import('pdfmake/build/pdfmake'), import('pdfmake/build/vfs_fonts')])
  const pdfMake = unwrap(pdfMakeModule)
  pdfMake.addVirtualFileSystem(vfs.default)
  return pdfMake.createPdf({ content: htmlToPdfmake(html), defaultStyle: { fontSize: 10 } }).getBase64()
}
