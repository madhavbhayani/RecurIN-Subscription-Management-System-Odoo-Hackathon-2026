import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import ToastMessage from '../../components/common/ToastMessage'
import { addCartItem } from '../../services/cartApi'
import { listProductsPublic } from '../../services/productApi'
import { getAuthSession } from '../../services/session'

const CURRENCY_SYMBOL = '\u20b9'

function formatPrice(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return `${CURRENCY_SYMBOL} 0.00`
  }

  return `${CURRENCY_SYMBOL} ${numericValue.toFixed(2)}`
}

function resolveStaticImageURL(product) {
  const rawSeed = String(product?.product_id ?? product?.product_name ?? 'detail').trim()
  const encodedSeed = encodeURIComponent(rawSeed || 'detail')
  return `https://picsum.photos/seed/recurin-shop-${encodedSeed}/1200/800`
}

function calculateEstimatedDiscount(baseAmount, discounts) {
  if (baseAmount <= 0 || !Array.isArray(discounts) || discounts.length === 0) {
    return 0
  }

  let totalDiscount = 0
  discounts.forEach((discount) => {
    const discountUnit = String(discount?.discount_unit ?? '').trim()
    const discountValue = Number(discount?.discount_value)
    if (!Number.isFinite(discountValue) || discountValue <= 0) {
      return
    }

    if (discountUnit === 'Percentage') {
      totalDiscount += baseAmount * (discountValue / 100)
      return
    }

    if (discountUnit === 'Fixed Price') {
      totalDiscount += discountValue
    }
  })

  return Math.min(baseAmount, Math.round(totalDiscount * 100) / 100)
}

