import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import MultiSelectDropdown from '../../../../components/common/MultiSelectDropdown'
import ToastMessage from '../../../../components/common/ToastMessage'
import { listProducts } from '../../../../services/productApi'
import { listRecurringPlans } from '../../../../services/recurringPlanApi'
import { getQuotationTemplateById, updateQuotationTemplate } from '../../../../services/quotationTemplateApi'

function Toggle({ checked, onChange, label, description }) {
  return (
    <label className="flex items-start justify-between gap-4 rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[rgba(0,0,128,0.02)] px-4 py-3">
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
        className={`relative inline-flex h-7 w-12 flex-none items-center rounded-full border transition-colors duration-200 ${
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

function formatSalesPrice(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '$0.00'
  }

  return `$${numericValue.toFixed(2)}`
}

function QuotationTemplateEditPage() {
  const { quotationId = '' } = useParams()

  const [lastForever, setLastForever] = useState(false)
  const [quotationValidityDays, setQuotationValidityDays] = useState('')
  const [recurringPlanID, setRecurringPlanID] = useState('')
  const [recurringPlans, setRecurringPlans] = useState([])
  const [products, setProducts] = useState([])
  const [productOptions, setProductOptions] = useState([])
  const [selectedProductIDs, setSelectedProductIDs] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectedProducts = useMemo(() => {
    if (!Array.isArray(selectedProductIDs) || selectedProductIDs.length === 0) {
      return []
    }

    const productLookup = new Map(
      (Array.isArray(products) ? products : []).map((product) => [String(product?.product_id ?? '').trim(), product])
    )

    return selectedProductIDs.map((productID) => {
      const normalizedProductID = String(productID ?? '').trim()
      const product = productLookup.get(normalizedProductID)

      return {
        product_id: normalizedProductID,
        product_name: String(product?.product_name ?? 'Unknown Product'),
        product_type: String(product?.product_type ?? '-'),
        sales_price: product?.sales_price,
      }
    })
  }, [products, selectedProductIDs])

  useEffect(() => {
    let isMounted = true

    const loadQuotationAndRecurringPlans = async () => {
      setIsLoading(true)
      try {
        const [recurringPlansResponse, productsResponse, quotationResponse] = await Promise.all([
          listRecurringPlans('', false),
          listProducts(''),
          getQuotationTemplateById(quotationId),
        ])

        if (!isMounted) {
          return
        }

        const recurringPlanItems = Array.isArray(recurringPlansResponse?.recurring_plans)
          ? recurringPlansResponse.recurring_plans
          : []
        const productRows = Array.isArray(productsResponse?.products) ? productsResponse.products : []
        const productItems = mapProductsToOptions(productRows)
        const quotation = quotationResponse?.quotation

        setRecurringPlans(recurringPlanItems)
        setProducts(productRows)
        setProductOptions(productItems)
        setLastForever(Boolean(quotation?.last_forever))
        setQuotationValidityDays(
          quotation?.quotation_validity_days !== null && quotation?.quotation_validity_days !== undefined
            ? String(quotation.quotation_validity_days)
            : ''
        )
        setRecurringPlanID(String(quotation?.recurring_plan_id ?? ''))
        const incomingProductIDs = Array.isArray(quotation?.products)
          ? quotation.products
            .map((product) => String(product?.product_id ?? '').trim())
            .filter((productID) => productID !== '')
          : []
        setSelectedProductIDs(Array.from(new Set(incomingProductIDs)))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadQuotationAndRecurringPlans()

    return () => {
      isMounted = false
    }
  }, [quotationId])

  const handleSubmit = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const normalizedRecurringPlanID = recurringPlanID.trim()
    if (!normalizedRecurringPlanID) {
      setToastVariant('error')
      setToastMessage('Recurring plan is required.')
      return
    }

    if (selectedProductIDs.length === 0) {
      setToastVariant('error')
      setToastMessage('At least one product must be selected.')
      return
    }

    let normalizedValidityDays = null
    if (!lastForever) {
      const parsedValidity = Number(quotationValidityDays)
      if (!Number.isInteger(parsedValidity) || parsedValidity <= 0) {
        setToastVariant('error')
        setToastMessage('Quotation Validity (in days) must be a whole number greater than zero.')
        return
      }

      normalizedValidityDays = parsedValidity
    }

    setIsSubmitting(true)
    try {
      const response = await updateQuotationTemplate(quotationId, {
        last_forever: lastForever,
        quotation_validity_days: normalizedValidityDays,
        recurring_plan_id: normalizedRecurringPlanID,
        product_ids: selectedProductIDs,
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Quotation updated successfully.')

      const updatedQuotation = response?.quotation
      if (updatedQuotation) {
        setLastForever(Boolean(updatedQuotation?.last_forever))
        setQuotationValidityDays(
          updatedQuotation?.quotation_validity_days !== null && updatedQuotation?.quotation_validity_days !== undefined
            ? String(updatedQuotation.quotation_validity_days)
            : ''
        )
        setRecurringPlanID(String(updatedQuotation?.recurring_plan_id ?? normalizedRecurringPlanID))
        const updatedProductIDs = Array.isArray(updatedQuotation?.products)
          ? updatedQuotation.products
            .map((product) => String(product?.product_id ?? '').trim())
            .filter((productID) => productID !== '')
          : selectedProductIDs
        setSelectedProductIDs(Array.from(new Set(updatedProductIDs)))
      }
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8 sm:p-10">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Edit Quotation Template</h1>
        <Link
          to="/admin/configurations/quotation-template"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Quotations
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Update quotation validity, linked recurring plan, and selected products.
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      {isLoading ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          Loading quotation details...
        </div>
      ) : (
        <form className="mt-6 space-y-6" onSubmit={handleSubmit} noValidate>
          <div className="space-y-2">
            <label htmlFor="recurring-plan" className="block text-sm font-semibold text-[var(--navy)]">
              Select Recurring Plan
            </label>
            <select
              id="recurring-plan"
              name="recurring-plan"
              value={recurringPlanID}
              onChange={(event) => setRecurringPlanID(event.target.value)}
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
            >
              <option value="">Select recurring plan</option>
              {recurringPlans.map((recurringPlan) => (
                <option key={recurringPlan.recurring_plan_id} value={recurringPlan.recurring_plan_id}>
                  {recurringPlan.recurring_name}
                </option>
              ))}
            </select>
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

            <div className="overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
              <div className="grid grid-cols-[1.5fr_1fr_1fr] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                <span>Product Name</span>
                <span>Product Type</span>
                <span>Sales Price</span>
              </div>

              {selectedProducts.length === 0 ? (
                <div className="px-4 py-6 text-sm text-[color:rgba(0,0,128,0.62)]">
                  No products selected yet.
                </div>
              ) : (
                <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
                  {selectedProducts.map((product) => (
                    <div
                      key={`selected-product-${product.product_id}`}
                      className="grid grid-cols-[1.5fr_1fr_1fr] gap-4 px-4 py-3 text-sm text-[var(--navy)]"
                    >
                      <span className="font-semibold">{product.product_name}</span>
                      <span>{product.product_type}</span>
                      <span>{formatSalesPrice(product.sales_price)}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <Toggle
            checked={lastForever}
            onChange={setLastForever}
            label="Last Forever"
            description="If enabled, quotation validity in days is not required."
          />

          {!lastForever ? (
            <div className="space-y-2">
              <label htmlFor="quotation-validity-days" className="block text-sm font-semibold text-[var(--navy)]">
                Quotation Validity (in days)
              </label>
              <input
                id="quotation-validity-days"
                name="quotation-validity-days"
                type="number"
                min="1"
                step="1"
                value={quotationValidityDays}
                onChange={(event) => setQuotationValidityDays(event.target.value)}
                placeholder="Enter validity in days"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
            </div>
          ) : null}

          <div className="flex flex-wrap items-center justify-end gap-3">
            <Link
              to="/admin/configurations/quotation-template"
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting ? 'Updating...' : 'Update Quotation'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default QuotationTemplateEditPage
