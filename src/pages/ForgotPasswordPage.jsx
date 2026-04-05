import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import ToastMessage from '../components/common/ToastMessage'
import { requestForgotPasswordOTP } from '../services/authApi'

const EMAIL_REGEX = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/

function ForgotPasswordPage() {
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')

  const handleSendOTP = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const normalizedEmail = email.trim().toLowerCase()
    if (!EMAIL_REGEX.test(normalizedEmail)) {
      setToastVariant('error')
      setToastMessage('Please enter a valid email address.')
      return
    }

    setIsSubmitting(true)
    try {
      const response = await requestForgotPasswordOTP({ email: normalizedEmail })
      const message = String(response?.message ?? 'If this email is registered, a 6-digit OTP has been sent.').trim()

      navigate('/verify-otp', {
        state: {
          email: normalizedEmail,
          toastVariant: 'success',
          toastMessage: message,
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
          Forgot Password
        </h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
          Enter your registered email address. We will send you a 6-digit OTP.
        </p>

        <ToastMessage
          message={toastMessage}
          variant={toastVariant}
          onClose={() => setToastMessage('')}
        />

        <form className="mt-8 space-y-5" onSubmit={handleSendOTP} noValidate>
          <div className="space-y-2">
            <label htmlFor="forgot-email" className="block text-sm font-semibold text-[var(--navy)]">
              Email
            </label>
            <input
              id="forgot-email"
              name="forgot-email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full rounded-lg bg-[var(--orange)] px-4 py-3 text-base font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            {isSubmitting ? 'Sending OTP...' : 'Send OTP'}
          </button>
        </form>

        <p className="mt-6 text-sm text-[color:rgba(0,0,128,0.78)]">
          Remembered your password?{' '}
          <Link to="/login" className="font-semibold text-[var(--navy)] hover:text-[var(--orange)]">
            Login
          </Link>
        </p>
      </div>
    </section>
  )
}

export default ForgotPasswordPage
