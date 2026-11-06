import { createContext, useContext } from 'react'

export type Density = 'comfortable' | 'compact'

export const DensityContext = createContext<Density>('comfortable')

export const useDensity = () => useContext(DensityContext)

// Virtualized lists position rows themselves, so these must match the row styles.
export const commitRowHeight = (density: Density) => (density === 'compact' ? 28 : 38)
export const fileRowHeight = (density: Density) => (density === 'compact' ? 26 : 28)
