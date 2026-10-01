import '@testing-library/jest-dom/vitest'

vi.mock('@current-client-bindings/app', async (importOriginal) => {
  const { mockServices } = await import('./bindings')
  return mockServices(await importOriginal())
})
