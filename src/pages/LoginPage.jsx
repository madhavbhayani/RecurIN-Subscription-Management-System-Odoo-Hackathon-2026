import { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { loginUser } from '../services/authApi'
import { saveAuthSession } from '../services/session'
import ToastMessage from '../components/common/ToastMessage'

const EMAIL_REGEX = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    const stateMessage = location.state?.toastMessage
    const stateVariant = location.state?.toastVariant

    if (stateMessage) {
      setToastVariant(stateVariant === 'success' ? 'success' : 'info')
      setToastMessage(stateMessage)
      navigate(location.pathname, { replace: true, state: null })
    }
  }, [location.pathname, location.state, navigate])

  const handleLoginSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    if (!EMAIL_REGEX.test(email.trim())) {
      setToastVariant('error')
      setToastMessage('Please enter a valid email address.')
      return
    }
    if (password.trim() === '') {
      setToastVariant('error')
      setToastMessage('Please enter your password.')
      return
    }

    setIsSubmitting(true)

    try {
      const response = await loginUser({ email, password })
      saveAuthSession(response)
      setToastVariant('success')
      setToastMessage('Login successful. Redirecting...')
      navigate('/')
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
          Login
        </h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
          Sign in to your RecurIN account.
        </p>

        <ToastMessage
          message={toastMessage}
          variant={toastVariant}
          onClose={() => setToastMessage('')}
        />

        <form className="mt-8 space-y-5" onSubmit={handleLoginSubmit} noValidate>
          <div className="space-y-2">
            <label htmlFor="email" className="block text-sm font-semibold text-[var(--navy)]">
              Email
            </label>
            <input
              id="email"
              name="email"
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
            <label htmlFor="password" className="block text-sm font-semibold text-[var(--navy)]">
              Password
            </label>
            <input
              id="password"
              name="password"
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter your password"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full rounded-lg bg-[var(--orange)] px-4 py-3 text-base font-semibold text-[var(--white)] hover:bg-[#e65f00]"
          >
            {isSubmitting ? 'Signing In...' : 'Login'}
          </button>
        </form>

        <div className="mt-6 flex flex-col gap-3 text-sm sm:flex-row sm:items-center sm:justify-between">
          <Link to="/signup" className="w-fit font-semibold text-[var(--navy)] hover:text-[var(--orange)]">
            Create Account
          </Link>
          <button type="button" className="w-fit font-semibold text-[var(--navy)] hover:text-[var(--orange)]">
            Forgot Password?
          </button>
        </div>
      </div>
    </section>
  )
}

export default LoginPage