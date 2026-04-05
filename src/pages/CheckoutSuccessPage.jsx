import { useEffect, useRef, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { capturePayPalCheckoutOrder, downloadMySubscriptionInvoicePdf } from '../services/checkoutApi'
import { getAuthSession } from '../services/session'

const CHECKOUT_SNAPSHOT_KEY = 'recurin_checkout_snapshot'

function decodePayPalParam(rawValue) {
  let value = String(rawValue ?? '').trim()
  if (!value) {
    return ''
  }

  for (let index = 0; index < 2; index += 1) {
    try {
      const decoded = decodeURIComponent(value)
      if (decoded === value) {
        break
      }
      value = decoded
    } catch {
      break
    }
  }

  return value
}

function readCheckoutSnapshot() {
  try {
    const snapshotText = sessionStorage.getItem(CHECKOUT_SNAPSHOT_KEY)
    if (!snapshotText) {
      return null
    }

    const parsedSnapshot = JSON.parse(snapshotText)
    return parsedSnapshot && typeof parsedSnapshot === 'object' ? parsedSnapshot : null
  } catch {
    return null
  }
}

function formatInrCurrency(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return 'INR 0.00'
  }

  try {
    return new Intl.NumberFormat('en-IN', {
      style: 'currency',
      currency: 'INR',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(numericValue)
  } catch {
    return `INR ${numericValue.toFixed(2)}`
  }
}

function CheckoutSuccessPage() {
  const [searchParams] = useSearchParams()
  const [payment, setPayment] = useState(null)
  const [downloadMessage, setDownloadMessage] = useState('')
  const [isDownloadingInvoice, setIsDownloadingInvoice] = useState(false)
  const [displayedAt] = useState(() => new Date())
  const checkoutSnapshotRef = useRef(readCheckoutSnapshot())
  const hasCapturedRef = useRef(false)

  const hasSession = Boolean(getAuthSession()?.token)
  const orderID = decodePayPalParam(searchParams.get('token'))
  const paymentID = decodePayPalParam(searchParams.get('paymentId') ?? searchParams.get('paymentID'))
  const payerID = decodePayPalParam(searchParams.get('PayerID') ?? searchParams.get('payerId'))
  const subscriptionIDFromURL = decodePayPalParam(searchParams.get('subscription_id'))
  const checkoutSnapshot = checkoutSnapshotRef.current
  const checkoutModeParam = String(searchParams.get('checkout_mode') ?? '').trim().toLowerCase()
  const checkoutModeFromSnapshot = String(checkoutSnapshot?.checkout_mode ?? '').trim().toLowerCase()
  const checkoutMode = checkoutModeParam || checkoutModeFromSnapshot
  const snapshotSubscriptionID = String(checkoutSnapshot?.subscription_id ?? '').trim()
  const quotationSubscriptionID = String(subscriptionIDFromURL || snapshotSubscriptionID).trim()
  const isQuotationCheckout = checkoutMode === 'quotation' || quotationSubscriptionID !== ''
  const invoiceNumber = String(
    payment?.capture_id
      ?? payment?.order_id
      ?? paymentID
      ?? orderID
      ?? ''
  ).trim()
  const checkedOutAmount = Number(checkoutSnapshotRef.current?.amount_inr)
  const amountInINR = Number.isFinite(checkedOutAmount) ? checkedOutAmount : Number(payment?.amount)
  const capturedSubscriptionIDs = Array.isArray(payment?.subscription_ids) ? payment.subscription_ids : []
  const resolvedSubscriptionID = String(capturedSubscriptionIDs[0] ?? quotationSubscriptionID).trim()

  const handleDownloadInvoice = async () => {
    setDownloadMessage('')

    if (!resolvedSubscriptionID) {
      setDownloadMessage('Invoice is being prepared. You can download it from My Subscriptions shortly.')
      return
    }

    setIsDownloadingInvoice(true)
    try {
      const fallbackFileName = `Invoice-${invoiceNumber || resolvedSubscriptionID}.pdf`
      await downloadMySubscriptionInvoicePdf(resolvedSubscriptionID, fallbackFileName)
    } catch (error) {
      setDownloadMessage(error.message)
    } finally {
      setIsDownloadingInvoice(false)
    }
  }

  useEffect(() => {
    if (hasCapturedRef.current) {
      return
    }
    hasCapturedRef.current = true

    let isMounted = true

    const capturePayment = async () => {
      if (!hasSession) {
        console.log('[CAPTURE] No session, skipping capture')
        return
      }

      try {
        if (!orderID && !paymentID) {
          console.log('[CAPTURE] No orderID or paymentID found in URL params')
          return
        }

        const payload = paymentID
          ? { payment_id: paymentID, payer_id: payerID }
          : { order_id: orderID }

        if (quotationSubscriptionID) {
          payload.subscription_id = quotationSubscriptionID
        }

        console.log('[CAPTURE] URL params:', { orderID, paymentID, payerID })
        console.log('[CAPTURE] Sending capture request with payload:', JSON.stringify(payload))

        const response = await capturePayPalCheckoutOrder(payload)
        if (!isMounted) {
          return
        }

        console.log('[CAPTURE] Capture response:', JSON.stringify(response))
        setPayment(response?.payment ?? null)
      } catch (error) {
        console.error('[CAPTURE] ERROR capturing payment:', error?.message || error)
        if (!isMounted) {
          return
        }
      }
    }

    capturePayment()

    return () => {
      isMounted = false
    }
  }, [hasSession, orderID, paymentID, payerID, quotationSubscriptionID])

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">Payment Status</h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)] sm:text-base">
          {isQuotationCheckout
            ? 'Your quotation payment has been completed and the existing subscription is now confirmed.'
            : 'Your PayPal checkout has been completed and the invoice is ready.'}
        </p>

        {!hasSession ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[var(--navy)]">
            Please log in first.
          </div>
        ) : (
          <>
          <div className="mt-6 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-5 text-sm text-emerald-800">
            <p className="text-base font-bold">Payment completed successfully.</p>

            <div className="mt-3 space-y-1 text-sm">
              <p><span className="font-semibold">Invoice Number:</span> {invoiceNumber || '-'}</p>
              <p><span className="font-semibold">Order ID:</span> {payment?.order_id || orderID || paymentID || '-'}</p>
              <p><span className="font-semibold">Status:</span> COMPLETED</p>
              <p><span className="font-semibold">Amount:</span> {formatInrCurrency(amountInINR)}</p>
              <p><span className="font-semibold">Payment Date:</span> {displayedAt.toLocaleString()}</p>
            </div>
          </div>

          <div className="mt-3 rounded-xl border border-blue-200 bg-blue-50 px-4 py-4 text-sm text-blue-800">
            <p className="font-bold">{isQuotationCheckout ? 'Subscription Confirmed' : 'Subscription Created'}</p>
            <p className="mt-1">
              {isQuotationCheckout
                ? 'Payment has been recorded for your existing quotation subscription. You can view it in My Subscriptions.'
                : 'Your subscription has been automatically activated based on your cart items. You can view and manage it from My Subscriptions.'}
            </p>
          </div>

          {downloadMessage && (
            <div className="mt-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
              {downloadMessage}
            </div>
          )}
          </>
        )}

        <div className="mt-6 flex flex-wrap gap-3">
          <button
            type="button"
            onClick={handleDownloadInvoice}
            disabled={isDownloadingInvoice}
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:border-[var(--orange)] hover:text-[var(--orange)]"
          >
            {isDownloadingInvoice ? 'Preparing Invoice...' : 'Download Invoice'}
          </button>
          <Link
            to="/shop"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:border-[var(--orange)] hover:text-[var(--orange)]"
          >
            Back to Shop
          </Link>
          <Link
            to="/subscription"
            className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
          >
            View My Subscriptions
          </Link>
        </div>
      </section>
    </div>
  )
}

export default CheckoutSuccessPage
