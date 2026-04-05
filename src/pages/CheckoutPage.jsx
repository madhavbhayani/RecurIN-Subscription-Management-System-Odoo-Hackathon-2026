import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import ToastMessage from '../components/common/ToastMessage'
import { listCartItems } from '../services/cartApi'
import {
  createPayPalCheckoutOrder,
  getMyCheckoutProfile,
  updateMyCheckoutAddress,
} from '../services/checkoutApi'
import { getAuthSession } from '../services/session'

const CHECKOUT_SNAPSHOT_KEY = 'recurin_checkout_snapshot'

function formatUsdCurrency(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return 'USD 0.00'
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(numericValue)
}

function CheckoutPage() {
  const [cartItems, setCartItems] = useState([])
  const [addressInput, setAddressInput] = useState('')
  const [storedAddress, setStoredAddress] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')

  const hasSession = Boolean(getAuthSession()?.token)

  useEffect(() => {
    let isMounted = true

    const fetchCheckoutData = async () => {
      if (!hasSession) {
        setCartItems([])
        setAddressInput('')
        setStoredAddress('')
        setErrorMessage('')
        return
      }

      setIsLoading(true)
      setErrorMessage('')

      try {
        const [cartResponse, profileResponse] = await Promise.all([
          listCartItems(),
          getMyCheckoutProfile(),
        ])

        if (!isMounted) {
          return
        }

        const items = Array.isArray(cartResponse?.cart_items) ? cartResponse.cart_items : []
        const userAddress = String(profileResponse?.user?.address ?? '').trim()

        setCartItems(items)
        setAddressInput(userAddress)
        setStoredAddress(userAddress)
      } catch (error) {
        if (!isMounted) {
          return
        }

        setCartItems([])
        setErrorMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchCheckoutData()

    return () => {
      isMounted = false
    }
  }, [hasSession])

  const cartTotal = useMemo(() => {
    return cartItems.reduce((runningTotal, item) => {
      const lineTotal = Number(item?.line_total)
      return runningTotal + (Number.isFinite(lineTotal) ? lineTotal : 0)
    }, 0)
  }, [cartItems])

  const handleProceedToPay = async () => {
    if (!hasSession) {
      setToastVariant('error')
      setToastMessage('Please log in to continue checkout.')
      return
    }

    const normalizedAddress = addressInput.trim()
    if (!normalizedAddress) {
      setToastVariant('error')
      setToastMessage('Please enter your address before payment.')
      return
    }

    if (cartItems.length === 0) {
      setToastVariant('error')
      setToastMessage('Your cart is empty.')
      return
    }

    setIsSubmitting(true)
    try {
      if (normalizedAddress !== storedAddress) {
        const updateResponse = await updateMyCheckoutAddress(normalizedAddress)
        const updatedAddress = String(updateResponse?.user?.address ?? normalizedAddress).trim()
        setStoredAddress(updatedAddress)
        setAddressInput(updatedAddress)
      }

      const orderResponse = await createPayPalCheckoutOrder()
      const approvalURL = String(orderResponse?.approval_url ?? '').trim()

      if (!approvalURL) {
        throw new Error('Unable to start PayPal checkout. Please try again.')
      }

      try {
        const snapshotItems = cartItems.map((item) => {
          const quantity = Number(item?.quantity)
          const safeQuantity = Number.isFinite(quantity) && quantity > 0 ? quantity : 1
          const unitPrice = Number(item?.unit_price)
          const selectedVariantPrice = Number(item?.selected_variant_price)
          const discountAmount = Number(item?.discount_amount)
          const effectiveUnitPrice = Number(item?.effective_unit_price)
          const lineTotal = Number(item?.line_total)

          return {
            product_id: item?.product_id || null,
            product_name: item?.product_name || 'Product',
            quantity: safeQuantity,
            unit_price: Number.isFinite(unitPrice) ? unitPrice : 0,
            variant_extra_amount: Number.isFinite(selectedVariantPrice) ? selectedVariantPrice : 0,
            discount_amount: Number.isFinite(discountAmount) ? discountAmount : 0,
            effective_unit_price: Number.isFinite(effectiveUnitPrice) ? effectiveUnitPrice : 0,
            tax_amount: 0,
            total_amount: Number.isFinite(lineTotal) ? lineTotal : 0,
            billing_period: item?.billing_period || '',
            selected_variant_attribute_name: item?.selected_variant_attribute_name || null,
          }
        })

        const checkoutSnapshot = {
          amount_inr: Number(cartTotal.toFixed(2)),
          currency: 'INR',
          item_count: snapshotItems.length,
          payment_method: 'PayPal',
          address: normalizedAddress,
          items: snapshotItems,
          created_at: new Date().toISOString(),
        }
        sessionStorage.setItem(CHECKOUT_SNAPSHOT_KEY, JSON.stringify(checkoutSnapshot))
      } catch {
        // Ignore storage failures and continue checkout.
      }

      window.location.assign(approvalURL)
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
      setIsSubmitting(false)
    }
  }

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">Checkout</h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)] sm:text-base">
          Add your address, preview your order, and continue to PayPal to complete payment.
        </p>

        {!hasSession ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[var(--navy)]">
            Please log in to continue checkout.
            <div className="mt-3">
              <Link
                to="/login"
                className="inline-flex h-9 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-200 hover:bg-[#e65f00]"
              >
                Login
              </Link>
            </div>
          </div>
        ) : isLoading ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
            Loading checkout details...
          </div>
        ) : errorMessage ? (
          <div className="mt-6 rounded-xl border border-red-200 bg-red-50 px-4 py-5 text-sm text-red-700">
            {errorMessage}
          </div>
        ) : cartItems.length === 0 ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
            Your cart is empty. Add products before checkout.
            <div className="mt-3">
              <Link
                to="/shop"
                className="inline-flex h-9 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:border-[var(--orange)] hover:text-[var(--orange)]"
              >
                Go to Shop
              </Link>
            </div>
          </div>
        ) : (
          <div className="mt-6 grid gap-5 lg:grid-cols-[1fr_1.1fr]">
            <div className="rounded-xl border border-[color:rgba(0,0,128,0.12)] p-4 sm:p-5">
              <h2 className="text-lg font-bold text-[var(--navy)]">Shipping Address</h2>
              <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.68)]">
                This address will be saved to your profile and shown in this order preview.
              </p>

              <label htmlFor="checkout-address" className="mt-4 block text-sm font-semibold text-[var(--navy)]">
                Address
              </label>
              <textarea
                id="checkout-address"
                rows={5}
                value={addressInput}
                onChange={(event) => setAddressInput(event.target.value)}
                placeholder="Enter your full delivery/billing address"
                className="mt-2 w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-3 py-2 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              />

              <div className="mt-4 rounded-lg border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.03)] px-3 py-2 text-xs text-[color:rgba(0,0,128,0.75)]">
                <span className="font-semibold text-[var(--navy)]">Address Preview:</span>{' '}
                {addressInput.trim() || 'No address entered yet.'}
              </div>
            </div>

            <div className="rounded-xl border border-[color:rgba(0,0,128,0.12)] p-4 sm:p-5">
              <h2 className="text-lg font-bold text-[var(--navy)]">Order Preview</h2>

              <div className="mt-4 space-y-3">
                {cartItems.map((item) => {
                  const cartItemID = String(item?.cart_item_id ?? '')
                  const quantity = Number(item?.quantity)
                  const safeQuantity = Number.isFinite(quantity) && quantity > 0 ? quantity : 1

                  return (
                    <div
                      key={cartItemID}
                      className="rounded-lg border border-[color:rgba(0,0,128,0.1)] px-3 py-2 text-sm text-[var(--navy)]"
                    >
                      <div className="flex items-start justify-between gap-2">
                        <p className="font-semibold">{item.product_name}</p>
                        <p className="font-semibold">{formatUsdCurrency(item.line_total)}</p>
                      </div>

                      <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.7)]">
                        Qty: {safeQuantity} | Recurring: {item.billing_period || 'Monthly'}
                      </p>
                      {item.selected_variant_attribute_name && (
                        <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.7)]">
                          Variant: {item.selected_variant_attribute_name}
                        </p>
                      )}
                    </div>
                  )
                })}
              </div>

              <div className="mt-4 rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-3 text-sm text-[var(--navy)]">
                <p className="flex items-center justify-between">
                  <span className="font-semibold">Total Amount (USD)</span>
                  <span className="text-base font-bold">{formatUsdCurrency(cartTotal)}</span>
                </p>
              </div>

              <button
                type="button"
                onClick={handleProceedToPay}
                disabled={isSubmitting}
                className="mt-4 inline-flex h-11 w-full items-center justify-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-200 hover:bg-[#e65f00] disabled:cursor-not-allowed disabled:opacity-70"
              >
                {isSubmitting ? 'Redirecting to PayPal...' : `Pay ${formatUsdCurrency(cartTotal)}`}
              </button>

              <p className="mt-2 text-xs text-[color:rgba(0,0,128,0.68)]">
                Clicking the button will redirect you to the PayPal payment gateway.
              </p>
            </div>
          </div>
        )}
      </section>
    </div>
  )
}

export default CheckoutPage
