import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import Pagination from '../../components/common/Pagination'
import { listProductsPublic } from '../../services/productApi'

const CURRENCY_SYMBOL = '$'
const SHOP_PAGE_SIZE = 30

const PRICE_RANGE_OPTIONS = [
  {
    value: 'all',
    label: 'All Prices',
    matches: () => true,
  },
  {
    value: 'under-1000',
    label: 'Under $1,000',
    matches: (price) => price < 1000,
  },
  {
    value: '1000-5000',
    label: '$1,000 - $5,000',
    matches: (price) => price >= 1000 && price <= 5000,
  },
  {
    value: '5000-plus',
    label: 'Above $5,000',
    matches: (price) => price > 5000,
  },
]

function formatPrice(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return `${CURRENCY_SYMBOL}0.00`
  }

  return `${CURRENCY_SYMBOL}${numericValue.toFixed(2)}`
}

function resolveStaticImageURL(product, index) {
  const rawSeed = String(product?.product_id ?? product?.product_name ?? index + 1).trim()
  const encodedSeed = encodeURIComponent(rawSeed || String(index + 1))
  return `https://picsum.photos/seed/recurin-shop-${encodedSeed}/720/480`
}

function ProductCardSkeleton({ index }) {
  return (
    <div
      key={`product-card-skeleton-${index}`}
      className="overflow-hidden rounded-2xl border border-[color:rgba(0,0,128,0.12)] bg-[var(--white)]"
    >
      <div className="h-40 w-full animate-pulse bg-gradient-to-r from-[rgba(0,0,128,0.08)] via-[rgba(0,0,128,0.04)] to-[rgba(0,0,128,0.08)]" />
      <div className="space-y-3 p-4">
        <div className="h-4 w-3/5 animate-pulse rounded bg-[rgba(0,0,128,0.12)]" />
        <div className="h-3 w-2/5 animate-pulse rounded bg-[rgba(0,0,128,0.08)]" />
        <div className="h-3 w-1/3 animate-pulse rounded bg-[rgba(0,0,128,0.12)]" />
      </div>
    </div>
  )
}

