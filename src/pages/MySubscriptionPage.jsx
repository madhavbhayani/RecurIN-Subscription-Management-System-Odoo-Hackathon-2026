import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  downloadMySubscriptionInvoicePdf,
  downloadMySubscriptionQuotationPdf,
  listMySubscriptions,
  respondToQuotation,
} from '../services/checkoutApi'
import { getAuthSession } from '../services/session'

const TAB_ACTIVE = 'active'
const TAB_QUOTATIONS = 'quotations'

function formatDate(dateValue) {
  const value = String(dateValue ?? '').trim()
  if (!value) {
    return '-'
  }

  const parsedDate = new Date(value)
  if (Number.isNaN(parsedDate.getTime())) {
    return value
  }

  return parsedDate.toLocaleDateString('en-IN', {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
  })
}

function formatMoneyUSD(amountValue) {
  const numericValue = Number(amountValue)
  if (!Number.isFinite(numericValue)) {
    return '$0.00'
  }

  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(numericValue)
  } catch {
    return `$${numericValue.toFixed(2)}`
  }
}

function SubscriptionCards({
  items,
  emptyMessage,
  isQuotation = false,
  onQuotationAction,
  onDownloadInvoice,
  onDownloadQuotation,
}) {
  if (!items.length) {
    return (
      <div className="rounded-xl border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[color:rgba(0,0,128,0.7)]">
        {emptyMessage}
      </div>
    )
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {items.map((subscription) => {
        const products = Array.isArray(subscription.products) ? subscription.products : []
        const payment = subscription.payment ?? null

        return (
          <article
            key={subscription.subscription_id}
            className="rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] px-4 py-4 shadow-[0_6px_20px_rgba(0,0,128,0.07)]"
          >
            <p className="text-xs font-semibold uppercase tracking-[0.08em] text-[color:rgba(0,0,128,0.62)]">
              {subscription.status || '-'}
            </p>
            <h2 className="mt-1 text-lg font-bold text-[var(--navy)]">
              {subscription.subscription_number || '-'}
            </h2>

            <dl className="mt-3 space-y-1.5 text-sm text-[color:rgba(0,0,128,0.82)]">
              <div className="flex items-center justify-between gap-3">
                <dt className="font-semibold text-[var(--navy)]">Recurring Plan</dt>
                <dd>{subscription.plan || '-'}</dd>
              </div>
              <div className="flex items-center justify-between gap-3">
                <dt className="font-semibold text-[var(--navy)]">Billing Period</dt>
                <dd>{subscription.recurring || '-'}</dd>
              </div>
              <div className="flex items-center justify-between gap-3">
                <dt className="font-semibold text-[var(--navy)]">Next Invoice</dt>
                <dd>{formatDate(subscription.next_invoice_date)}</dd>
              </div>
              <div className="flex items-center justify-between gap-3">
                <dt className="font-semibold text-[var(--navy)]">Payment Term</dt>
                <dd>{subscription.payment_term_name || '-'}</dd>
              </div>
              <div className="flex items-center justify-between gap-3">
                <dt className="font-semibold text-[var(--navy)]">Products</dt>
                <dd>{products.length}</dd>
              </div>
            </dl>

            {payment ? (
              <div className="mt-3 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-emerald-800">
                <p><span className="font-semibold">Amount Paid:</span> {formatMoneyUSD(payment.amount_inr)}</p>
                <p className="mt-1"><span className="font-semibold">Status:</span> {payment.paypal_status || '-'}</p>
                <p className="mt-1"><span className="font-semibold">Date:</span> {formatDate(payment.payment_date)}</p>
              </div>
            ) : null}

            {/* Action buttons */}
            <div className="mt-3 flex flex-wrap gap-2">
              {!isQuotation && (
                <button
                  type="button"
                  onClick={() => onDownloadInvoice?.(subscription)}
                  className="inline-flex h-8 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-3 text-xs font-semibold text-[var(--navy)] transition-colors hover:border-[var(--orange)] hover:text-[var(--orange)]"
                >
                  📄 Download Invoice
                </button>
              )}

              {isQuotation && (
                <>
                  <button
                    type="button"
                    onClick={() => onDownloadQuotation?.(subscription)}
                    className="inline-flex h-8 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-3 text-xs font-semibold text-[var(--navy)] transition-colors hover:border-[var(--orange)] hover:text-[var(--orange)]"
                  >
                    📄 Download Quotation
                  </button>
                  {subscription.status === 'Quotation Sent' && (
                    <>
                      <button
                        type="button"
                        onClick={() => onQuotationAction?.(subscription.subscription_id, 'accept')}
                        className="inline-flex h-8 items-center rounded-lg bg-emerald-600 px-4 text-xs font-semibold text-white transition-colors hover:bg-emerald-700"
                      >
                        ✓ Accept
                      </button>
                      <button
                        type="button"
                        onClick={() => onQuotationAction?.(subscription.subscription_id, 'reject')}
                        className="inline-flex h-8 items-center rounded-lg bg-red-500 px-4 text-xs font-semibold text-white transition-colors hover:bg-red-600"
                      >
                        ✕ Reject
                      </button>
                    </>
                  )}
                </>
              )}
            </div>
          </article>
        )
      })}
    </div>
  )
}

