import '@testing-library/jest-dom/vitest'

// Every Current Client service call rejects until a test stubs it, as in Current Client's own tests.
vi.mock('@current-client-bindings/app', async (importOriginal) => {
  const { mockServices } = await import('@current-client/test/bindings')
  return mockServices(await importOriginal())
})
