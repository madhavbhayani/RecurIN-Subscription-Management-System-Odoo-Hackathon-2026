import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { listMySubscriptions } from '../services/checkoutApi'
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

function formatMoney(amountValue, currencyCode = 'USD') {
  const numericValue = Number(amountValue)
  if (!Number.isFinite(numericValue)) {
    return `${currencyCode} 0.00`
  }

  const normalizedCurrency = String(currencyCode || 'USD').trim().toUpperCase()
  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: normalizedCurrency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(numericValue)
  } catch {
    return `${normalizedCurrency} ${numericValue.toFixed(2)}`
  }
}

function SubscriptionCards({ items, emptyMessage }) {
  if (!items.length) {
    return (
      <div className="rounded-xl border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[color:rgba(0,0,128,0.7)]">
        {emptyMessage}
      </div>
    )
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      {items.map((subscription) => (
        (() => {
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
              <dt className="font-semibold text-[var(--navy)]">Plan</dt>
              <dd>{subscription.plan || '-'}</dd>
            </div>
            <div className="flex items-center justify-between gap-3">
              <dt className="font-semibold text-[var(--navy)]">Recurring</dt>
              <dd>{subscription.recurring || '-'}</dd>
            </div>
            <div className="flex items-center justify-between gap-3">
              <dt className="font-semibold text-[var(--navy)]">Next Invoice</dt>
              <dd>{formatDate(subscription.next_invoice_date)}</dd>
            </div>
            <div className="flex items-center justify-between gap-3">
              <dt className="font-semibold text-[var(--navy)]">Quotation</dt>
              <dd>{subscription.quotation_id || '-'}</dd>
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
            <div className="mt-3 rounded-lg border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.03)] px-3 py-2 text-xs text-[color:rgba(0,0,128,0.8)]">
              <p><span className="font-semibold text-[var(--navy)]">Invoice:</span> {payment.invoice_number || '-'}</p>
              <p className="mt-1"><span className="font-semibold text-[var(--navy)]">Amount:</span> {formatMoney(payment.payment_amount, payment.payment_currency)}</p>
              <p className="mt-1"><span className="font-semibold text-[var(--navy)]">Date:</span> {formatDate(payment.payment_date)}</p>
            </div>
          ) : null}
        </article>
          )
        })()
      ))}
    </div>
  )
}

function MySubscriptionPage() {
  const [activeTab, setActiveTab] = useState(TAB_ACTIVE)
  const [activeSubscriptions, setActiveSubscriptions] = useState([])
  const [quotationSubscriptions, setQuotationSubscriptions] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')

  const hasSession = Boolean(getAuthSession()?.token)

  useEffect(() => {
    let isMounted = true

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
        if (!isMounted) {
          return
        }

        setActiveSubscriptions(Array.isArray(response?.active_subscriptions) ? response.active_subscriptions : [])
        setQuotationSubscriptions(Array.isArray(response?.quotation_subscriptions) ? response.quotation_subscriptions : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setActiveSubscriptions([])
        setQuotationSubscriptions([])
        setErrorMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchMySubscriptions()

    return () => {
      isMounted = false
    }
  }, [hasSession])

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
