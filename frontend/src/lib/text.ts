export function capitalize(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1)
}

export function shortSha(sha: string): string {
  return sha.slice(0, 7)
}