function ShopPage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [products, setProducts] = useState([])
  const [currentPage, setCurrentPage] = useState(1)
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('all')
  const [selectedPriceRange, setSelectedPriceRange] = useState('all')
  const [sortBy, setSortBy] = useState('price-asc')

  useEffect(() => {
    const debounceTimer = window.setTimeout(() => {
      setSearchTerm(searchInput.trim())
      setCurrentPage(1)
    }, 300)

    return () => {
      window.clearTimeout(debounceTimer)
    }
  }, [searchInput])

  useEffect(() => {
    let isMounted = true

    const fetchProducts = async () => {
      setIsLoading(true)
      setErrorMessage('')

      try {
        const response = await listProductsPublic(searchTerm)
        if (!isMounted) {
          return
        }

        setProducts(Array.isArray(response?.products) ? response.products : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setProducts([])
        setErrorMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchProducts()

    return () => {
      isMounted = false
    }
  }, [searchTerm])

  const availableCategories = useMemo(() => {
    const categories = new Set()

    products.forEach((product) => {
      const category = String(product?.product_type ?? '').trim()
      if (category) {
        categories.add(category)
      }
    })

    return Array.from(categories)
  }, [products])

  useEffect(() => {
    if (selectedCategory === 'all') {
      return
    }

    if (!availableCategories.includes(selectedCategory)) {
      setSelectedCategory('all')
    }
  }, [availableCategories, selectedCategory])

  const selectedPriceRule = useMemo(() => {
    return PRICE_RANGE_OPTIONS.find((option) => option.value === selectedPriceRange) ?? PRICE_RANGE_OPTIONS[0]
  }, [selectedPriceRange])

  const visibleProducts = useMemo(() => {
    const filteredProducts = products.filter((product) => {
      const category = String(product?.product_type ?? '').trim()
      const matchesCategory = selectedCategory === 'all' || category === selectedCategory

      const price = Number(product?.sales_price)
      const safePrice = Number.isFinite(price) ? price : 0
      const matchesPriceRange = selectedPriceRule.matches(safePrice)

      return matchesCategory && matchesPriceRange
    })

    filteredProducts.sort((firstProduct, secondProduct) => {
      const firstPrice = Number(firstProduct?.sales_price)
      const secondPrice = Number(secondProduct?.sales_price)

      const safeFirstPrice = Number.isFinite(firstPrice) ? firstPrice : 0
      const safeSecondPrice = Number.isFinite(secondPrice) ? secondPrice : 0

      if (sortBy === 'price-desc') {
        return safeSecondPrice - safeFirstPrice
      }
      if (sortBy === 'name-asc') {
        return String(firstProduct?.product_name ?? '').localeCompare(String(secondProduct?.product_name ?? ''))
      }

      return safeFirstPrice - safeSecondPrice
    })

    return filteredProducts
  }, [products, selectedCategory, selectedPriceRule, sortBy])

  useEffect(() => {
    setCurrentPage(1)
  }, [selectedCategory, selectedPriceRange, sortBy])

  const paginationState = useMemo(() => {
    const totalPages = visibleProducts.length > 0
      ? Math.ceil(visibleProducts.length / SHOP_PAGE_SIZE)
      : 0
    const safeCurrentPage = totalPages > 0
      ? Math.min(Math.max(currentPage, 1), totalPages)
      : 1
    const startIndex = (safeCurrentPage - 1) * SHOP_PAGE_SIZE

    return {
      totalPages,
      safeCurrentPage,
      items: visibleProducts.slice(startIndex, startIndex + SHOP_PAGE_SIZE),
    }
  }, [visibleProducts, currentPage])

  useEffect(() => {
    if (paginationState.totalPages > 0 && currentPage > paginationState.totalPages) {
      setCurrentPage(paginationState.totalPages)
    }
  }, [currentPage, paginationState.totalPages])

  return (
    <div className="w-full px-4 py-8 sm:px-6 lg:px-8">
      <div className="overflow-hidden rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] shadow-[0_8px_24px_rgba(0,0,128,0.08)]">
        <section className="border-b border-[color:rgba(0,0,128,0.1)] px-5 py-5 sm:px-7">
          <h1 className="text-3xl font-bold text-[var(--navy)] sm:text-4xl">All Products</h1>
          <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)] sm:text-base">
            Browse products and open a dedicated detail view for quantity, variants, and cart actions.
          </p>
        </section>

        <div className="grid gap-0 lg:grid-cols-[220px_1fr]">
          <aside className="border-b border-[color:rgba(0,0,128,0.1)] px-5 py-5 lg:border-b-0 lg:border-r lg:px-5">
            <div>
              <h2 className="text-sm font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">Category</h2>
              <div className="mt-3 space-y-2 text-sm text-[var(--navy)]">
                <label className="flex cursor-pointer items-center gap-2">
                  <input
                    type="radio"
                    name="shop-category"
                    value="all"
                    checked={selectedCategory === 'all'}
                    onChange={() => setSelectedCategory('all')}
                    className="h-4 w-4 accent-[var(--orange)]"
                  />
                  <span>All</span>
                </label>

                {availableCategories.map((category) => (
                  <label key={category} className="flex cursor-pointer items-center gap-2">
                    <input
                      type="radio"
                      name="shop-category"
                      value={category}
                      checked={selectedCategory === category}
                      onChange={() => setSelectedCategory(category)}
                      className="h-4 w-4 accent-[var(--orange)]"
                    />
                    <span>{category}</span>
                  </label>
                ))}
              </div>
            </div>

            <div className="mt-7">
              <h2 className="text-sm font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">Price Range</h2>
              <div className="mt-3 space-y-2 text-sm text-[var(--navy)]">
                {PRICE_RANGE_OPTIONS.map((priceOption) => (
                  <label key={priceOption.value} className="flex cursor-pointer items-center gap-2">
                    <input
                      type="radio"
                      name="shop-price-range"
                      value={priceOption.value}
                      checked={selectedPriceRange === priceOption.value}
                      onChange={() => setSelectedPriceRange(priceOption.value)}
                      className="h-4 w-4 accent-[var(--orange)]"
                    />
                    <span>{priceOption.label}</span>
                  </label>
                ))}
              </div>
            </div>
          </aside>

          <section className="px-5 py-5 sm:px-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <input
                type="search"
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                placeholder="Search"
                className="h-10 w-full rounded-lg border border-[color:rgba(0,0,128,0.18)] px-4 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.48)] focus:border-[var(--orange)] sm:max-w-sm"
              />

              <label className="inline-flex items-center gap-2 text-sm font-medium text-[var(--navy)]">
                <span>Sort By:</span>
                <select
                  value={sortBy}
                  onChange={(event) => setSortBy(event.target.value)}
                  className="h-10 rounded-lg border border-[color:rgba(0,0,128,0.18)] bg-[var(--white)] px-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
                >
                  <option value="price-asc">Price (Low to High)</option>
                  <option value="price-desc">Price (High to Low)</option>
                  <option value="name-asc">Name (A-Z)</option>
                </select>
              </label>
            </div>

            {errorMessage && (
              <p className="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {errorMessage}
              </p>
            )}

            {isLoading ? (
              <div className="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                {Array.from({ length: 8 }, (_, index) => (
                  <ProductCardSkeleton key={`skeleton-${index}`} index={index} />
                ))}
              </div>
            ) : visibleProducts.length === 0 ? (
              <div className="mt-5 rounded-xl border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.02)] px-4 py-8 text-center text-sm text-[color:rgba(0,0,128,0.72)]">
                No products match your selected filters.
              </div>
            ) : (
              <div className="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                {paginationState.items.map((product, index) => {
                  const productID = String(product?.product_id ?? '').trim()
                  const productName = String(product?.product_name ?? '').trim() || 'Untitled Product'
                  const productType = String(product?.product_type ?? '').trim() || 'Subscription Product'
                  const recurringName = String(product?.recurring_name ?? '').trim() || 'Recurring subscription product'
                  const billingPeriod = String(product?.billing_period ?? '').trim() || 'Monthly'

                  return (
                    <article
                      key={productID || `${productName}-${index}`}
                      className="overflow-hidden rounded-2xl border border-[color:rgba(0,0,128,0.12)] bg-[var(--white)] transition-all duration-300 hover:-translate-y-0.5 hover:border-[color:rgba(255,107,0,0.4)] hover:shadow-[0_12px_26px_rgba(0,0,128,0.09)]"
                    >
                      <Link to={`/shop/${productID}/`} className="block">
                        <div className="aspect-[4/3] overflow-hidden bg-[rgba(0,0,128,0.05)]">
                          <img
                            src={resolveStaticImageURL(product, index)}
                            alt={`${productName} preview`}
                            loading="lazy"
                            className="h-full w-full object-cover transition-transform duration-500 hover:scale-105"
                          />
                        </div>

                        <div className="p-4">
                          <div className="flex items-start justify-between gap-2">
                            <div className="min-w-0">
                              <h3 className="truncate text-sm font-bold text-[var(--navy)]">{productName}</h3>
                              <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.68)]">{productType}</p>
                            </div>

                            <div className="text-right">
                              <p className="text-sm font-bold text-[var(--navy)]">{formatPrice(product?.sales_price)}</p>
                              <p className="text-[11px] font-medium uppercase tracking-[0.04em] text-[color:rgba(0,0,128,0.62)]">
                                /{billingPeriod}
                              </p>
                            </div>
                          </div>

                          <p className="mt-2 text-xs text-[color:rgba(0,0,128,0.7)]">{recurringName}</p>

                          <span className="mt-3 inline-flex rounded-md border border-[color:rgba(0,0,128,0.2)] px-2 py-1 text-[11px] font-semibold text-[var(--navy)]">
                            View Details
                          </span>
                        </div>
                      </Link>
                    </article>
                  )
                })}
              </div>
            )}

            {!isLoading && !errorMessage && visibleProducts.length > 0 && (
              <Pagination
                currentPage={paginationState.safeCurrentPage}
                totalPages={paginationState.totalPages}
                onPageChange={setCurrentPage}
              />
            )}
          </section>
        </div>
      </div>
    </div>
  )
}

export default ShopPage
