import { useState, useEffect } from 'react'

const MOBILE_BREAKPOINT = 640
const TABLET_BREAKPOINT = 1024

export function useMobile() {
  const [isMobile, setIsMobile] = useState(false)
  const [isTablet, setIsTablet] = useState(false)
  const [isDesktop, setIsDesktop] = useState(true)

  useEffect(() => {
    const checkViewport = () => {
      const width = window.innerWidth
      setIsMobile(width < MOBILE_BREAKPOINT)
      setIsTablet(width >= MOBILE_BREAKPOINT && width < TABLET_BREAKPOINT)
      setIsDesktop(width >= TABLET_BREAKPOINT)
    }

    // Check on mount
    checkViewport()

    // Listen for resize
    window.addEventListener('resize', checkViewport)
    return () => window.removeEventListener('resize', checkViewport)
  }, [])

  return { isMobile, isTablet, isDesktop }
}
