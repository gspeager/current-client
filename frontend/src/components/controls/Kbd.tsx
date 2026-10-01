import './Kbd.scss'

interface KbdProps {
  children: string
}

function Kbd({ children }: KbdProps) {
  return <kbd className="kbd">{children}</kbd>
}

export default Kbd
