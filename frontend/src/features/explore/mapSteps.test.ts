import { ageStep, commitSteps } from './mapSteps'

describe('commitSteps', () => {
  it('splits what there is in thirds, with none by itself', () => {
    const { stepOf, legend } = commitSteps([0, 0, 1, 1, 1, 2, 2, 3, 5, 9, 40])
    expect([0, 1, 2, 3, 9, 40].map(stepOf)).toEqual([1, 2, 2, 3, 4, 4])
    expect(legend.map(({ step, label }) => [step, label])).toEqual([
      [1, 'None'],
      [2, '1–2'],
      [3, '3–5'],
      [4, '6–40'],
    ])
  })

  it('makes the most the brightest, however little there is', () => {
    const one = commitSteps([0, 1, 1])
    expect([0, 1].map(one.stepOf)).toEqual([1, 4])
    expect(one.legend.map(({ step, label }) => [step, label])).toEqual([
      [1, 'None'],
      [4, '1'],
    ])
    const two = commitSteps([1, 3])
    expect([1, 3].map(two.stepOf)).toEqual([3, 4])
    expect(commitSteps([]).legend).toEqual([{ step: 1, label: 'None' }])
  })
})

it('steps by age', () => {
  const now = Date.parse('2026-11-25T00:00:00Z')
  const ago = (days: number) => now - days * 24 * 60 * 60 * 1000
  expect([ago(1), ago(8), ago(100), ago(181), undefined].map((last) => ageStep(last, now))).toEqual([4, 3, 2, 1, 1])
})
