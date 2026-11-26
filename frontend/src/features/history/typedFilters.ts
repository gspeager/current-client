import { BREAKING_FILTER } from '../../lib/conventionalCommit'
import { localDate } from '../../lib/relativeTime'

export const FILTER_KEYS = ['author', 'type', 'since', 'until'] as const
export type FilterKey = (typeof FILTER_KEYS)[number]

interface Token {
  text: string
  key?: FilterKey
  value?: string
}

// key:value, key:"a value" or key:'a value', or any other word. An unknown key is a word.
const TOKEN = /([A-Za-z]+):(?:"([^"]*)"?|'([^']*)'?|(\S*))|\S+/g

function tokens(box: string): Token[] {
  return [...box.matchAll(TOKEN)].map((match) => {
    const key = match[1]?.toLowerCase() as FilterKey | undefined
    if (!key || !FILTER_KEYS.includes(key)) return { text: match[0] }
    return { text: match[0], key, value: match[2] ?? match[3] ?? match[4] ?? '' }
  })
}

// What's typed for each key, as typed; the last of a key wins.
export function typedValues(box: string): Record<FilterKey, string> {
  const values = { author: '', type: '', since: '', until: '' }
  for (const token of tokens(box)) if (token.key) values[token.key] = token.value ?? ''
  return values
}

// What's left to search for.
export function typedText(box: string): string {
  return tokens(box)
    .filter((token) => !token.key)
    .map((token) => token.text)
    .join(' ')
}

// The history filter the box sets. A bare date gets a time, since Git's --since
// with one alone can miss that day (core/git's sinceMidnight).
export function typedFilters(box: string, now: Date = new Date()) {
  const { author, type, since, until } = typedValues(box)
  return {
    author,
    type: type === 'breaking' || type === BREAKING_FILTER ? BREAKING_FILTER : type.toLowerCase(),
    since: dateArg(since, '00:00:00', now),
    until: dateArg(until, '23:59:59', now),
  }
}

function dateArg(value: string, time: string, now: Date): string {
  const day = dayOf(value, now)
  return day ? `${day} ${time}` : value
}

function dayOf(value: string, now: Date): string | null {
  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) return value
  const days = value.toLowerCase() === 'today' ? 0 : /^(\d+)d$/i.exec(value)?.[1]
  if (days === undefined) return null
  const day = new Date(now)
  day.setDate(day.getDate() - Number(days))
  return localDate(day)
}

// The box with key set to value, or taken out when value is empty, the rest as typed.
export function withTyped(box: string, key: FilterKey, value: string): string {
  const kept = tokens(box)
    .filter((token) => token.key !== key)
    .map((token) => token.text)
  if (value) kept.push(`${key}:${/[\s"']/.test(value) ? `"${value.split('"').join('')}"` : value}`)
  return kept.join(' ')
}

export function describeSince(value: string): string {
  if (value.toLowerCase() === 'today') return 'today'
  const days = /^(\d+)d$/i.exec(value)?.[1]
  return days ? `in the past ${days} days` : `since ${value}`
}
