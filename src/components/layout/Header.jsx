import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { navItems } from '../../constants/site'
import { clearAuthSession, getAuthSession } from '../../services/session'
import RecurInLogo from '../common/RecurInLogo'

const baseAccountItems = [
  { label: 'Account Options' },
  { label: 'Subscriptions' },
  { label: 'Cart', to: '/cart' },
]

function isAdminOrInternalRole(role) {
  const normalizedRole = String(role ?? '').trim().toLowerCase()
  return normalizedRole === 'admin' || normalizedRole === 'internal' || normalizedRole === 'internal-user'
}

function Header() {
  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const [isAccountMenuOpen, setIsAccountMenuOpen] = useState(false)
  const [session, setSession] = useState(() => getAuthSession())

  const accountMenuRef = useRef(null)
  const location = useLocation()
  const navigate = useNavigate()

  const desktopNavLinkClass =
    "relative inline-flex items-center px-3 py-1.5 text-[15px] font-semibold text-[var(--navy)] transition-colors duration-300 hover:text-[var(--orange)] after:absolute after:bottom-0 after:left-3 after:h-0.5 after:w-[calc(100%-1.5rem)] after:origin-left after:scale-x-0 after:bg-[var(--orange)] after:content-[''] after:transition-transform after:duration-300 after:ease-out hover:after:scale-x-100"

  const mobileNavLinkClass =
    'rounded px-3 py-1.5 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:text-[var(--orange)]'

  const toggleMenu = () => {
    setIsMenuOpen((previous) => !previous)
  }

  const toggleAccountMenu = () => {
    setIsAccountMenuOpen((previous) => !previous)
  }

  const closeMenu = () => {
    setIsMenuOpen(false)
  }

  const closeAccountMenu = () => {
    setIsAccountMenuOpen(false)
  }

  const handleSignOut = () => {
    clearAuthSession()
    setSession(null)
    setIsAccountMenuOpen(false)
    setIsMenuOpen(false)
    navigate('/login')
  }

  useEffect(() => {
    setSession(getAuthSession())
    setIsAccountMenuOpen(false)
    setIsMenuOpen(false)
  }, [location.pathname])

  useEffect(() => {
    if (!isAccountMenuOpen) {
      return undefined
    }

    const handleOutsideClick = (event) => {
      if (accountMenuRef.current && !accountMenuRef.current.contains(event.target)) {
        setIsAccountMenuOpen(false)
      }
    }

    document.addEventListener('mousedown', handleOutsideClick)

    return () => {
      document.removeEventListener('mousedown', handleOutsideClick)
    }
  }, [isAccountMenuOpen])

  const currentRole = String(session?.user?.role ?? '').toLowerCase()
  const hasActiveSession = Boolean(session?.token)
  const accountItems = isAdminOrInternalRole(currentRole)
    ? [...baseAccountItems, { label: 'Admin Panel', to: '/admin/subscriptions' }]
    : baseAccountItems

  const renderAccountItem = (item, closeHandler) => {
    if (item.to) {
      return (
        <Link
          key={item.label}
          to={item.to}
          onClick={closeHandler}
          className="block w-full px-4 py-2 text-left text-sm font-medium text-[var(--navy)] transition-colors duration-200 hover:bg-[color:rgba(255,107,0,0.08)]"
        >
          {item.label}
        </Link>
      )
    }

    return (
      <button
        key={item.label}
        type="button"
        onClick={closeHandler}
        className="block w-full px-4 py-2 text-left text-sm font-medium text-[var(--navy)] transition-colors duration-200 hover:bg-[color:rgba(255,107,0,0.08)]"
      >
        {item.label}
      </button>
    )
  }

  const renderDesktopAccountArea = () => {
    if (!hasActiveSession) {
      return (
        <Link
          to="/login"
          className="rounded-md bg-[var(--orange)] px-4 py-1.5 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
        >
          Login
        </Link>
      )
    }

    return (
      <div className="relative" ref={accountMenuRef}>
        <button
          type="button"
          onClick={toggleAccountMenu}
          className="inline-flex items-center gap-2 rounded-md bg-[var(--orange)] px-4 py-1.5 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          aria-haspopup="menu"
          aria-expanded={isAccountMenuOpen}
        >
          My Account
          <span className="text-sm">▾</span>
        </button>

        {isAccountMenuOpen && (
          <div
            role="menu"
            className="absolute right-0 z-20 mt-2 w-56 rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] py-2 shadow-[0_10px_30px_rgba(0,0,128,0.12)]"
          >
            {accountItems.map((item) => renderAccountItem(item, closeAccountMenu))}

            <div className="my-1 h-px bg-[color:rgba(0,0,128,0.14)]"></div>

            <button
              type="button"
              onClick={handleSignOut}
              className="block w-full px-4 py-2 text-left text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[color:rgba(255,107,0,0.08)]"
            >
              Sign out
            </button>
          </div>
        )}
      </div>
    )
  }

  const renderMobileAuthArea = () => {
    if (!hasActiveSession) {
      return (
        <Link
          to="/login"
          onClick={closeMenu}
          className="mt-2 inline-flex w-fit rounded-md bg-[var(--orange)] px-4 py-1.5 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
        >
          Login
        </Link>
      )
    }

    return (
      <div className="mt-2 rounded-lg border border-[color:rgba(0,0,128,0.14)] p-2">
        <p className="px-2 py-2 text-base font-semibold text-[var(--navy)]">My Account</p>
        {accountItems.map((item) => {
          if (item.to) {
            return (
              <Link
                key={item.label}
                to={item.to}
                onClick={closeMenu}
                className="block w-full rounded px-2 py-2 text-left text-sm font-medium text-[var(--navy)] transition-colors duration-200 hover:bg-[color:rgba(255,107,0,0.08)]"
              >
                {item.label}
              </Link>
            )
          }

          return (
            <button
              key={item.label}
              type="button"
              onClick={closeMenu}
              className="block w-full rounded px-2 py-2 text-left text-sm font-medium text-[var(--navy)] transition-colors duration-200 hover:bg-[color:rgba(255,107,0,0.08)]"
            >
              {item.label}
            </button>
          )
        })}
        <button
          type="button"
          onClick={handleSignOut}
          className="mt-1 block w-full rounded px-2 py-2 text-left text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[color:rgba(255,107,0,0.08)]"
        >
          Sign out
        </button>
      </div>
    )
  }

  return (
    <header className="border-b border-[color:rgba(0,0,128,0.14)] bg-[var(--white)]">
      <div className="w-full px-2 sm:px-3 lg:px-4">
        <div className="flex items-center justify-between py-4 md:hidden">
          <Link to="/" onClick={closeMenu} aria-label="RecurIN home">
            <RecurInLogo compact taglineClassName="hidden" />
          </Link>

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
          <Link to="/" onClick={closeMenu} className="justify-self-start" aria-label="RecurIN home">
            <RecurInLogo compact />
          </Link>

          <nav className="flex items-center justify-center gap-0.5" aria-label="Main navigation">
            {navItems.map((item) => (
              <Link key={item.label} to={item.to} className={desktopNavLinkClass}>
                {item.label}
              </Link>
            ))}
          </nav>

          <div className="flex justify-end">
            {renderDesktopAccountArea()}
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

          {renderMobileAuthArea()}
        </nav>
      </div>
    </header>
  )
}

export default Header