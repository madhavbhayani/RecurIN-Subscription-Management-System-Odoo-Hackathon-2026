import { useState } from 'react'
import { Link } from 'react-router-dom'
import { navItems } from '../../constants/site'

function Header() {
  const [isMenuOpen, setIsMenuOpen] = useState(false)

  const desktopNavLinkClass =
    "relative inline-flex items-center px-4 py-2 text-lg font-semibold text-[var(--navy)] transition-colors duration-300 hover:text-[var(--orange)] after:absolute after:bottom-1 after:left-4 after:h-0.5 after:w-[calc(100%-2rem)] after:origin-left after:scale-x-0 after:bg-[var(--orange)] after:content-[''] after:transition-transform after:duration-300 after:ease-out hover:after:scale-x-100"

  const mobileNavLinkClass =
    'rounded px-3 py-2 text-base font-semibold text-[var(--navy)] transition-colors duration-300 hover:text-[var(--orange)]'

  const toggleMenu = () => {
    setIsMenuOpen((previous) => !previous)
  }

  const closeMenu = () => {
    setIsMenuOpen(false)
  }

  return (
    <header className="border-b border-[color:rgba(0,0,128,0.14)] bg-[var(--white)]">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between py-4 md:hidden">
          <Link
            to="/"
            onClick={closeMenu}
            className="[font-family:var(--font-display)] text-4xl font-bold tracking-tight text-[var(--navy)]"
          >
            <span className="text-[var(--orange)]">Recur</span>IN
          </Link>

          <nav className="hidden items-center gap-2 md:flex" aria-label="Main navigation">
            {navItems.map((item) => (
              <Link
                key={item.label}
                to={item.to}
                className="rounded-md px-4 py-2 text-lg font-semibold text-[var(--navy)] hover:bg-[color:rgba(255,107,0,0.08)] hover:text-[var(--orange)]"
              >
                {item.label}
              </Link>
            ))}

            <Link
              to="/login"
              className="rounded-md bg-[var(--orange)] px-5 py-2 text-lg font-semibold text-[var(--white)] hover:bg-[#e65f00]"
            >
              Login
            </Link>
          </nav>

          <button
            type="button"
            onClick={toggleMenu}
            className="rounded border border-[color:rgba(0,0,128,0.18)] px-3 py-2 text-sm font-semibold text-[var(--navy)] md:hidden"
            aria-controls="mobile-navigation"
            aria-expanded={isMenuOpen}
          >
            Menu
          </button>
        </div>

        <div className="hidden items-center py-5 md:grid md:grid-cols-[1fr_auto_1fr]">
          <Link
            to="/"
            onClick={closeMenu}
            className="justify-self-start [font-family:var(--font-display)] text-4xl font-bold tracking-tight text-[var(--navy)]"
          >
            <span className="text-[var(--orange)]">Recur</span>IN
          </Link>

          <nav className="flex items-center justify-center gap-1" aria-label="Main navigation">
            {navItems.map((item) => (
              <Link key={item.label} to={item.to} className={desktopNavLinkClass}>
                {item.label}
              </Link>
            ))}
          </nav>

          <div className="flex justify-end">
            <Link
              to="/login"
              className="rounded-md bg-[var(--orange)] px-5 py-2 text-lg font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
            >
              Login
            </Link>
          </div>
        </div>

        <nav
          id="mobile-navigation"
          className={`${isMenuOpen ? 'flex' : 'hidden'} flex-col gap-1 pb-4 md:hidden`}
          aria-label="Mobile navigation"
        >
          {navItems.map((item) => (
            <Link
              key={item.label}
              to={item.to}
              onClick={closeMenu}
              className={mobileNavLinkClass}
            >
              {item.label}
            </Link>
          ))}

          <Link
            to="/login"
            onClick={closeMenu}
            className="mt-2 inline-flex w-fit rounded-md bg-[var(--orange)] px-4 py-2 text-base font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            Login
          </Link>
        </nav>
      </div>
    </header>
  )
}

export default Header