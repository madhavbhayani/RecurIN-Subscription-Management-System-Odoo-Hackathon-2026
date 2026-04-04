import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { countryCodeOptions } from '../constants/countryCodes'
import { signupUser } from '../services/authApi'
import ToastMessage from '../components/common/ToastMessage'

const DOMAIN_EMAIL_REGEX = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.com$/
const TEN_DIGIT_PHONE_REGEX = /^\d{10}$/

function SignupPage() {
  const navigate = useNavigate()

  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [phoneNumber, setPhoneNumber] = useState('')
  const [countryCode, setCountryCode] = useState('+91')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handlePhoneChange = (event) => {
    const digitsOnly = event.target.value.replace(/\D/g, '').slice(0, 10)
    setPhoneNumber(digitsOnly)
  }

  const handleSignupSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    if (name.trim() === '') {
      setToastVariant('error')
      setToastMessage('Please enter your name.')
      return
    }
    if (!DOMAIN_EMAIL_REGEX.test(email.trim())) {
      setToastVariant('error')
      setToastMessage('Please enter an email in this format: name@domain.com.')
      return
    }
    if (password.trim().length < 8) {
      setToastVariant('error')
      setToastMessage('Password must be at least 8 characters.')
      return
    }
    if (password !== confirmPassword) {
      setToastVariant('error')
      setToastMessage('Password and re-enter password must match.')
      return
    }
    if (!TEN_DIGIT_PHONE_REGEX.test(phoneNumber)) {
      setToastVariant('error')
      setToastMessage('Phone number must be exactly 10 digits.')
      return
    }

    setIsSubmitting(true)

    try {
      await signupUser({
        name,
        email,
        password,
        country_code: countryCode,
        phone_number: phoneNumber,
      })

      setToastVariant('success')
      setToastMessage('Account created successfully. Please log in to continue.')
      navigate('/login', {
        state: {
          toastVariant: 'success',
          toastMessage: 'Account created successfully. Please log in to continue.',
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
          Create Account
        </h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
          Join RecurIN to manage all subscriptions in one place.
        </p>

        <ToastMessage
          message={toastMessage}
          variant={toastVariant}
          onClose={() => setToastMessage('')}
        />

        <form className="mt-8 space-y-5" onSubmit={handleSignupSubmit} noValidate>
          <div className="space-y-2">
            <label htmlFor="full-name" className="block text-sm font-semibold text-[var(--navy)]">
              Name
            </label>
            <input
              id="full-name"
              name="full-name"
              type="text"
              required
              autoComplete="name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Enter your full name"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="signup-email" className="block text-sm font-semibold text-[var(--navy)]">
              Email
            </label>
            <input
              id="signup-email"
              name="signup-email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="name@domain.com"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="signup-password" className="block text-sm font-semibold text-[var(--navy)]">
              Password
            </label>
            <input
              id="signup-password"
              name="signup-password"
              type="password"
              required
              autoComplete="new-password"
              minLength={8}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Minimum 8 characters"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="signup-confirm-password" className="block text-sm font-semibold text-[var(--navy)]">
              Re-enter Password
            </label>
            <input
              id="signup-confirm-password"
              name="signup-confirm-password"
              type="password"
              required
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              placeholder="Re-enter your password"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="phone-number" className="block text-sm font-semibold text-[var(--navy)]">
              Phone Number
            </label>

            <div className="flex gap-2">
              <select
                id="country-code"
                name="country-code"
                value={countryCode}
                onChange={(event) => setCountryCode(event.target.value)}
                className="w-32 rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-3 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              >
                {countryCodeOptions.map((option) => (
                  <option key={option.code} value={option.code}>
                    {option.label}
                  </option>
                ))}
              </select>

              <input
                id="phone-number"
                name="phone-number"
                type="tel"
                required
                inputMode="numeric"
                autoComplete="tel-national"
                value={phoneNumber}
                onChange={handlePhoneChange}
                maxLength={10}
                placeholder="10-digit number"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full rounded-lg bg-[var(--orange)] px-4 py-3 text-base font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            {isSubmitting ? 'Creating Account...' : 'Create Account'}
          </button>
        </form>

        <p className="mt-6 text-sm text-[color:rgba(0,0,128,0.78)]">
          Already have an account?{' '}
          <Link to="/login" className="font-semibold text-[var(--navy)] hover:text-[var(--orange)]">
            Login
          </Link>
        </p>
      </div>
    </section>
  )
}

export default SignupPage