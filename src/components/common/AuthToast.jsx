function AuthToast({ message, variant = 'error' }) {
  if (!message) {
    return null
  }

  const variantClasses = {
    error: 'border-red-200 bg-red-50 text-red-700',
    success: 'border-green-200 bg-green-50 text-green-700',
    info: 'border-[color:rgba(0,0,128,0.2)] bg-[color:rgba(0,0,128,0.04)] text-[var(--navy)]',
  }

  return (
    <div
      role="status"
      aria-live="polite"
      className={`mt-4 rounded-lg border px-4 py-3 text-sm font-medium ${variantClasses[variant] ?? variantClasses.info}`}
    >
      {message}
    </div>
  )
}

export default AuthToast