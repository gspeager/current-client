// A number of things, with its thousands apart.
export const counted = (count: number, word: string) => `${count.toLocaleString()} ${word}${count === 1 ? '' : 's'}`
