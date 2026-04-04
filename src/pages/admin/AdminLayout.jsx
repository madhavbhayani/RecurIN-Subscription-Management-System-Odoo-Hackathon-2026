import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { clearAuthSession } from '../../services/session'

const adminMenuItems = [
  { label: 'Subscriptions', to: '/admin/subscriptions' },
  { label: 'Products', to: '/admin/products' },
  { label: 'Reporting', to: '/admin/reporting' },
  { label: 'Users', to: '/admin/users' },
  { label: 'Configurations', to: '/admin/configurations' },
]

function AdminLayout() {
  const navigate = useNavigate()

  const handleSignOut = () => {
    clearAuthSession()
    navigate('/login')
  }

  return (
    <div className="flex min-h-screen bg-[var(--light-bg)]">
      <aside className="flex w-72 flex-none flex-col justify-between bg-black px-5 py-6 text-white">
        <div>
          <p className="[font-family:var(--font-display)] text-3xl font-bold tracking-tight">
            RecurIN
          </p>
          <p className="mt-2 text-xs text-white/70">admin workspace</p>

          <nav className="mt-8 space-y-2" aria-label="Admin navigation">
            {adminMenuItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  `block rounded-lg px-4 py-3 text-sm font-semibold transition-colors duration-200 ${
                    isActive
                      ? 'bg-[var(--orange)] text-white'
                      : 'text-white/85 hover:bg-white/12 hover:text-white'
                  }`
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </div>

        <div className="border-t border-white/20 pt-4">
          <button
            type="button"
            className="block w-full rounded px-2 py-2 text-left text-xs font-medium lowercase text-white/75 transition-colors duration-200 hover:bg-white/12 hover:text-white"
          >
            account options
          </button>
          <button
            type="button"
            onClick={handleSignOut}
            className="mt-1 block w-full rounded px-2 py-2 text-left text-xs font-semibold lowercase text-white transition-colors duration-200 hover:bg-white/12"
          >
            signout
          </button>
        </div>
      </aside>

      <section className="flex-1 px-6 py-8 sm:px-8">
        <Outlet />
      </section>
    </div>
  )
}

export default AdminLayout
