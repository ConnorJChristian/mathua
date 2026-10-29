'use client'

import { Toaster as Sonner, toast } from 'sonner'
import 'sonner/dist/styles.css'
import { useTheme } from '@/hooks/useTheme'
import { DEFAULT_THEME } from '@/lib/theme'

export { toast }

type ToasterProps = React.ComponentProps<typeof Sonner>

export function Toaster({ ...props }: ToasterProps) {
  // Follow Mathua's own theme, not the OS: the page defaults to light whatever
  // the system prefers, and a toast must match the page it lands on.
  // Until mounted the server and client must agree, so the default is used.
  const { theme, mounted } = useTheme()
  return (
    <Sonner
      theme={mounted ? theme : DEFAULT_THEME}
      position="bottom-right"
      closeButton
      {...props}
    />
  )
}
