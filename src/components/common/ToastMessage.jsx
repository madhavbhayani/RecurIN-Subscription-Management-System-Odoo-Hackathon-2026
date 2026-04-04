import { useEffect, useRef, useState } from 'react'
import { FiX } from 'react-icons/fi'

function ToastMessage({ message, variant = 'error', duration = 3000, onClose }) {
  const [activeMessage, setActiveMessage] = useState('')
  const [activeVariant, setActiveVariant] = useState(variant)
  const [isVisible, setIsVisible] = useState(false)

  const hideTimerRef = useRef(null)
  const removeTimerRef = useRef(null)

  const clearTimers = () => {
    if (hideTimerRef.current !== null) {
      window.clearTimeout(hideTimerRef.current)
      hideTimerRef.current = null
    }
    if (removeTimerRef.current !== null) {
      window.clearTimeout(removeTimerRef.current)
      removeTimerRef.current = null
    }
  }

  const dismissToast = () => {
    clearTimers()
    setIsVisible(false)

    removeTimerRef.current = window.setTimeout(() => {
      setActiveMessage('')
      if (onClose) {
        onClose()
      }
    }, 260)
  }

  useEffect(() => {
    if (!message) {
      return undefined
    }

    clearTimers()
    setActiveMessage(message)
    setActiveVariant(variant)

    const frameId = window.requestAnimationFrame(() => {
      setIsVisible(true)
    })

    hideTimerRef.current = window.setTimeout(() => {
      dismissToast()
    }, duration)

    return () => {
      window.cancelAnimationFrame(frameId)
      clearTimers()
    }
  }, [duration, message, variant])

  useEffect(() => () => {
    clearTimers()
  }, [])

  if (!activeMessage) {
    return null
  }

  const variantClasses = {
    error: 'border-red-200 bg-red-50 text-red-700',
    success: 'border-green-200 bg-green-50 text-green-700',
    info: 'border-[color:rgba(0,0,128,0.2)] bg-[color:rgba(0,0,128,0.04)] text-[var(--navy)]',
  }

  return (
    <div className="pointer-events-none fixed right-4 top-4 z-50 w-[calc(100vw-2rem)] max-w-sm sm:right-6 sm:top-6">
      <div
        role="status"
        aria-live="polite"
        className={`pointer-events-auto rounded-lg border px-4 py-3 text-sm font-medium shadow-[0_10px_25px_rgba(0,0,0,0.12)] transition-all duration-300 ease-out ${variantClasses[activeVariant] ?? variantClasses.info} ${isVisible ? 'translate-x-0 opacity-100' : 'translate-x-8 opacity-0'}`}
      >
        <div className="flex items-start gap-3">
          <p className="flex-1 leading-relaxed">{activeMessage}</p>
          <button
            type="button"
            onClick={dismissToast}
            className="inline-flex h-6 w-6 items-center justify-center rounded text-current/70 transition-colors duration-200 hover:bg-black/10 hover:text-current"
            aria-label="Close notification"
            title="Close"
          >
            <FiX className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  )
}

export default ToastMessage