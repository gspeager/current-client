export interface SignIn {
  username: string
  password: string
}

export interface SignInOptions {
  // The previous attempt was turned down, rather than none having been made.
  rejected: boolean
}

// The messages core/gitexec gives when Git had no credentials, or the remote
// refused them.
const SIGN_IN_MESSAGES = new Set([
  'This remote needs credentials. Set up a credential helper, such as Git Credential Manager, for it.',
  'Authentication failed. Check the credentials Git uses for this remote.',
])

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

export function needsSignIn(err: unknown): boolean {
  return SIGN_IN_MESSAGES.has(messageOf(err))
}

// Runs start without credentials, and while the remote asks for them, asks the
// user and retries with what they enter. Cancelling the form rethrows the
// last error.
export async function withSignIn<T>(
  start: (auth: SignIn | null) => Promise<T>,
  signIn: (options: SignInOptions) => Promise<SignIn | null>,
): Promise<T> {
  let auth: SignIn | null = null
  for (;;) {
    try {
      return await start(auth)
    } catch (err: unknown) {
      if (!needsSignIn(err)) throw err
      const entered = await signIn({ rejected: auth !== null })
      if (!entered) throw err
      auth = entered
    }
  }
}
