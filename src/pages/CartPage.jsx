import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import ToastMessage from '../components/common/ToastMessage'
import { deleteCartItem, listCartItems, updateCartItemQuantity } from '../services/cartApi'
import { getAuthSession } from '../services/session'

const CURRENCY_SYMBOL = '$'

function formatPrice(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return `${CURRENCY_SYMBOL}0.00`
  }

  return `${CURRENCY_SYMBOL}${numericValue.toFixed(2)}`
}

function CartPage() {
  const [cartItems, setCartItems] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isUpdatingItemID, setIsUpdatingItemID] = useState('')
  const [isRemovingItemID, setIsRemovingItemID] = useState('')

  const hasSession = Boolean(getAuthSession()?.token)

  useEffect(() => {
    let isMounted = true

    const fetchCartItems = async () => {
      if (!hasSession) {
        setCartItems([])
        setErrorMessage('')
        return
      }

      setIsLoading(true)
      setErrorMessage('')

      try {
        const response = await listCartItems()
        if (!isMounted) {
          return
        }

        setCartItems(Array.isArray(response?.cart_items) ? response.cart_items : [])
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

    fetchCartItems()

    return () => {
      isMounted = false
    }
  }, [hasSession])

  const cartGrandTotal = useMemo(() => {
    return cartItems.reduce((runningTotal, item) => {
      const lineTotal = Number(item?.line_total)
      return runningTotal + (Number.isFinite(lineTotal) ? lineTotal : 0)
    }, 0)
  }, [cartItems])

  const handleUpdateQuantity = async (item, nextQuantity) => {
    const cartItemID = String(item?.cart_item_id ?? '').trim()
    if (!cartItemID || nextQuantity < 1) {
      return
    }

    setIsUpdatingItemID(cartItemID)
    try {
      const response = await updateCartItemQuantity(cartItemID, {
        quantity: nextQuantity,
      })

      const updatedItem = response?.cart_item
      if (updatedItem) {
        setCartItems((previousItems) => previousItems.map((existingItem) => (
          existingItem.cart_item_id === cartItemID ? updatedItem : existingItem
        )))
      }
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsUpdatingItemID('')
    }
  }

  const handleRemoveItem = async (item) => {
    const cartItemID = String(item?.cart_item_id ?? '').trim()
    if (!cartItemID) {
      return
    }

    setIsRemovingItemID(cartItemID)
    try {
      await deleteCartItem(cartItemID)
      setCartItems((previousItems) => previousItems.filter((existingItem) => existingItem.cart_item_id !== cartItemID))
      setToastVariant('success')
      setToastMessage('Item removed from cart.')
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsRemovingItemID('')
    }
  }

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-6 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-8">
        <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">Cart</h1>
        <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.8)] sm:text-base">
          Review your selected products, quantities, and billing values before checkout.
        </p>

        {!hasSession ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-5 text-sm text-[var(--navy)]">
            Please log in to view and manage your cart.
          </div>
        ) : isLoading ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
            Loading cart items...
          </div>
        ) : errorMessage ? (
          <div className="mt-6 rounded-xl border border-red-200 bg-red-50 px-4 py-5 text-sm text-red-700">
            {errorMessage}
          </div>
        ) : cartItems.length === 0 ? (
          <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.12)] px-4 py-6 text-sm text-[color:rgba(0,0,128,0.66)]">
            Your cart is empty. Explore products to add items.
          </div>
        ) : (
          <>
            <div className="mt-6 overflow-x-auto rounded-xl border border-[color:rgba(0,0,128,0.12)]">
              <div className="min-w-[920px]">
                <div className="grid grid-cols-[1.65fr_0.75fr_0.5fr_0.75fr_0.75fr_105px] gap-3 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                  <span>Product</span>
                  <span>Recurring</span>
                  <span>Quantity</span>
                  <span>Unit</span>
                  <span>Line Total</span>
                  <span className="text-right">Action</span>
                </div>

                <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
                  {cartItems.map((item) => {
                    const cartItemID = String(item?.cart_item_id ?? '')
                    const quantity = Number(item?.quantity)
                    const safeQuantity = Number.isFinite(quantity) && quantity > 0 ? quantity : 1

                    return (
                      <div key={cartItemID} className="grid grid-cols-[1.65fr_0.75fr_0.5fr_0.75fr_0.75fr_105px] gap-3 px-4 py-4 text-sm text-[var(--navy)]">
                        <div>
                          <p className="font-semibold">{item.product_name}</p>
                          {item.selected_variant_attribute_name && (
                            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.66)]">
                              Variant: {item.selected_variant_attribute_name} (+{formatPrice(item.selected_variant_price)})
                            </p>
                          )}
                          {Number(item?.discount_amount) > 0 && (
                            <p className="mt-1 text-xs text-emerald-700">
                              Discount applied: {formatPrice(item.discount_amount)}
                            </p>
                          )}
                        </div>

                        <div>{item.billing_period || 'Monthly'}</div>

                        <div className="inline-grid h-8 w-[84px] grid-cols-[24px_36px_24px] items-center overflow-hidden rounded-md border border-[color:rgba(0,0,128,0.18)]">
                          <button
                            type="button"
                            onClick={() => handleUpdateQuantity(item, safeQuantity - 1)}
                            disabled={safeQuantity <= 1 || isUpdatingItemID === cartItemID}
                            className="inline-flex h-full w-full items-center justify-center text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)] disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            -
                          </button>
                          <span className="inline-flex h-full w-full items-center justify-center border-x border-[color:rgba(0,0,128,0.16)] text-xs font-semibold">
                            {safeQuantity}
                          </span>
                          <button
                            type="button"
                            onClick={() => handleUpdateQuantity(item, safeQuantity + 1)}
                            disabled={isUpdatingItemID === cartItemID}
                            className="inline-flex h-full w-full items-center justify-center text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)] disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            +
                          </button>
                        </div>

                        <div>{formatPrice(item.effective_unit_price)}</div>
                        <div className="font-semibold">{formatPrice(item.line_total)}</div>

                        <div className="flex justify-end">
                          <button
                            type="button"
                            onClick={() => handleRemoveItem(item)}
                            disabled={isRemovingItemID === cartItemID}
                            className="inline-flex h-9 items-center rounded-lg border border-red-300 px-3 text-xs font-semibold text-red-700 transition-colors duration-200 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60"
                          >
                            {isRemovingItemID === cartItemID ? 'Removing...' : 'Remove'}
                          </button>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>

            <div className="mt-4 flex justify-end">
              <div className="rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[rgba(0,0,128,0.03)] px-4 py-3 text-sm text-[var(--navy)]">
                <span className="font-semibold">Grand Total: </span>
                <span className="text-base font-bold">{formatPrice(cartGrandTotal)}</span>
              </div>
            </div>
          </>
        )}

        <div className="mt-6 flex flex-wrap gap-3">
          <Link
            to="/shop"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-300 hover:border-[var(--orange)] hover:text-[var(--orange)]"
          >
            Continue Shopping
          </Link>
          {!hasSession ? (
            <Link
              to="/login"
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
            >
              Login to Checkout
            </Link>
          ) : (
            <Link
              to="/check-out"
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-4 text-sm font-semibold text-[var(--white)] transition-colors duration-300 hover:bg-[#e65f00]"
            >
              Proceed to Checkout
            </Link>
          )}
        </div>
      </section>
    </div>
  )
}

export default CartPage
