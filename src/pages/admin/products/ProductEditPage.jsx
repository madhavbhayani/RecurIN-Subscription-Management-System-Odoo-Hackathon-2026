import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import MultiSelectDropdown from '../../../components/common/MultiSelectDropdown'
import ToastMessage from '../../../components/common/ToastMessage'
import { listAttributes } from '../../../services/attributeApi'
import { listDiscounts } from '../../../services/discountApi'
import { getProductById, updateProduct } from '../../../services/productApi'
import { listTaxes } from '../../../services/taxApi'

const PRODUCT_TYPES = ['Service', 'Goods']
const CURRENCY_SYMBOL = '\u20b9'

function formatDecimalValue(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '0.00'
  }

  return numericValue.toFixed(2)
}

function buildTaxOptions(taxes) {
  return taxes.map((tax) => {
    const taxID = String(tax?.tax_id ?? '').trim()
    const taxName = String(tax?.tax_name ?? '').trim()
    const taxUnit = String(tax?.tax_computation_unit ?? '').trim()
    const taxValue = formatDecimalValue(tax?.tax_computation_value)

    let valueLabel = taxValue
    if (taxUnit === 'Percentage') {
      valueLabel = `${taxValue}%`
    }
    if (taxUnit === 'Fixed Price') {
      valueLabel = `${CURRENCY_SYMBOL} ${taxValue}`
    }

    return {
      value: taxID,
      label: `${taxName} (${valueLabel})`,
      taxName,
      valueLabel,
    }
  }).filter((taxOption) => taxOption.value !== '' && taxOption.taxName !== '')
}

function buildDiscountOptions(discounts) {
  return discounts.map((discount) => {
    const discountID = String(discount?.discount_id ?? '').trim()
    const discountName = String(discount?.discount_name ?? '').trim()
    const discountUnit = String(discount?.discount_unit ?? '').trim()
    const discountValue = formatDecimalValue(discount?.discount_value)

    let valueLabel = discountValue
    if (discountUnit === 'Percentage') {
      valueLabel = `${discountValue}%`
    }
    if (discountUnit === 'Fixed Price') {
      valueLabel = `${CURRENCY_SYMBOL} ${discountValue}`
    }

    return {
      value: discountID,
      label: `${discountName} (${valueLabel})`,
      discountName,
      valueLabel,
    }
  }).filter((discountOption) => discountOption.value !== '' && discountOption.discountName !== '')
}

function buildAttributeOptions(attributes) {
  const options = []
  const seenAttributeIDs = new Set()

  for (const attribute of attributes) {
    const attributeID = String(attribute?.attribute_id ?? '').trim()
    const attributeName = String(attribute?.attribute_name ?? '').trim()

    if (!attributeID || !attributeName || seenAttributeIDs.has(attributeID)) {
      continue
    }

    seenAttributeIDs.add(attributeID)
    options.push({
      value: attributeID,
      label: attributeName,
      attributeID,
      attributeName,
    })
  }

  return options
}

