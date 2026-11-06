import { taskProgress } from './taskProgress'

describe('taskProgress', () => {
  it('counts checked and total boxes, in either attribute order', () => {
    const html = '<li><input type="checkbox" disabled checked> a</li><li><input disabled type="checkbox"> b</li>'
    expect(taskProgress(html)).toEqual({ done: 1, total: 2 })
  })

  it('has nothing to report without a task list', () => {
    expect(taskProgress('<p>No tasks</p>')).toBeNull()
  })
})
