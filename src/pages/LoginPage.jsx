import { Link } from 'react-router-dom'

function LoginPage() {
  const handleLoginSubmit = (event) => {
    event.preventDefault()
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

        <form className="mt-8 space-y-5" onSubmit={handleLoginSubmit}>
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
              placeholder="Enter your password"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <button
            type="submit"
            className="w-full rounded-lg bg-[var(--orange)] px-4 py-3 text-base font-semibold text-[var(--white)] hover:bg-[#e65f00]"
          >
            Login
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