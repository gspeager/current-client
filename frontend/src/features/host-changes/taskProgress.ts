interface TaskProgress {
  done: number
  total: number
}

// Rendered Markdown's task list: one checkbox per item, checked when done.
export function taskProgress(html: string): TaskProgress | null {
  const boxes = html.match(/<input\b[^>]*\btype="checkbox"[^>]*>/gi) ?? []
  if (boxes.length === 0) return null
  return { done: boxes.filter((box) => /\bchecked\b/i.test(box)).length, total: boxes.length }
}