function ShopProductDetailPage() {
  const { productId = '' } = useParams()

  const [product, setProduct] = useState(null)
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [selectedVariantID, setSelectedVariantID] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [isAdding, setIsAdding] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')

  useEffect(() => {
    let isMounted = true

    const fetchProduct = async () => {
      const normalizedProductID = String(productId).trim()
      if (!normalizedProductID) {
        setProduct(null)
        setErrorMessage('Invalid product ID.')
        return
      }

      setIsLoading(true)
      setErrorMessage('')

      try {
        const response = await listProductsPublic('')
        if (!isMounted) {
          return
        }

        const allProducts = Array.isArray(response?.products) ? response.products : []
        const matchedProduct = allProducts.find((item) => String(item?.product_id ?? '').trim() === normalizedProductID) ?? null

        if (!matchedProduct) {
          setProduct(null)
          setErrorMessage('Product not found.')
          return
        }

        setProduct(matchedProduct)
        setSelectedVariantID('')
      } catch (error) {
        if (!isMounted) {
          return
        }

        setProduct(null)
        setErrorMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchProduct()

    return () => {
      isMounted = false
    }
  }, [productId])

  const variants = useMemo(() => {
    return Array.isArray(product?.variants) ? product.variants : []
  }, [product])

  const selectedVariant = useMemo(() => {
    if (!selectedVariantID) {
      return null
    }

    return variants.find((variant) => String(variant?.attribute_id ?? '') === selectedVariantID) ?? null
  }, [selectedVariantID, variants])

  const selectedVariantExtraPrice = useMemo(() => {
    const numericValue = Number(selectedVariant?.default_extra_price)
    if (!Number.isFinite(numericValue) || numericValue < 0) {
      return 0
    }

    return Math.round(numericValue * 100) / 100
  }, [selectedVariant])

  const selectedDiscountRules = useMemo(() => {
    return Array.isArray(product?.discounts) ? product.discounts : []
  }, [product?.discounts])

  const basePrice = Number(product?.sales_price)
  const safeBasePrice = Number.isFinite(basePrice) ? basePrice : 0

  const estimatedDiscount = useMemo(() => {
    return calculateEstimatedDiscount(safeBasePrice + selectedVariantExtraPrice, selectedDiscountRules)
  }, [safeBasePrice, selectedVariantExtraPrice, selectedDiscountRules])

  const estimatedUnitTotal = Math.max(0, safeBasePrice + selectedVariantExtraPrice - estimatedDiscount)
  const estimatedLineTotal = Math.round(estimatedUnitTotal * quantity * 100) / 100

  const handleAddToCart = async () => {
    const normalizedProductID = String(product?.product_id ?? '').trim()
    if (!normalizedProductID) {
      return
    }

    const session = getAuthSession()
    if (!session?.token) {
      setToastVariant('error')
      setToastMessage('Please log in to add this product to your cart.')
      return
    }

    setIsAdding(true)
    try {
      const response = await addCartItem({
        product_id: normalizedProductID,
        quantity,
        selected_variant_attribute_id: String(selectedVariant?.attribute_id ?? ''),
      })

      const lineTotal = Number(response?.cart_item?.line_total)
      const lineTotalLabel = Number.isFinite(lineTotal) ? formatPrice(lineTotal) : 'cart'

      setToastVariant('success')
      setToastMessage(`Added to cart successfully. Line total: ${lineTotalLabel}`)
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsAdding(false)
    }
  }

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <section className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-5 shadow-[0_8px_24px_rgba(0,0,128,0.08)] sm:p-7">
        <div className="mb-4">
          <Link
            to="/shop"
            className="inline-flex items-center rounded-md border border-[color:rgba(0,0,128,0.2)] px-3 py-1.5 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:border-[var(--orange)] hover:text-[var(--orange)]"
          >
            Back to Shop
          </Link>
        </div>

        {isLoading ? (
          <div className="rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.68)]">
            Loading product details...
          </div>
        ) : errorMessage ? (
          <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-6 text-sm text-red-700">
            {errorMessage}
          </div>
        ) : !product ? (
          <div className="rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.68)]">
            Product not available.
          </div>
        ) : (
          <div className="grid gap-6 lg:grid-cols-[1.1fr_1fr]">
            <div className="overflow-hidden rounded-xl bg-[rgba(0,0,128,0.04)]">
              <img
                src={resolveStaticImageURL(product)}
                alt={`${product.product_name} preview`}
                loading="lazy"
                className="h-full w-full object-cover"
              />
            </div>

            <div>
              <h1 className="text-3xl font-bold text-[var(--navy)]">{product.product_name}</h1>
              <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.72)]">
                {product.recurring_name || 'Standard Recurring Plan'}
              </p>

              <p className="mt-3 text-lg font-bold text-[var(--navy)]">
                {formatPrice(product.sales_price)}/{product.billing_period || 'Monthly'}
              </p>

              {Array.isArray(product.discounts) && product.discounts.length > 0 && (
                <p className="mt-2 text-xs text-emerald-700">
                  Discount available on this product. Final discount is applied at cart calculation.
                </p>
              )}

              <div className="mt-6 space-y-4">
                <div>
                  <label htmlFor="variant-select" className="block text-sm font-semibold text-[var(--navy)]">
                    Variant
                  </label>
                  {variants.length === 0 ? (
                    <p className="mt-1 text-sm text-[color:rgba(0,0,128,0.68)]">No variants available for this product.</p>
                  ) : (
                    <select
                      id="variant-select"
                      value={selectedVariantID}
                      onChange={(event) => setSelectedVariantID(event.target.value)}
                      className="mt-2 h-10 w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] bg-[var(--white)] px-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
                    >
                      <option value="">No variant (optional)</option>
                      {variants.map((variant) => {
                        const variantID = String(variant?.attribute_id ?? '')
                        const variantName = String(variant?.attribute_name ?? '').trim() || 'Variant'
                        return (
                          <option key={variantID} value={variantID}>
                            {variantName}
                          </option>
                        )
                      })}
                    </select>
                  )}
                </div>

                <div>
                  <label className="block text-sm font-semibold text-[var(--navy)]">Quantity</label>
                  <div className="mt-2 inline-flex h-10 items-center overflow-hidden rounded-lg border border-[color:rgba(0,0,128,0.22)]">
                    <button
                      type="button"
                      onClick={() => setQuantity((previousQuantity) => Math.max(1, previousQuantity - 1))}
                      className="inline-flex h-full w-10 items-center justify-center text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.06)]"
                    >
                      -
                    </button>
                    <span className="inline-flex h-full min-w-10 items-center justify-center border-x border-[color:rgba(0,0,128,0.16)] px-3 text-sm font-semibold text-[var(--navy)]">
                      {quantity}
                    </span>
                    <button
                      type="button"
                      onClick={() => setQuantity((previousQuantity) => previousQuantity + 1)}
                      className="inline-flex h-full w-10 items-center justify-center text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.06)]"
                    >
                      +
                    </button>
                  </div>
                </div>
              </div>

              <div className="mt-6 rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-4 py-3 text-sm text-[var(--navy)]">
                <p>Base Price: <span className="font-semibold">{formatPrice(safeBasePrice)}</span></p>
                <p>Variant Extra Price (DB): <span className="font-semibold">{formatPrice(selectedVariantExtraPrice)}</span></p>
                <p>Estimated Discount: <span className="font-semibold">{formatPrice(estimatedDiscount)}</span></p>
                <p className="mt-1 text-base font-bold">Estimated Total: {formatPrice(estimatedLineTotal)}</p>
              </div>

              <button
                type="button"
                onClick={handleAddToCart}
                disabled={isAdding}
                className="mt-6 inline-flex h-11 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-[var(--white)] transition-colors duration-200 hover:bg-[#e65f00] disabled:cursor-not-allowed disabled:opacity-70"
              >
                {isAdding ? 'Adding to Cart...' : 'Add to Cart'}
              </button>

              {selectedVariant && (
                <p className="mt-3 text-xs text-[color:rgba(0,0,128,0.72)]">
                  Selected Variant: <span className="font-semibold text-[var(--navy)]">{selectedVariant.attribute_name}</span>
                </p>
              )}

              {selectedDiscountRules.length > 0 && (
                <p className="mt-2 text-xs text-[color:rgba(0,0,128,0.72)]">
                  Discount rules are fetched from database and applied on the server during cart calculation.
                </p>
              )}
            </div>
          </div>
        )}
      </section>
    </div>
  )
}

export default ShopProductDetailPage
