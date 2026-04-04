import { Navigate, Route, Routes, useLocation } from 'react-router-dom'
import Header from './components/layout/Header'
import Footer from './components/layout/Footer'
import HomePage from './pages/HomePage'
import LoginPage from './pages/LoginPage'
import ShopPage from './pages/shop/ShopPage'
import ShopProductDetailPage from './pages/shop/ShopProductDetailPage'
import CartPage from './pages/CartPage'
import CheckoutPage from './pages/CheckoutPage'
import CheckoutSuccessPage from './pages/CheckoutSuccessPage'
import SignupPage from './pages/SignupPage'
import MySubscriptionPage from './pages/MySubscriptionPage'
import AdminLayout from './pages/admin/AdminLayout'
import SubscriptionPage from './pages/admin/subscription/SubscriptionPage'
import SubscriptionNewPage from './pages/admin/subscription/SubscriptionNewPage'
import AdminReportingPage from './pages/admin/AdminReportingPage'
import ProductEditPage from './pages/admin/products/ProductEditPage'
import ProductNewPage from './pages/admin/products/ProductNewPage'
import ProductsPage from './pages/admin/products/ProductsPage'
import RoleEditPage from './pages/admin/roles/RoleEditPage'
import RoleNewPage from './pages/admin/roles/RoleNewPage'
import RolesPage from './pages/admin/roles/RolesPage'
import UserEditPage from './pages/admin/users/UserEditPage'
import UsersPage from './pages/admin/users/UsersPage'
import AdminConfigurationsPage from './pages/admin/configurations/AdminConfigurationsPage'
import AttributePage from './pages/admin/configurations/attribute/AttributePage'
import AttributeNewPage from './pages/admin/configurations/attribute/AttributeNewPage'
import AttributeEditPage from './pages/admin/configurations/attribute/AttributeEditPage'
import RecurringPlanPage from './pages/admin/configurations/recurring-plan/RecurringPlanPage'
import RecurringPlanNewPage from './pages/admin/configurations/recurring-plan/RecurringPlanNewPage'
import RecurringPlanEditPage from './pages/admin/configurations/recurring-plan/RecurringPlanEditPage'
import QuotationTemplatePage from './pages/admin/configurations/quotation-template/QuotationTemplatePage'
import QuotationTemplateNewPage from './pages/admin/configurations/quotation-template/QuotationTemplateNewPage'
import QuotationTemplateEditPage from './pages/admin/configurations/quotation-template/QuotationTemplateEditPage'
import PaymentTermPage from './pages/admin/configurations/payment-term/PaymentTermPage'
import PaymentTermNewPage from './pages/admin/configurations/payment-term/PaymentTermNewPage'
import PaymentTermEditPage from './pages/admin/configurations/payment-term/PaymentTermEditPage'
import DiscountPage from './pages/admin/configurations/discount/DiscountPage'
import DiscountNewPage from './pages/admin/configurations/discount/DiscountNewPage'
import DiscountEditPage from './pages/admin/configurations/discount/DiscountEditPage'
import TaxesPage from './pages/admin/configurations/taxes/TaxesPage'
import TaxesNewPage from './pages/admin/configurations/taxes/TaxesNewPage'
import TaxesEditPage from './pages/admin/configurations/taxes/TaxesEditPage'
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
          <Route path="subscriptions" element={<SubscriptionPage />} />
          <Route path="subscriptions/new" element={<SubscriptionNewPage />} />
          <Route path="subscriptions/:subscriptionId" element={<SubscriptionNewPage />} />
          <Route path="products" element={<ProductsPage />} />
          <Route path="products/new" element={<ProductNewPage />} />
          <Route path="products/:productId" element={<ProductEditPage />} />
          <Route path="reporting" element={<AdminReportingPage />} />
          <Route path="users" element={<UsersPage />} />
          <Route path="users/:userId" element={<UserEditPage />} />
          <Route path="roles" element={<RolesPage />} />
          <Route path="roles/new" element={<RoleNewPage />} />
          <Route path="roles/:roleId" element={<RoleEditPage />} />
          <Route path="configurations" element={<AdminConfigurationsPage />} />
          <Route path="configurations/attribute" element={<AttributePage />} />
          <Route path="configurations/attribute/new" element={<AttributeNewPage />} />
          <Route path="configurations/attribute/:attributeId" element={<AttributeEditPage />} />
          <Route path="configurations/recurring-plan" element={<RecurringPlanPage />} />
          <Route path="configurations/recurring-plan/new" element={<RecurringPlanNewPage />} />
          <Route path="configurations/recurring-plan/:recurringPlanId" element={<RecurringPlanEditPage />} />
          <Route path="configurations/quotation-template" element={<QuotationTemplatePage />} />
          <Route path="configurations/quotation-template/new" element={<QuotationTemplateNewPage />} />
          <Route path="configurations/quotation-template/:quotationId" element={<QuotationTemplateEditPage />} />
          <Route path="configurations/payment-term" element={<PaymentTermPage />} />
          <Route path="configurations/payment-term/new" element={<PaymentTermNewPage />} />
          <Route path="configurations/payment-term/:paymentTermId" element={<PaymentTermEditPage />} />
          <Route path="configurations/discount" element={<DiscountPage />} />
          <Route path="configurations/discount/new" element={<DiscountNewPage />} />
          <Route path="configurations/discount/:discountId" element={<DiscountEditPage />} />
          <Route path="configurations/taxes" element={<TaxesPage />} />
          <Route path="configurations/taxes/new" element={<TaxesNewPage />} />
          <Route path="configurations/taxes/:taxId" element={<TaxesEditPage />} />
          <Route path="*" element={<Navigate to="subscriptions" replace />} />
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    )
  }

  return (
    <div className="public-workspace flex min-h-screen flex-col bg-[var(--light-bg)]">
      <Header />

      <main className="flex-1">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/home" element={<HomePage />} />
          <Route path="/shop" element={<ShopPage />} />
          <Route path="/shop/:productId" element={<ShopProductDetailPage />} />
          <Route path="/cart" element={<CartPage />} />
          <Route path="/check-out" element={<CheckoutPage />} />
          <Route path="/sucess" element={<CheckoutSuccessPage />} />
          <Route path="/success" element={<Navigate to={`/sucess${location.search}`} replace />} />
          <Route path="/subscription" element={<MySubscriptionPage />} />
          <Route path="/about" element={<HomePage />} />
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
