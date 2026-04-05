import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import ToastMessage from '../components/common/ToastMessage'
import { resetForgotPassword } from '../services/authApi'

const RESET_TOKEN_STORAGE_KEY = 'recurin_password_reset_token'

function ResetPasswordPage() {
  const navigate = useNavigate()
  const location = useLocation()

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [hasResetToken, setHasResetToken] = useState(() => {
    const token = String(sessionStorage.getItem(RESET_TOKEN_STORAGE_KEY) ?? '').trim()
    return token !== ''
  })

  useEffect(() => {
    const stateToastMessage = String(location.state?.toastMessage ?? '').trim()
    if (!stateToastMessage) {
      return
    }

    const stateToastVariant = String(location.state?.toastVariant ?? 'info').trim()
    setToastVariant(stateToastVariant === 'success' ? 'success' : 'info')
    setToastMessage(stateToastMessage)
    navigate(location.pathname, { replace: true, state: null })
  }, [location.pathname, location.state, navigate])

  const handleResetPassword = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const resetToken = String(sessionStorage.getItem(RESET_TOKEN_STORAGE_KEY) ?? '').trim()
    if (!resetToken) {
      setHasResetToken(false)
      setToastVariant('error')
      setToastMessage('Reset session expired. Please verify OTP again.')
      return
    }

    if (newPassword.trim().length < 8) {
      setToastVariant('error')
      setToastMessage('New password must be at least 8 characters.')
      return
    }

    if (newPassword !== confirmPassword) {
      setToastVariant('error')
      setToastMessage('New password and re-enter password must match.')
      return
    }

    setIsSubmitting(true)
    try {
      await resetForgotPassword({
        reset_token: resetToken,
        new_password: newPassword,
      })

      sessionStorage.removeItem(RESET_TOKEN_STORAGE_KEY)
      navigate('/login', {
        state: {
          toastVariant: 'success',
          toastMessage: 'Password reset successfully. Please login with your new password.',
        },
      })
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <section className="mx-auto w-full max-w-lg px-4 py-10 sm:px-6 sm:py-14">
      <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_28px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)] sm:text-4xl">
          Reset Password
        </h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
          Create your new password to finish account recovery.
        </p>

        <ToastMessage
          message={toastMessage}
          variant={toastVariant}
          onClose={() => setToastMessage('')}
        />

        {!hasResetToken ? (
          <div className="mt-6 rounded-xl border border-amber-200 bg-amber-50 px-4 py-4 text-sm text-amber-800">
            Your password reset session is not available. Please verify OTP again.
            <div className="mt-3">
              <Link
                to="/verify-otp"
                className="inline-flex h-9 items-center rounded-lg border border-amber-300 px-4 text-sm font-semibold text-amber-800 transition-colors duration-200 hover:bg-amber-100"
              >
                Go to Verify OTP
              </Link>
            </div>
          </div>
        ) : (
          <form className="mt-8 space-y-5" onSubmit={handleResetPassword} noValidate>
            <div className="space-y-2">
              <label htmlFor="new-password" className="block text-sm font-semibold text-[var(--navy)]">
                Enter New Password
              </label>
              <input
                id="new-password"
                name="new-password"
                type="password"
                required
                minLength={8}
                autoComplete="new-password"
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
                placeholder="Minimum 8 characters"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="confirm-new-password" className="block text-sm font-semibold text-[var(--navy)]">
                Re-enter New Password
              </label>
              <input
                id="confirm-new-password"
                name="confirm-new-password"
                type="password"
                required
                autoComplete="new-password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                placeholder="Re-enter your new password"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>

            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full rounded-lg bg-[var(--orange)] px-4 py-3 text-base font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
            >
              {isSubmitting ? 'Resetting Password...' : 'Reset Password'}
            </button>
          </form>
        )}

        <p className="mt-6 text-sm text-[color:rgba(0,0,128,0.78)]">
          Back to{' '}
          <Link to="/login" className="font-semibold text-[var(--navy)] hover:text-[var(--orange)]">
            Login
          </Link>
        </p>
      </div>
    </section>
  )
}

export default ResetPasswordPage