function MySubscriptionPage() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState(TAB_ACTIVE)
  const [activeSubscriptions, setActiveSubscriptions] = useState([])
  const [quotationSubscriptions, setQuotationSubscriptions] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [actionMessage, setActionMessage] = useState('')

  const hasSession = Boolean(getAuthSession()?.token)

  const fetchMySubscriptions = async () => {
    if (!hasSession) {
      setActiveSubscriptions([])
      setQuotationSubscriptions([])
      setErrorMessage('')
      return
    }

    setIsLoading(true)
    setErrorMessage('')

    try {
      const response = await listMySubscriptions()
      setActiveSubscriptions(Array.isArray(response?.active_subscriptions) ? response.active_subscriptions : [])
      setQuotationSubscriptions(Array.isArray(response?.quotation_subscriptions) ? response.quotation_subscriptions : [])
    } catch (error) {
      setActiveSubscriptions([])
      setQuotationSubscriptions([])
      setErrorMessage(error.message)
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchMySubscriptions()
  }, [hasSession])

  const handleQuotationAction = async (subscriptionId, action) => {
    setActionMessage('')

    const normalizedSubscriptionID = String(subscriptionId ?? '').trim()
    if (!normalizedSubscriptionID) {
      setActionMessage('Error: Subscription ID is required.')
      return
    }

    if (action === 'accept') {
      const query = new URLSearchParams({
        subscription_id: normalizedSubscriptionID,
        checkout_mode: 'quotation',
      })
      navigate(`/check-out?${query.toString()}`)
      return
    }

    try {
      await respondToQuotation(normalizedSubscriptionID, action)
      setActionMessage(
        action === 'accept'
          ? 'Quotation accepted! Subscription is now active.'
          : 'Quotation has been rejected.',
      )
      // Refresh the list
      await fetchMySubscriptions()
    } catch (error) {
      setActionMessage(`Error: ${error.message}`)
    }
  }

  const handleInvoiceDownload = async (subscription) => {
    setActionMessage('')

    try {
      const fallbackFileName = `Invoice-${subscription?.subscription_number || 'RecurIN'}.pdf`
      await downloadMySubscriptionInvoicePdf(subscription?.subscription_id, fallbackFileName)
    } catch (error) {
      setActionMessage(`Error: ${error.message}`)
    }
  }

  const handleQuotationDownload = async (subscription) => {
    setActionMessage('')

    try {
      const fallbackFileName = `Quotation-${subscription?.subscription_number || 'RecurIN'}.pdf`
      await downloadMySubscriptionQuotationPdf(subscription?.subscription_id, fallbackFileName)
    } catch (error) {
      setActionMessage(`Error: ${error.message}`)
    }
  }

  const currentItems = useMemo(() => {
    return activeTab === TAB_ACTIVE ? activeSubscriptions : quotationSubscriptions
  }, [activeTab, activeSubscriptions, quotationSubscriptions])

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">My Subscriptions</h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)] sm:text-base">
          Track your currently active plans and quotations from one place.
        </p>

        {actionMessage && (
          <div className={`mt-4 rounded-xl px-4 py-3 text-sm font-medium ${actionMessage.startsWith('Error') ? 'border border-red-200 bg-red-50 text-red-700' : 'border border-emerald-200 bg-emerald-50 text-emerald-700'}`}>
            {actionMessage}
          </div>
        )}

        {!hasSession ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[var(--navy)]">
            Please log in to view your subscriptions.
            <div className="mt-3">
              <Link
                to="/login"
                className="inline-flex h-9 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-200 hover:bg-[#e65f00]"
              >
                Login
              </Link>
            </div>
          </div>
        ) : (
          <>
            <div className="mt-6 inline-flex rounded-xl border border-[color:rgba(0,0,128,0.16)] p-1">
              <button
                type="button"
                onClick={() => setActiveTab(TAB_ACTIVE)}
                className={`rounded-lg px-4 py-2 text-sm font-semibold transition-colors duration-200 ${
                  activeTab === TAB_ACTIVE
                    ? 'bg-[var(--orange)] text-[var(--white)]'
                    : 'text-[var(--navy)] hover:bg-[rgba(0,0,128,0.06)]'
                }`}
              >
                Active ({activeSubscriptions.length})
              </button>
              <button
                type="button"
                onClick={() => setActiveTab(TAB_QUOTATIONS)}
                className={`rounded-lg px-4 py-2 text-sm font-semibold transition-colors duration-200 ${
                  activeTab === TAB_QUOTATIONS
                    ? 'bg-[var(--orange)] text-[var(--white)]'
                    : 'text-[var(--navy)] hover:bg-[rgba(0,0,128,0.06)]'
                }`}
              >
                Quotations ({quotationSubscriptions.length})
              </button>
            </div>

            <div className="mt-5">
              {isLoading ? (
                <div className="rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-5 text-sm text-[color:rgba(0,0,128,0.66)]">
                  Loading subscriptions...
                </div>
              ) : errorMessage ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-5 text-sm text-red-700">
                  {errorMessage}
                </div>
              ) : (
                <SubscriptionCards
                  items={currentItems}
                  isQuotation={activeTab === TAB_QUOTATIONS}
                  onQuotationAction={handleQuotationAction}
                  onDownloadInvoice={handleInvoiceDownload}
                  onDownloadQuotation={handleQuotationDownload}
                  emptyMessage={
                    activeTab === TAB_ACTIVE
                      ? 'No active subscriptions available right now.'
                      : 'No quotations available right now.'
                  }
                />
              )}
            </div>
          </>
        )}
      </section>
    </div>
  )
}

export default MySubscriptionPage
