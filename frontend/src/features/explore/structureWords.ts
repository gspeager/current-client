// A count with thousands separators and its word, plural unless the count is 1.
export const counted = (count: number, word: string) => `${count.toLocaleString()} ${word}${count === 1 ? '' : 's'}`
