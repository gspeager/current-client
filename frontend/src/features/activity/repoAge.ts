export function formatAge(days: number): string {
  if (days < 1) return 'Today'
  if (days < 30) return `${days}d`
  if (days < 365) return `${Math.round(days / 30)}mo`
  return `${Math.round(days / 365)}y`
}

const AGE_COLOR_BANDS: { maxDays: number; color: string }[] = [
  { maxDays: 7, color: 'var(--tertiary)' },
  { maxDays: 365, color: 'var(--on-surface)' },
  { maxDays: Infinity, color: 'var(--outline)' },
]

export function ageColor(days: number): string {
  for (const band of AGE_COLOR_BANDS) {
    if (days <= band.maxDays) return band.color
  }
  return AGE_COLOR_BANDS[AGE_COLOR_BANDS.length - 1].color
}
