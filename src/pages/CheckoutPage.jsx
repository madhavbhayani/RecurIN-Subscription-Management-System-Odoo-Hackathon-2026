import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import ToastMessage from '../components/common/ToastMessage'
import { listCartItems } from '../services/cartApi'
import {
  createPayPalCheckoutOrder,
  getMyCheckoutProfile,
  listMySubscriptions,
  updateMyCheckoutAddress,
} from '../services/checkoutApi'
import { getAuthSession } from '../services/session'

const CHECKOUT_SNAPSHOT_KEY = 'recurin_checkout_snapshot'

function formatUsdCurrency(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '$0.00'
  }

  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(numericValue)
}

function CheckoutPage() {
  const [searchParams] = useSearchParams()
  const [cartItems, setCartItems] = useState([])
  const [quotationSubscription, setQuotationSubscription] = useState(null)
  const [addressInput, setAddressInput] = useState('')
  const [storedAddress, setStoredAddress] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')

  const hasSession = Boolean(getAuthSession()?.token)
  const checkoutMode = String(searchParams.get('checkout_mode') ?? '').trim().toLowerCase()
  const quotationSubscriptionID = String(searchParams.get('subscription_id') ?? '').trim()
  const isQuotationCheckout = checkoutMode === 'quotation' || quotationSubscriptionID !== ''

  useEffect(() => {
    let isMounted = true

    const fetchCheckoutData = async () => {
      if (!hasSession) {
        setCartItems([])
        setQuotationSubscription(null)
        setAddressInput('')
        setStoredAddress('')
        setErrorMessage('')
        return
      }

      setIsLoading(true)
      setErrorMessage('')

      try {
        if (isQuotationCheckout) {
          const [profileResponse, subscriptionsResponse] = await Promise.all([
            getMyCheckoutProfile(),
            listMySubscriptions(),
          ])

          if (!isMounted) {
            return
          }

          const userAddress = String(profileResponse?.user?.address ?? '').trim()
          const activeSubscriptions = Array.isArray(subscriptionsResponse?.active_subscriptions)
            ? subscriptionsResponse.active_subscriptions
            : []
          const quotationSubscriptions = Array.isArray(subscriptionsResponse?.quotation_subscriptions)
            ? subscriptionsResponse.quotation_subscriptions
            : []
          const allSubscriptions = [...quotationSubscriptions, ...activeSubscriptions]
          const selectedSubscription = allSubscriptions.find(
            (item) => String(item?.subscription_id ?? '').trim() === quotationSubscriptionID,
          )

          if (!selectedSubscription) {
            throw new Error('Selected quotation was not found for checkout.')
          }

          const selectedStatus = String(selectedSubscription?.status ?? '').trim()
          if (selectedStatus && selectedStatus !== 'Quotation Sent') {
            throw new Error(`Payment is only allowed when quotation status is Quotation Sent (current: ${selectedStatus}).`)
          }

          setQuotationSubscription(selectedSubscription)
          setCartItems([])
          setAddressInput(userAddress)
          setStoredAddress(userAddress)
          return
        }

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
        setQuotationSubscription(null)
        setAddressInput(userAddress)
        setStoredAddress(userAddress)
      } catch (error) {
        if (!isMounted) {
          return
        }

        setCartItems([])
        setQuotationSubscription(null)
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
  }, [hasSession, isQuotationCheckout, quotationSubscriptionID])

  const checkoutItems = useMemo(() => {
    if (!isQuotationCheckout) {
      return cartItems
    }

    const products = Array.isArray(quotationSubscription?.products) ? quotationSubscription.products : []
    const fallbackBillingPeriod = String(quotationSubscription?.recurring ?? '').trim()

    return products.map((product, index) => {
      const quantity = Number(product?.quantity)
      const safeQuantity = Number.isFinite(quantity) && quantity > 0 ? quantity : 1
      const unitPrice = Number(product?.unit_price)
      const totalAmount = Number(product?.total_amount)
      const variantExtraAmount = Number(product?.variant_extra_amount)
      const discountAmount = Number(product?.discount_amount)

      const normalizedLineTotal = Number.isFinite(totalAmount) && totalAmount > 0
        ? totalAmount
        : (Number.isFinite(unitPrice) ? unitPrice * safeQuantity : 0)

      const selectedVariants = Array.isArray(product?.selected_variants) ? product.selected_variants : []
      const selectedVariantLabel = selectedVariants
        .map((selectedVariant) => {
          const attributeName = String(selectedVariant?.attribute_name ?? '').trim()
          const attributeValue = String(selectedVariant?.attribute_value ?? '').trim()
          if (attributeName && attributeValue) {
            return `${attributeName}: ${attributeValue}`
          }
          return attributeValue || attributeName
        })
        .filter(Boolean)
        .join(', ')

      return {
        cart_item_id: String(product?.subscription_product_id ?? `${product?.product_id ?? 'quotation'}-${index}`),
        product_id: product?.product_id || null,
        product_name: product?.product_name || 'Product',
        quantity: safeQuantity,
        unit_price: Number.isFinite(unitPrice) ? unitPrice : 0,
        selected_variant_price: Number.isFinite(variantExtraAmount) ? variantExtraAmount : 0,
        discount_amount: Number.isFinite(discountAmount) ? discountAmount : 0,
        effective_unit_price: Number.isFinite(unitPrice) ? unitPrice : 0,
        line_total: Number.isFinite(normalizedLineTotal) ? normalizedLineTotal : 0,
        billing_period: fallbackBillingPeriod,
        selected_variant_attribute_name: selectedVariantLabel || null,
      }
    })
  }, [cartItems, isQuotationCheckout, quotationSubscription])

  const checkoutTotal = useMemo(() => {
    return checkoutItems.reduce((runningTotal, item) => {
      const lineTotal = Number(item?.line_total)
      return runningTotal + (Number.isFinite(lineTotal) ? lineTotal : 0)
    }, 0)
  }, [checkoutItems])

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

    if (checkoutItems.length === 0) {
      setToastVariant('error')
      setToastMessage(isQuotationCheckout ? 'No quotation items available for payment.' : 'Your cart is empty.')
      return
    }

    if (isQuotationCheckout && !quotationSubscriptionID) {
      setToastVariant('error')
      setToastMessage('Subscription ID is missing for quotation checkout.')
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

      const orderResponse = await createPayPalCheckoutOrder(
        isQuotationCheckout ? { subscription_id: quotationSubscriptionID } : undefined,
      )
      const approvalURL = String(orderResponse?.approval_url ?? '').trim()

      if (!approvalURL) {
        throw new Error('Unable to start PayPal checkout. Please try again.')
      }

      try {
        const snapshotItems = checkoutItems.map((item) => {
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
          amount_inr: Number(checkoutTotal.toFixed(2)),
          currency: 'USD',
          item_count: snapshotItems.length,
          payment_method: 'PayPal',
          address: normalizedAddress,
          checkout_mode: isQuotationCheckout ? 'quotation' : 'cart',
          subscription_id: isQuotationCheckout ? quotationSubscriptionID : null,
          subscription_number: isQuotationCheckout
            ? String(quotationSubscription?.subscription_number ?? '').trim() || null
            : null,
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
          {isQuotationCheckout
            ? 'Review your quotation, confirm your address, and continue to PayPal to confirm this subscription.'
            : 'Add your address, preview your order, and continue to PayPal to complete payment.'}
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
        ) : checkoutItems.length === 0 ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
            {isQuotationCheckout
              ? 'No quotation items are available for this subscription.'
              : 'Your cart is empty. Add products before checkout.'}
            <div className="mt-3">
              <Link
                to={isQuotationCheckout ? '/subscription' : '/shop'}
                className="inline-flex h-9 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:border-[var(--orange)] hover:text-[var(--orange)]"
              >
                {isQuotationCheckout ? 'Back to Subscriptions' : 'Go to Shop'}
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
              <h2 className="text-lg font-bold text-[var(--navy)]">
                {isQuotationCheckout ? 'Quotation Preview' : 'Order Preview'}
              </h2>

              {isQuotationCheckout && quotationSubscription ? (
                <div className="mt-3 rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-3 py-2 text-xs text-[color:rgba(0,0,128,0.76)]">
                  <p><span className="font-semibold text-[var(--navy)]">Subscription:</span> {quotationSubscription.subscription_number || '-'}</p>
                  <p className="mt-1"><span className="font-semibold text-[var(--navy)]">Status:</span> {quotationSubscription.status || '-'}</p>
                </div>
              ) : null}

              <div className="mt-4 space-y-3">
                {checkoutItems.map((item) => {
                  const cartItemID = String(item?.cart_item_id ?? item?.product_id ?? '')
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
                  <span className="font-semibold">Total Amount ($)</span>
                  <span className="text-base font-bold">{formatUsdCurrency(checkoutTotal)}</span>
                </p>
              </div>

              <button
                type="button"
                onClick={handleProceedToPay}
                disabled={isSubmitting}
                className="mt-4 inline-flex h-11 w-full items-center justify-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-200 hover:bg-[#e65f00] disabled:cursor-not-allowed disabled:opacity-70"
              >
                {isSubmitting ? 'Redirecting to PayPal...' : `Pay ${formatUsdCurrency(checkoutTotal)}`}
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
