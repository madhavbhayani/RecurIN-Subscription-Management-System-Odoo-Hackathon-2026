import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import MultiSelectDropdown from '../../../../components/common/MultiSelectDropdown'
import ToastMessage from '../../../../components/common/ToastMessage'
import { createDiscount } from '../../../../services/discountApi'
import { listProducts } from '../../../../services/productApi'

const DISCOUNT_UNITS = [
  {
    value: 'Percentage',
    label: 'Percentage (%)',
    symbol: '%',
  },
  {
    value: 'Fixed Price',
    label: 'Fixed Price (\u20b9)',
    symbol: '\u20b9',
  },
]

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

function Toggle({ checked, onChange, label, description }) {
  return (
    <label className="flex flex-col gap-3 rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[rgba(0,0,128,0.02)] px-4 py-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <p className="text-sm font-semibold text-[var(--navy)]">{label}</p>
        {description ? (
          <p className="mt-0.5 text-xs text-[color:rgba(0,0,128,0.66)]">{description}</p>
        ) : null}
      </div>

      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative inline-flex h-7 w-12 flex-none items-center self-end rounded-full border transition-colors duration-200 sm:self-auto ${
          checked
            ? 'border-[var(--orange)] bg-[var(--orange)]'
            : 'border-[color:rgba(0,0,128,0.24)] bg-white'
        }`}
      >
        <span
          className={`inline-block h-5 w-5 rounded-full bg-white shadow transition-transform duration-200 ${
            checked ? 'translate-x-6' : 'translate-x-1'
          }`}
        />
      </button>
    </label>
  )
}

function mapProductsToOptions(products) {
  if (!Array.isArray(products)) {
    return []
  }

  return products
    .map((product) => {
      const productID = String(product?.product_id ?? '').trim()
      const productName = String(product?.product_name ?? '').trim()
      if (!productID || !productName) {
        return null
      }

      return {
        value: productID,
        label: productName,
      }
    })
    .filter(Boolean)
}

function DiscountNewPage() {
  const [discountName, setDiscountName] = useState('')
  const [discountUnit, setDiscountUnit] = useState('Percentage')
  const [discountValue, setDiscountValue] = useState('')
  const [minimumPurchase, setMinimumPurchase] = useState('0')
  const [maximumPurchase, setMaximumPurchase] = useState('0')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [isLimit, setIsLimit] = useState(false)
  const [limitUsers, setLimitUsers] = useState('')
  const [status, setStatus] = useState('active')
  const [productOptions, setProductOptions] = useState([])
  const [selectedProductIDs, setSelectedProductIDs] = useState([])
  const [isLoadingProducts, setIsLoadingProducts] = useState(true)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectedDiscountUnit = useMemo(() => {
    return DISCOUNT_UNITS.find((unit) => unit.value === discountUnit) ?? DISCOUNT_UNITS[0]
  }, [discountUnit])

  useEffect(() => {
    let isMounted = true

    const loadProducts = async () => {
      setIsLoadingProducts(true)
      try {
        const response = await listProducts('')
        if (!isMounted) {
          return
        }

        setProductOptions(mapProductsToOptions(response?.products))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setProductOptions([])
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoadingProducts(false)
        }
      }
    }

    loadProducts()

    return () => {
      isMounted = false
    }
  }, [])

  const handleSubmit = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const normalizedDiscountName = discountName.trim()
    if (!normalizedDiscountName) {
      setToastVariant('error')
      setToastMessage('Discount name is required.')
      return
    }

    const parsedDiscountValue = Number(discountValue)
    if (!Number.isFinite(parsedDiscountValue) || parsedDiscountValue <= 0) {
      setToastVariant('error')
      setToastMessage('Discount value must be greater than zero.')
      return
    }
    if (discountUnit === 'Percentage' && parsedDiscountValue > 100) {
      setToastVariant('error')
      setToastMessage('Percentage discount value cannot be greater than 100.')
      return
    }

    const parsedMinimumPurchase = Number(minimumPurchase)
    if (!Number.isFinite(parsedMinimumPurchase) || parsedMinimumPurchase < 0) {
      setToastVariant('error')
      setToastMessage('Minimum purchase cannot be negative.')
      return
    }

    const parsedMaximumPurchase = Number(maximumPurchase)
    if (!Number.isFinite(parsedMaximumPurchase) || parsedMaximumPurchase < 0) {
      setToastVariant('error')
      setToastMessage('Maximum purchase cannot be negative.')
      return
    }
    if (parsedMaximumPurchase < parsedMinimumPurchase) {
      setToastVariant('error')
      setToastMessage('Maximum purchase must be greater than or equal to minimum purchase.')
      return
    }

    if (!startDate || !endDate) {
      setToastVariant('error')
      setToastMessage('Start date and end date are required.')
      return
    }
    if (new Date(endDate).getTime() < new Date(startDate).getTime()) {
      setToastVariant('error')
      setToastMessage('End date cannot be before start date.')
      return
    }

    if (selectedProductIDs.length === 0) {
      setToastVariant('error')
      setToastMessage('At least one product must be selected.')
      return
    }

    let parsedLimitUsers = null
    if (isLimit) {
      const parsedLimit = Number(limitUsers)
      if (!Number.isInteger(parsedLimit) || parsedLimit <= 0) {
        setToastVariant('error')
        setToastMessage('Limit users must be a whole number greater than zero.')
        return
      }
      parsedLimitUsers = parsedLimit
    }

    setIsSubmitting(true)
    try {
      const response = await createDiscount({
        discount_name: normalizedDiscountName,
        discount_unit: discountUnit,
        discount_value: Number(parsedDiscountValue.toFixed(2)),
        minimum_purchase: Number(parsedMinimumPurchase.toFixed(2)),
        maximum_purchase: Number(parsedMaximumPurchase.toFixed(2)),
        start_date: startDate,
        end_date: endDate,
        is_limit: isLimit,
        limit_users: parsedLimitUsers,
        is_active: status === 'active',
        product_ids: selectedProductIDs,
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Discount created successfully.')

      setDiscountName('')
      setDiscountUnit('Percentage')
      setDiscountValue('')
      setMinimumPurchase('0')
      setMaximumPurchase('0')
      setStartDate('')
      setEndDate('')
      setIsLimit(false)
      setLimitUsers('')
      setStatus('active')
      setSelectedProductIDs([])
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-5xl rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-5 sm:p-8 lg:p-10">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">New Discount</h1>
        <Link
          to="/admin/configurations/discount"
          className="inline-flex h-10 w-full items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)] sm:w-auto"
        >
          Back to Discounts
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Configure discount amount, product applicability, purchase range, date range, user limit, and status.
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      {isLoadingProducts ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          Loading products...
        </div>
      ) : (
        <form className="mt-6 space-y-6" onSubmit={handleSubmit} noValidate>
          <div className="space-y-2">
            <label htmlFor="discount-name" className="block text-sm font-semibold text-[var(--navy)]">
              Discount Name
            </label>
            <input
              id="discount-name"
              name="discount-name"
              type="text"
              value={discountName}
              onChange={(event) => setDiscountName(event.target.value)}
              placeholder="Enter discount name"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="discount-unit" className="block text-sm font-semibold text-[var(--navy)]">
                Discount Unit
              </label>
              <select
                id="discount-unit"
                name="discount-unit"
                value={discountUnit}
                onChange={(event) => setDiscountUnit(event.target.value)}
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              >
                {DISCOUNT_UNITS.map((unit) => (
                  <option key={unit.value} value={unit.value}>{unit.label}</option>
                ))}
              </select>
            </div>

            <div className="space-y-2">
              <label htmlFor="discount-value" className="block text-sm font-semibold text-[var(--navy)]">
                Discount Value
              </label>
              <div className="relative">
                <input
                  id="discount-value"
                  name="discount-value"
                  type="number"
                  inputMode="decimal"
                  min="0"
                  max={discountUnit === 'Percentage' ? 100 : undefined}
                  step="0.01"
                  value={discountValue}
                  onChange={(event) => setDiscountValue(event.target.value)}
                  placeholder="0.00"
                  className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 pr-10 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
                />
                <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-[color:rgba(0,0,128,0.7)]">
                  {selectedDiscountUnit.symbol}
                </span>
              </div>
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="minimum-purchase" className="block text-sm font-semibold text-[var(--navy)]">
                Minimum Purchase
              </label>
              <input
                id="minimum-purchase"
                name="minimum-purchase"
                type="number"
                inputMode="decimal"
                min="0"
                step="0.01"
                value={minimumPurchase}
                onChange={(event) => setMinimumPurchase(event.target.value)}
                placeholder="0.00"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="maximum-purchase" className="block text-sm font-semibold text-[var(--navy)]">
                Maximum Purchase
              </label>
              <input
                id="maximum-purchase"
                name="maximum-purchase"
                type="number"
                inputMode="decimal"
                min="0"
                step="0.01"
                value={maximumPurchase}
                onChange={(event) => setMaximumPurchase(event.target.value)}
                placeholder="0.00"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="start-date" className="block text-sm font-semibold text-[var(--navy)]">
                Start Date
              </label>
              <input
                id="start-date"
                name="start-date"
                type="date"
                value={startDate}
                onChange={(event) => setStartDate(event.target.value)}
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              />
            </div>

            <div className="space-y-2">
              <label htmlFor="end-date" className="block text-sm font-semibold text-[var(--navy)]">
                End Date
              </label>
              <input
                id="end-date"
                name="end-date"
                type="date"
                value={endDate}
                onChange={(event) => setEndDate(event.target.value)}
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-semibold text-[var(--navy)]">Products</label>
            <MultiSelectDropdown
              buttonLabel="Products"
              options={productOptions}
              selectedValues={selectedProductIDs}
              onChange={setSelectedProductIDs}
              placeholder="Select products"
              emptyMessage="No products available."
            />
            <p className="text-xs text-[color:rgba(0,0,128,0.62)]">
              Selected products: {selectedProductIDs.length}
            </p>
          </div>

          <Toggle
            checked={isLimit}
            onChange={setIsLimit}
            label="Is Limit"
            description="Enable user limit for discount usage."
          />

          {isLimit ? (
            <div className="space-y-2">
              <label htmlFor="limit-users" className="block text-sm font-semibold text-[var(--navy)]">
                Limit Number For Users
              </label>
              <input
                id="limit-users"
                name="limit-users"
                type="number"
                inputMode="numeric"
                min="1"
                step="1"
                value={limitUsers}
                onChange={(event) => setLimitUsers(event.target.value)}
                placeholder="Enter user limit (e.g. 50)"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>
          ) : null}

          <div className="space-y-2">
            <label htmlFor="status" className="block text-sm font-semibold text-[var(--navy)]">
              Status
            </label>
            <select
              id="status"
              name="status"
              value={status}
              onChange={(event) => setStatus(event.target.value)}
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
            >
              {STATUS_OPTIONS.map((statusOption) => (
                <option key={statusOption.value} value={statusOption.value}>{statusOption.label}</option>
              ))}
            </select>
          </div>

          <div className="flex flex-col-reverse items-stretch gap-3 sm:flex-row sm:items-center sm:justify-end">
            <Link
              to="/admin/configurations/discount"
              className="inline-flex h-10 w-full items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)] sm:w-auto"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-10 w-full items-center justify-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70 sm:w-auto"
            >
              {isSubmitting ? 'Saving...' : 'Save Discount'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default DiscountNewPage
