export type Step = 1 | 2 | 3 | 4

export interface StepLegend {
  step: Step
  label: string
}

const range = (from: number, to: number) => (from === to ? `${from}` : `${from}–${to}`)

// Commits in four steps, none and then thirds of what there is, so a quiet
// repository has as much to show as a busy one.
export function commitSteps(counts: number[]): { stepOf: (count: number) => Step; legend: StepLegend[] } {
  const some = counts.filter((count) => count > 0).sort((a, b) => a - b)
  const values = [...new Set(some)]
  const tops =
    values.length <= 3
      ? values
      : [
          ...new Set([
            some[Math.floor(some.length / 3)],
            some[Math.floor((some.length * 2) / 3)],
            values[values.length - 1],
          ]),
        ]
  // The last of the steps, so the most is always the brightest.
  const first = 5 - tops.length
  const stepOf = (count: number): Step => {
    if (count <= 0) return 1
    const at = tops.findIndex((top) => count <= top)
    return (first + (at < 0 ? tops.length - 1 : at)) as Step
  }
  const legend: StepLegend[] = [{ step: 1, label: 'None' }]
  tops.forEach((top, i) => {
    const from = i === 0 ? 1 : tops[i - 1] + 1
    legend.push({ step: (first + i) as Step, label: range(i === 0 ? values[0] : from, top) })
  })
  return { stepOf, legend }
}

const DAY = 24 * 60 * 60 * 1000
const AGES: { days: number; step: Step; label: string }[] = [
  { days: 7, step: 4, label: 'Last 7 days' },
  { days: 30, step: 3, label: 'Last 30 days' },
  { days: 180, step: 2, label: 'Last 180 days' },
]

export const ageLegend: StepLegend[] = [
  { step: 1, label: 'Earlier' },
  ...AGES.map(({ step, label }) => ({ step, label })).reverse(),
]

export function ageStep(last: number | undefined, now: number): Step {
  if (last === undefined) return 1
  return AGES.find(({ days }) => now - last <= days * DAY)?.step ?? 1
}
