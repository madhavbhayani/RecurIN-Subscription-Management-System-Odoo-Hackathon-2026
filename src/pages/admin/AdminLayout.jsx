import { useEffect, useState } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import {
  FiArrowLeft,
  FiBox,
  FiChevronDown,
  FiChevronRight,
  FiClipboard,
  FiKey,
  FiLayers,
  FiLogOut,
  FiSettings,
  FiShield,
  FiUsers,
} from 'react-icons/fi'
import RecurInLogo from '../../components/common/RecurInLogo'
import { clearAuthSession, getAuthSession } from '../../services/session'

const baseAdminMenuItems = [
  { label: 'Subscriptions', to: '/admin/subscriptions', icon: FiLayers },
  { label: 'Products', to: '/admin/products', icon: FiBox },
  { label: 'Reporting', to: '/admin/reporting', icon: FiClipboard },
  { label: 'Users', to: '/admin/users', icon: FiUsers },
  { label: 'Roles', to: '/admin/roles', icon: FiKey },
]

const configurationMenuItems = [
  { label: 'Attribute', to: '/admin/configurations/attribute' },
  { label: 'Recurring Plan', to: '/admin/configurations/recurring-plan' },
  { label: 'Quotation Template', to: '/admin/configurations/quotation-template' },
  { label: 'Payment Term', to: '/admin/configurations/payment-term' },
  { label: 'Discount', to: '/admin/configurations/discount' },
  { label: 'Taxes', to: '/admin/configurations/taxes' },
]

function AdminLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const session = getAuthSession()
  const [isConfigurationsOpen, setIsConfigurationsOpen] = useState(false)

  const normalizedRole = String(session?.user?.role ?? '').trim().toLowerCase()
  const workspaceLabel = normalizedRole === 'internal' || normalizedRole === 'internal-user'
    ? 'Internal Workspace'
    : 'Admin Workspace'

  const isConfigurationsRoute = location.pathname.startsWith('/admin/configurations')

  useEffect(() => {
    if (isConfigurationsRoute) {
      setIsConfigurationsOpen(true)
    }
  }, [isConfigurationsRoute])

  const handleSignOut = () => {
    clearAuthSession()
    navigate('/login')
  }

  const handleBackToSite = () => {
    navigate('/home')
  }

  return (
    <div className="admin-workspace flex min-h-screen bg-[var(--light-bg)]">
      <aside className="flex w-56 flex-none flex-col justify-between border-r border-[#2a2a2f] bg-[#0d0d11] px-3 py-4 text-white shadow-[8px_0_24px_rgba(0,0,0,0.35)]">
        <div>
          <div className="inline-flex rounded-md border border-[#2f2f35] bg-white px-2 py-1.5">
            <RecurInLogo compact />
          </div>

          <p className="mt-2 inline-flex items-center gap-1.5 rounded border border-white/20 bg-white/6 px-2 py-1 text-[11px] font-semibold text-white/95">
            <FiShield className="h-3 w-3" />
            {workspaceLabel}
          </p>

          <nav className="mt-5 space-y-1" aria-label="Admin navigation">
            {baseAdminMenuItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                title={item.label}
                className={({ isActive }) =>
                  `flex items-center gap-2 rounded-md border px-3 py-2 text-[12px] font-semibold transition-colors duration-200 ${
                    isActive
                      ? 'border-[color:rgba(255,255,255,0.45)] bg-[var(--orange)] text-white'
                      : 'border-transparent bg-white/3 text-white hover:border-white/30 hover:bg-white/12 hover:text-white'
                  }`
                }
              >
                <item.icon className="h-3 w-3 flex-none" />
                {item.label}
              </NavLink>
            ))}

            <button
              type="button"
              onClick={() => setIsConfigurationsOpen((previous) => !previous)}
              className={`flex w-full items-center gap-2 rounded-md border px-3 py-2 text-left text-[12px] font-semibold transition-colors duration-200 ${
                isConfigurationsRoute
                  ? 'border-[color:rgba(255,255,255,0.45)] bg-[var(--orange)] text-white'
                  : 'border-transparent bg-white/3 text-white hover:border-white/30 hover:bg-white/12 hover:text-white'
              }`}
              aria-expanded={isConfigurationsOpen}
              aria-controls="configurations-submenu"
            >
              <FiSettings className="h-3 w-3 flex-none" />
              <span className="flex-1">Configurations</span>
              {isConfigurationsOpen ? (
                <FiChevronDown className="h-3 w-3 flex-none" />
              ) : (
                <FiChevronRight className="h-3 w-3 flex-none" />
              )}
            </button>

            {isConfigurationsOpen && (
              <div id="configurations-submenu" className="space-y-1 border-l border-white/20 pl-4">
                {configurationMenuItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) =>
                      `block rounded px-2.5 py-1 text-[10.5px] font-semibold transition-colors duration-200 ${
                        isActive
                          ? 'bg-white/18 text-white'
                          : 'text-white/90 hover:bg-white/12 hover:text-white'
                      }`
                    }
                  >
                    {item.label}
                  </NavLink>
                ))}
              </div>
            )}
          </nav>
        </div>

        <div className="border-t border-white/20 pt-3">
          <button
            type="button"
            onClick={handleBackToSite}
            className="inline-flex w-full items-center gap-1.5 rounded px-2 py-1.5 text-left text-[11px] font-medium text-white transition-colors duration-200 hover:bg-white/12"
          >
            <FiArrowLeft className="h-3 w-3" />
            Back to site
          </button>
          <button
            type="button"
            onClick={handleSignOut}
            className="mt-1 inline-flex w-full items-center gap-1.5 rounded px-2.5 py-1.5 text-left text-[11px] font-semibold text-red-500 transition-colors duration-200 hover:bg-white/10 hover:text-red-400"
          >
            <FiLogOut className="h-3 w-3" />
            Signout
          </button>
        </div>
      </aside>

      <section className="admin-content min-w-0 flex-1 px-3 py-4 sm:px-4 sm:py-5">
        <Outlet />
      </section>
    </div>
  )
}

export default AdminLayout
