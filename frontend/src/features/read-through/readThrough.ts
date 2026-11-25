import { createContext, useContext } from 'react'
import type { ReadThroughRequest } from '@current-client-bindings/app'

export type ReadThroughTarget = Pick<ReadThroughRequest, 'ref' | 'path' | 'target' | 'uncommitted'>

type OpenReadThrough = (request: Partial<ReadThroughTarget>) => void

export const ReadThroughContext = createContext<OpenReadThrough>(() => undefined)

export function useOpenReadThrough(): OpenReadThrough {
  return useContext(ReadThroughContext)
}
