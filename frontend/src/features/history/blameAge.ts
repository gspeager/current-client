const AGE_BANDS: { maxDays: number; intensity: number }[] = [
  { maxDays: 1, intensity: 100 },
  { maxDays: 7, intensity: 75 },
  { maxDays: 30, intensity: 50 },
  { maxDays: 365, intensity: 30 },
  { maxDays: Infinity, intensity: 15 },
]

export function ageIntensity(date: Date, now: Date = new Date()): number {
  const days = (now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24)
  for (const band of AGE_BANDS) {
    if (days <= band.maxDays) return band.intensity
  }
  return AGE_BANDS[AGE_BANDS.length - 1].intensity
}
