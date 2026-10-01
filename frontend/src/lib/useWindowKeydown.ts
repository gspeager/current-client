import { useEventListener } from 'usehooks-ts'
import { useCurrentClientRoot } from './currentClientRoot'

// Window-wide shortcuts pause while a host app shows another view, so they
// never act on keys meant for it.
export function useWindowKeydown(handler: (e: KeyboardEvent) => void) {
  const { active } = useCurrentClientRoot()
  useEventListener('keydown', (e) => {
    if (active) handler(e)
  })
}
