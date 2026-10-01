// NUL can't appear in a git name or email.
export function identityKey(name: string, email: string): string {
  return `${name}\x00${email}`
}
