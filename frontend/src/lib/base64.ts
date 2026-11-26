export function decodeBase64(data: string): Uint8Array {
  return Uint8Array.from(atob(data), (char) => char.charCodeAt(0))
}

export function encodeBase64(bytes: Uint8Array): string {
  let text = ''
  // In chunks, since each byte becomes an argument and a large array would overflow the call.
  for (let i = 0; i < bytes.length; i += 0x8000) text += String.fromCharCode(...bytes.subarray(i, i + 0x8000))
  return btoa(text)
}
