import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import ToastMessage from '../components/common/ToastMessage'
import { requestForgotPasswordOTP, verifyForgotPasswordOTP } from '../services/authApi'

const EMAIL_REGEX = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/
const OTP_REGEX = /^[0-9]{6}$/
const RESET_TOKEN_STORAGE_KEY = 'recurin_password_reset_token'

function VerifyOtpPage() {
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState(() => String(location.state?.email ?? '').trim().toLowerCase())
  const [otp, setOtp] = useState('')
  const [isVerifying, setIsVerifying] = useState(false)
  const [isResending, setIsResending] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')

  useEffect(() => {
    const stateToastMessage = String(location.state?.toastMessage ?? '').trim()
    if (!stateToastMessage) {
      return
    }

    const stateToastVariant = String(location.state?.toastVariant ?? 'info').trim()
    setToastVariant(stateToastVariant === 'success' ? 'success' : 'info')
    setToastMessage(stateToastMessage)
    navigate(location.pathname, { replace: true, state: { email } })
  }, [email, location.pathname, location.state, navigate])

  const handleOTPChange = (event) => {
    const nextOTP = event.target.value.replace(/\D/g, '').slice(0, 6)
    setOtp(nextOTP)
  }

  const handleVerifyOTP = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const normalizedEmail = email.trim().toLowerCase()
    if (!EMAIL_REGEX.test(normalizedEmail)) {
      setToastVariant('error')
      setToastMessage('Please enter a valid email address.')
      return
    }

    if (!OTP_REGEX.test(otp)) {
      setToastVariant('error')
      setToastMessage('Please enter the 6-digit OTP.')
      return
    }

    setIsVerifying(true)
    try {
      const response = await verifyForgotPasswordOTP({
        email: normalizedEmail,
        otp,
      })

      const resetToken = String(response?.reset_token ?? '').trim()
      if (!resetToken) {
        throw new Error('OTP verification did not return a reset token. Please try again.')
      }

      sessionStorage.setItem(RESET_TOKEN_STORAGE_KEY, resetToken)
      navigate('/reset-password', {
        state: {
          email: normalizedEmail,
          toastVariant: 'success',
          toastMessage: 'OTP verified. You can now reset your password.',
        },
      })
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsVerifying(false)
    }
  }

  const handleResendOTP = async () => {
    setToastMessage('')
    const normalizedEmail = email.trim().toLowerCase()

    if (!EMAIL_REGEX.test(normalizedEmail)) {
      setToastVariant('error')
      setToastMessage('Enter a valid email first to resend OTP.')
      return
    }

    setIsResending(true)
    try {
      const response = await requestForgotPasswordOTP({ email: normalizedEmail })
      setToastVariant('success')
      setToastMessage(String(response?.message ?? 'A new OTP has been sent.').trim())
      setOtp('')
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsResending(false)
    }
  }

  return (
    <section className="mx-auto w-full max-w-lg px-4 py-10 sm:px-6 sm:py-14">
      <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_28px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)] sm:text-4xl">
          Verify OTP
        </h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
          Enter the 6-digit OTP sent to your email to continue password reset.
        </p>

        <ToastMessage
          message={toastMessage}
          variant={toastVariant}
          onClose={() => setToastMessage('')}
        />

        <form className="mt-8 space-y-5" onSubmit={handleVerifyOTP} noValidate>
          <div className="space-y-2">
            <label htmlFor="verify-email" className="block text-sm font-semibold text-[var(--navy)]">
              Email
            </label>
            <input
              id="verify-email"
              name="verify-email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="verify-otp" className="block text-sm font-semibold text-[var(--navy)]">
              6-Digit OTP
            </label>
            <input
              id="verify-otp"
              name="verify-otp"
              type="text"
              required
              inputMode="numeric"
              maxLength={6}
              value={otp}
              onChange={handleOTPChange}
              placeholder="Enter OTP"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base tracking-[0.2em] text-[var(--navy)] outline-none placeholder:tracking-normal placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <button
            type="submit"
            disabled={isVerifying}
            className="w-full rounded-lg bg-[var(--orange)] px-4 py-3 text-base font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            {isVerifying ? 'Verifying OTP...' : 'Verify OTP'}
          </button>
        </form>

        <div className="mt-6 flex flex-col gap-3 text-sm sm:flex-row sm:items-center sm:justify-between">
          <button
            type="button"
            onClick={handleResendOTP}
            disabled={isResending}
            className="w-fit font-semibold text-[var(--navy)] hover:text-[var(--orange)]"
          >
            {isResending ? 'Resending OTP...' : 'Resend OTP'}
          </button>
          <Link to="/forgot-password" className="w-fit font-semibold text-[var(--navy)] hover:text-[var(--orange)]">
            Change Email
          </Link>
        </div>
      </div>
    </section>
  )
}

export default VerifyOtpPage
