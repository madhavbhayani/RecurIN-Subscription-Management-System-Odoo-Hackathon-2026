import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import Header from './components/layout/Header'
import Footer from './components/layout/Footer'
import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'
import AdminLayout from './pages/admin/AdminLayout'
import AdminSubscriptionsPage from './pages/admin/AdminSubscriptionsPage'
import AdminProductsPage from './pages/admin/AdminProductsPage'
import AdminReportingPage from './pages/admin/AdminReportingPage'
import AdminUsersPage from './pages/admin/AdminUsersPage'
import AdminConfigurationsPage from './pages/admin/AdminConfigurationsPage'
import { getAuthSession } from './services/session'

function canAccessAdminPage() {
  const session = getAuthSession()
  const role = String(session?.user?.role ?? '').trim().toLowerCase()

  return Boolean(session?.token) && (role === 'admin' || role === 'internal' || role === 'internal-user')
}

function AdminRouteGuard({ children }) {
  return canAccessAdminPage() ? children : <Navigate to="/" replace />
}

function App() {
  const location = useLocation()
  const isAdminArea = location.pathname.startsWith('/admin')

  if (isAdminArea) {
    return (
      <Routes>
        <Route
          path="/admin/*"
          element={(
            <AdminRouteGuard>
              <AdminLayout />
            </AdminRouteGuard>
          )}
        >
          <Route index element={<Navigate to="subscriptions" replace />} />
          <Route path="subscriptions" element={<AdminSubscriptionsPage />} />
          <Route path="products" element={<AdminProductsPage />} />
          <Route path="reporting" element={<AdminReportingPage />} />
          <Route path="users" element={<AdminUsersPage />} />
          <Route path="configurations" element={<AdminConfigurationsPage />} />
          <Route path="*" element={<Navigate to="subscriptions" replace />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    )
  }

  return (
    <div className="flex min-h-screen flex-col bg-[var(--light-bg)]">
      <Header />

      <main className="flex-1">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </main>

      <Footer />
    </div>
  )
}

export default App
