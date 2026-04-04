import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { capturePayPalCheckoutOrder } from '../services/checkoutApi'
import { getAuthSession } from '../services/session'

function formatCurrency(value, currency = 'USD') {
  const numericValue = Number(value)
  const normalizedCurrency = String(currency || 'USD').trim().toUpperCase()
  if (!Number.isFinite(numericValue)) {
    return `${normalizedCurrency} 0.00`
  }

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

function CheckoutSuccessPage() {
  const [searchParams] = useSearchParams()
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [payment, setPayment] = useState(null)

  const hasSession = Boolean(getAuthSession()?.token)
  const orderID = String(searchParams.get('token') ?? '').trim()
  const paymentID = String(searchParams.get('paymentId') ?? searchParams.get('paymentID') ?? '').trim()
  const payerID = String(searchParams.get('PayerID') ?? searchParams.get('payerId') ?? '').trim()

  useEffect(() => {
    let isMounted = true

    const capturePayment = async () => {
      if (!hasSession) {
        setErrorMessage('Please log in to complete payment confirmation.')
        return
      }

      setIsLoading(true)
      setErrorMessage('')

      try {
        if (!orderID && paymentID && !payerID) {
          setPayment({
            order_id: paymentID,
            capture_id: '',
            status: 'PENDING',
            amount: 0,
            currency: 'USD',
            payer_email: '',
          })
          return
        }

        if (!orderID && !paymentID) {
          setErrorMessage('Missing PayPal order reference in return URL.')
          return
        }

        const payload = paymentID
          ? { payment_id: paymentID, payer_id: payerID }
          : { order_id: orderID }

        const response = await capturePayPalCheckoutOrder(payload)
        if (!isMounted) {
          return
        }

        setPayment(response?.payment ?? null)
      } catch (error) {
        if (!isMounted) {
          return
        }

        setErrorMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    capturePayment()

    return () => {
      isMounted = false
    }
  }, [hasSession, orderID, paymentID, payerID])

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">Payment Status</h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)] sm:text-base">
          We are verifying your PayPal payment and updating your order details.
        </p>

        {!hasSession ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[var(--navy)]">
            Please log in first.
          </div>
        ) : isLoading ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
            Capturing PayPal payment...
          </div>
        ) : errorMessage ? (
          <div className="mt-6 rounded-xl border border-red-200 bg-red-50 px-4 py-5 text-sm text-red-700">
            {errorMessage}
          </div>
        ) : (
          <div className="mt-6 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-5 text-sm text-emerald-800">
            <p className="text-base font-bold">Payment completed successfully.</p>

            <div className="mt-3 space-y-1 text-sm">
              <p><span className="font-semibold">Order ID:</span> {payment?.order_id || '-'}</p>
              <p><span className="font-semibold">Capture ID:</span> {payment?.capture_id || '-'}</p>
              <p><span className="font-semibold">Status:</span> {payment?.status || '-'}</p>
              <p><span className="font-semibold">Amount:</span> {formatCurrency(payment?.amount, payment?.currency)}</p>
              <p><span className="font-semibold">Payer Email:</span> {payment?.payer_email || '-'}</p>
            </div>
          </div>
        )}

        <div className="mt-6 flex flex-wrap gap-3">
          <Link
            to="/shop"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:border-[var(--orange)] hover:text-[var(--orange)]"
          >
            Back to Shop
          </Link>
          <Link
            to="/cart"
            className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            View Cart
          </Link>
        </div>
      </section>
    </div>
  )
}

export default CheckoutSuccessPage