function ProductEditPage() {
  const { productId = '' } = useParams()

  const [productName, setProductName] = useState('')
  const [productType, setProductType] = useState('Service')
  const [salesPrice, setSalesPrice] = useState('')
  const [costPrice, setCostPrice] = useState('')
  const [availableTaxOptions, setAvailableTaxOptions] = useState([])
  const [availableDiscountOptions, setAvailableDiscountOptions] = useState([])
  const [availableAttributeOptions, setAvailableAttributeOptions] = useState([])
  const [selectedTaxIDs, setSelectedTaxIDs] = useState([])
  const [selectedDiscountIDs, setSelectedDiscountIDs] = useState([])
  const [selectedAttributeIDs, setSelectedAttributeIDs] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const taxOptionByID = useMemo(() => {
    return new Map(availableTaxOptions.map((option) => [option.value, option]))
  }, [availableTaxOptions])

  const discountOptionByID = useMemo(() => {
    return new Map(availableDiscountOptions.map((option) => [option.value, option]))
  }, [availableDiscountOptions])

  const attributeOptionByID = useMemo(() => {
    return new Map(availableAttributeOptions.map((option) => [option.value, option]))
  }, [availableAttributeOptions])

  const selectedTaxTooltips = useMemo(() => {
    return selectedTaxIDs
      .map((taxID) => taxOptionByID.get(taxID))
      .filter(Boolean)
  }, [selectedTaxIDs, taxOptionByID])

  const selectedDiscountTooltips = useMemo(() => {
    return selectedDiscountIDs
      .map((discountID) => discountOptionByID.get(discountID))
      .filter(Boolean)
  }, [selectedDiscountIDs, discountOptionByID])

  const selectedAttributeTooltips = useMemo(() => {
    return selectedAttributeIDs
      .map((attributeID) => attributeOptionByID.get(attributeID))
      .filter(Boolean)
  }, [selectedAttributeIDs, attributeOptionByID])

  useEffect(() => {
    let isMounted = true

    const loadDependenciesAndProduct = async () => {
      setIsLoading(true)
      try {
        const [taxesResponse, discountsResponse, attributesResponse, productResponse] = await Promise.all([
          listTaxes(''),
          listDiscounts(''),
          listAttributes(''),
          getProductById(productId),
        ])

        if (!isMounted) {
          return
        }

        const taxes = Array.isArray(taxesResponse?.taxes) ? taxesResponse.taxes : []
        const discounts = Array.isArray(discountsResponse?.discounts) ? discountsResponse.discounts : []
        const attributes = Array.isArray(attributesResponse?.attributes) ? attributesResponse.attributes : []
        const product = productResponse?.product

        setAvailableTaxOptions(buildTaxOptions(taxes))
        setAvailableDiscountOptions(buildDiscountOptions(discounts))
        setAvailableAttributeOptions(buildAttributeOptions(attributes))
        setProductName(String(product?.product_name ?? ''))

        const incomingProductType = String(product?.product_type ?? 'Service')
        setProductType(PRODUCT_TYPES.includes(incomingProductType) ? incomingProductType : 'Service')
        setSalesPrice(String(product?.sales_price ?? ''))
        setCostPrice(String(product?.cost_price ?? ''))

        const taxIDs = Array.isArray(product?.taxes)
          ? product.taxes
            .map((tax) => String(tax?.tax_id ?? '').trim())
            .filter((taxID) => taxID !== '')
          : []

        const discountIDs = Array.isArray(product?.discounts)
          ? product.discounts
            .map((discount) => String(discount?.discount_id ?? '').trim())
            .filter((discountID) => discountID !== '')
          : []

        const attributeIDs = Array.isArray(product?.variants)
          ? product.variants
            .map((variant) => String(variant?.attribute_id ?? '').trim())
            .filter((attributeID) => attributeID !== '')
          : []

        setSelectedTaxIDs(Array.from(new Set(taxIDs)))
        setSelectedDiscountIDs(Array.from(new Set(discountIDs)))
        setSelectedAttributeIDs(Array.from(new Set(attributeIDs)))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setToastVariant('error')
        setToastMessage(error.message)
        setAvailableTaxOptions([])
        setAvailableDiscountOptions([])
        setAvailableAttributeOptions([])
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    loadDependenciesAndProduct()

    return () => {
      isMounted = false
    }
  }, [productId])

  const handleSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    const normalizedProductName = productName.trim()
    if (!normalizedProductName) {
      setToastVariant('error')
      setToastMessage('Product name is required.')
      return
    }

    const normalizedSalesPriceText = String(salesPrice).trim()
    const normalizedSalesPrice = normalizedSalesPriceText === '' ? 0 : Number(normalizedSalesPriceText)
    if (!Number.isFinite(normalizedSalesPrice) || normalizedSalesPrice < 0) {
      setToastVariant('error')
      setToastMessage('Sales price must be a non-negative number.')
      return
    }

    const normalizedCostPriceText = String(costPrice).trim()
    const normalizedCostPrice = normalizedCostPriceText === '' ? 0 : Number(normalizedCostPriceText)
    if (!Number.isFinite(normalizedCostPrice) || normalizedCostPrice < 0) {
      setToastVariant('error')
      setToastMessage('Cost price must be a non-negative number.')
      return
    }

    const variantsPayload = []
    for (const attributeID of selectedAttributeIDs) {
      const option = attributeOptionByID.get(attributeID)
      if (!option) {
        setToastVariant('error')
        setToastMessage('One or more selected attributes are invalid. Please refresh and try again.')
        return
      }

      variantsPayload.push({
        attribute_id: option.attributeID,
      })
    }

    setIsSubmitting(true)
    try {
      const response = await updateProduct(productId, {
        product_name: normalizedProductName,
        product_type: productType,
        sales_price: Number(normalizedSalesPrice.toFixed(2)),
        cost_price: Number(normalizedCostPrice.toFixed(2)),
        tax_ids: selectedTaxIDs,
        discount_ids: selectedDiscountIDs,
        variants: variantsPayload,
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Product updated successfully.')

      const updatedProduct = response?.product
      if (updatedProduct) {
        setProductName(String(updatedProduct?.product_name ?? normalizedProductName))

        const updatedProductType = String(updatedProduct?.product_type ?? productType)
        setProductType(PRODUCT_TYPES.includes(updatedProductType) ? updatedProductType : productType)
        setSalesPrice(String(updatedProduct?.sales_price ?? normalizedSalesPrice))
        setCostPrice(String(updatedProduct?.cost_price ?? normalizedCostPrice))

        const updatedTaxIDs = Array.isArray(updatedProduct?.taxes)
          ? updatedProduct.taxes
            .map((tax) => String(tax?.tax_id ?? '').trim())
            .filter((taxID) => taxID !== '')
          : selectedTaxIDs

        const updatedDiscountIDs = Array.isArray(updatedProduct?.discounts)
          ? updatedProduct.discounts
            .map((discount) => String(discount?.discount_id ?? '').trim())
            .filter((discountID) => discountID !== '')
          : selectedDiscountIDs

        const updatedAttributeIDs = Array.isArray(updatedProduct?.variants)
          ? updatedProduct.variants
            .map((variant) => String(variant?.attribute_id ?? '').trim())
            .filter((attributeID) => attributeID !== '')
          : selectedAttributeIDs

        setSelectedTaxIDs(Array.from(new Set(updatedTaxIDs)))
        setSelectedDiscountIDs(Array.from(new Set(updatedDiscountIDs)))
        setSelectedAttributeIDs(Array.from(new Set(updatedAttributeIDs)))
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
      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">
          Edit Product
        </h1>
        <Link
          to="/admin/products"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Products
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Update product details, pricing, tax links, and variant mappings.
      </p>

      {isLoading ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          Loading product details...
        </div>
      ) : (
        <form className="mt-6 space-y-6" onSubmit={handleSubmit} noValidate>
          <div className="space-y-2">
            <label htmlFor="product-name" className="block text-sm font-semibold text-[var(--navy)]">
              Product Name
            </label>
            <input
              id="product-name"
              name="product-name"
              type="text"
              value={productName}
              onChange={(event) => setProductName(event.target.value)}
              placeholder="Enter product name"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <div className="space-y-2">
              <label htmlFor="product-type" className="block text-sm font-semibold text-[var(--navy)]">
                Product Type
              </label>
              <select
                id="product-type"
                name="product-type"
                value={productType}
                onChange={(event) => setProductType(event.target.value)}
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
              >
                {PRODUCT_TYPES.map((type) => (
                  <option key={type} value={type}>{type}</option>
                ))}
              </select>
            </div>

            <div className="space-y-2">
              <label htmlFor="sales-price" className="block text-sm font-semibold text-[var(--navy)]">
                Sales Price
              </label>
              <div className="relative">
                <input
                  id="sales-price"
                  name="sales-price"
                  type="number"
                  inputMode="decimal"
                  min="0"
                  step="0.01"
                  value={salesPrice}
                  onChange={(event) => setSalesPrice(event.target.value)}
                  placeholder="0.00"
                  className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 pr-10 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
                />
                <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-[color:rgba(0,0,128,0.7)]">
                  {CURRENCY_SYMBOL}
                </span>
              </div>
            </div>

            <div className="space-y-2">
              <label htmlFor="cost-price" className="block text-sm font-semibold text-[var(--navy)]">
                Cost
              </label>
              <div className="relative">
                <input
                  id="cost-price"
                  name="cost-price"
                  type="number"
                  inputMode="decimal"
                  min="0"
                  step="0.01"
                  value={costPrice}
                  onChange={(event) => setCostPrice(event.target.value)}
                  placeholder="0.00"
                  className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 pr-10 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
                />
                <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-[color:rgba(0,0,128,0.7)]">
                  {CURRENCY_SYMBOL}
                </span>
              </div>
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-3">
            <div className="space-y-2">
              <label className="block text-sm font-semibold text-[var(--navy)]">Taxes</label>
              <MultiSelectDropdown
                buttonLabel="Taxes"
                options={availableTaxOptions}
                selectedValues={selectedTaxIDs}
                onChange={setSelectedTaxIDs}
                placeholder="Select taxes"
                emptyMessage="No taxes available."
              />

              {selectedTaxTooltips.length > 0 ? (
                <div className="mt-2 space-y-2">
                  {selectedTaxTooltips.map((tax) => {
                    const tooltipLabel = `${tax.taxName} - ${tax.valueLabel}`
                    return (
                      <div
                        key={tax.value}
                        title={tooltipLabel}
                        className="rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-3 py-2 text-xs font-medium text-[var(--navy)]"
                      >
                        {tooltipLabel}
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className="text-xs text-[color:rgba(0,0,128,0.62)]">No taxes selected.</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-semibold text-[var(--navy)]">Discounts</label>
              <MultiSelectDropdown
                buttonLabel="Discounts"
                options={availableDiscountOptions}
                selectedValues={selectedDiscountIDs}
                onChange={setSelectedDiscountIDs}
                placeholder="Select discounts"
                emptyMessage="No discounts available."
              />

              {selectedDiscountTooltips.length > 0 ? (
                <div className="mt-2 space-y-2">
                  {selectedDiscountTooltips.map((discount) => {
                    const tooltipLabel = `${discount.discountName} - ${discount.valueLabel}`
                    return (
                      <div
                        key={discount.value}
                        title={tooltipLabel}
                        className="rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-3 py-2 text-xs font-medium text-[var(--navy)]"
                      >
                        {tooltipLabel}
                      </div>
                    )
                  })}
                </div>
              ) : (
                <p className="text-xs text-[color:rgba(0,0,128,0.62)]">No discounts selected.</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="block text-sm font-semibold text-[var(--navy)]">Product Variants</label>
              <MultiSelectDropdown
                buttonLabel="Variants"
                options={availableAttributeOptions}
                selectedValues={selectedAttributeIDs}
                onChange={setSelectedAttributeIDs}
                placeholder="Select attributes"
                emptyMessage="No attributes available."
              />

              {selectedAttributeTooltips.length > 0 ? (
                <div className="mt-2 space-y-2">
                  {selectedAttributeTooltips.map((attribute) => (
                    <div
                      key={attribute.value}
                      title={attribute.attributeName}
                      className="rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[rgba(0,0,128,0.03)] px-3 py-2 text-xs font-medium text-[var(--navy)]"
                    >
                      {attribute.attributeName}
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-[color:rgba(0,0,128,0.62)]">No attributes selected.</p>
              )}
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-3">
            <Link
              to="/admin/products"
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting ? 'Updating...' : 'Update Product'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default ProductEditPage
