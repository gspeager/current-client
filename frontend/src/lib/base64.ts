export function decodeBase64(data: string): Uint8Array {
  return Uint8Array.from(atob(data), (char) => char.charCodeAt(0))
}
