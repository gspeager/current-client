import { vi } from 'vitest'

type Module = Record<string, unknown>

// Every generated service method becomes a mock that rejects until a test
// stubs it, so an unexpected call to the Go backend fails loudly instead of
// resolving to undefined. vitest's mockReset restores this between tests.
export function mockServices(bindings: Module): Module {
  return Object.fromEntries(
    Object.entries(bindings).map(([name, value]) => [
      name,
      name.endsWith('Service') ? mockService(name, value as Module) : value,
    ]),
  )
}

function mockService(serviceName: string, service: Module): Module {
  return Object.fromEntries(
    Object.entries(service).map(([method, fn]) => [
      method,
      typeof fn === 'function'
        ? vi.fn(() => Promise.reject(new Error(`Unstubbed binding: ${serviceName}.${method}`)))
        : fn,
    ]),
  )
}

// Virtualized lists measure their scroll container, which jsdom reports as 0×0.
export function giveElementsLayout(width = 1000, height = 800) {
  vi.spyOn(HTMLElement.prototype, 'offsetWidth', 'get').mockReturnValue(width)
  vi.spyOn(HTMLElement.prototype, 'offsetHeight', 'get').mockReturnValue(height)
}
