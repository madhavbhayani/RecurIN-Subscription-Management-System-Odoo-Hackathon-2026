import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import MultiSelectDropdown from '../../../components/common/MultiSelectDropdown'
import ToastMessage from '../../../components/common/ToastMessage'
import { listAttributes } from '../../../services/attributeApi'
import { listPaymentTerms } from '../../../services/paymentTermApi'
import { getProductById, listProducts } from '../../../services/productApi'
import { getQuotationTemplateById, listQuotationTemplates } from '../../../services/quotationTemplateApi'
import { listRecurringPlans } from '../../../services/recurringPlanApi'
import { createSubscription, getSubscriptionById, updateSubscription } from '../../../services/subscriptionApi'
import { listCustomerUsers } from '../../../services/userApi'

const STATUS_OPTIONS = [
  'Draft',
  'Quotation Sent',
  'Active',
  'Confirmed',
]

const INFO_TAB_ORDER = 'order'
const INFO_TAB_OTHER = 'other'

function formatCurrency(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '\u20b9 0.00'
  }

  return `\u20b9 ${numericValue.toFixed(2)}`
}

function roundToTwo(value) {
  return Math.round(value * 100) / 100
}

function getTodayDateISO() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
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

function buildTemplateLabel(template) {
  const recurringPlanName = String(template?.recurring_plan_name ?? '').trim()
  if (!recurringPlanName) {
    return 'Quotation Template'
  }

  return recurringPlanName
}

function buildCustomerLabel(customer) {
  const customerName = String(customer?.name ?? '').trim()
  const customerEmail = String(customer?.email ?? '').trim()

  if (!customerName && !customerEmail) {
    return ''
  }

  if (!customerEmail) {
    return customerName
  }

  return `${customerName} (${customerEmail})`
}

function buildAttributeValuesByAttributeID(attributes) {
  const valuesByAttributeID = {}
  if (!Array.isArray(attributes)) {
    return valuesByAttributeID
  }

  attributes.forEach((attribute) => {
    const attributeID = String(attribute?.attribute_id ?? '').trim()
    if (!attributeID) {
      return
    }

    const values = Array.isArray(attribute?.values) ? attribute.values : []
    valuesByAttributeID[attributeID] = values
      .map((value) => {
        const attributeValueID = String(value?.attribute_value_id ?? '').trim()
        const attributeValue = String(value?.attribute_value ?? '').trim()
        const defaultExtraPrice = Number(value?.default_extra_price ?? 0)

        if (!attributeValueID || !attributeValue) {
          return null
        }

        return {
          attribute_value_id: attributeValueID,
          attribute_value: attributeValue,
          default_extra_price: Number.isFinite(defaultExtraPrice) ? Math.max(defaultExtraPrice, 0) : 0,
        }
      })
      .filter(Boolean)
  })

  return valuesByAttributeID
}

function getVariantValueOptionsForProduct(productDetails, attributeValuesByAttributeID) {
  const productVariants = Array.isArray(productDetails?.variants) ? productDetails.variants : []
  const options = []

  productVariants.forEach((variant) => {
    const attributeID = String(variant?.attribute_id ?? '').trim()
    const attributeName = String(variant?.attribute_name ?? '').trim()
    if (!attributeID || !attributeName) {
      return
    }

    const attributeValues = Array.isArray(attributeValuesByAttributeID[attributeID])
      ? attributeValuesByAttributeID[attributeID]
      : []

    attributeValues.forEach((attributeValue) => {
      const variantValueID = String(attributeValue?.attribute_value_id ?? '').trim()
      const variantLabel = String(attributeValue?.attribute_value ?? '').trim()
      const defaultExtraPrice = Number(attributeValue?.default_extra_price ?? 0)
      if (!variantValueID || !variantLabel) {
        return
      }

      const normalizedExtraPrice = Number.isFinite(defaultExtraPrice) ? Math.max(defaultExtraPrice, 0) : 0
      const priceSuffix = normalizedExtraPrice > 0 ? ` (+\u20b9 ${normalizedExtraPrice.toFixed(2)})` : ' (+\u20b9 0.00)'

      options.push({
        value: variantValueID,
        label: `${attributeName}: ${variantLabel}${priceSuffix}`,
        attribute_id: attributeID,
        attribute_name: attributeName,
        attribute_value: variantLabel,
        extra_price: normalizedExtraPrice,
      })
    })
  })

  return options
}

function calculatePaymentTermAmount(grandTotal, paymentTerm) {
  const normalizedGrandTotal = Number.isFinite(grandTotal) ? Math.max(grandTotal, 0) : 0
  if (normalizedGrandTotal === 0 || !paymentTerm) {
    return 0
  }

  const dueValue = Number(paymentTerm?.due_value ?? 0)
  if (!Number.isFinite(dueValue) || dueValue <= 0) {
    return 0
  }

  const dueUnit = String(paymentTerm?.due_unit ?? '').trim()
  if (dueUnit === 'Percentage') {
    return roundToTwo((normalizedGrandTotal * dueValue) / 100)
  }

  if (dueUnit === 'Fixed Price') {
    return roundToTwo(Math.min(dueValue, normalizedGrandTotal))
  }

  return 0
}

function calculateLineAmounts(product, quantity, perUnitVariantExtra = 0) {
  const unitPrice = Number(product?.sales_price ?? 0)
  const normalizedVariantExtra = Number.isFinite(perUnitVariantExtra) ? Math.max(perUnitVariantExtra, 0) : 0
  const normalizedUnitPrice = Number.isFinite(unitPrice) ? unitPrice + normalizedVariantExtra : normalizedVariantExtra
  const normalizedQuantity = Number.isInteger(quantity) && quantity > 0 ? quantity : 1

  const discounts = Array.isArray(product?.discounts) ? product.discounts : []
  const taxes = Array.isArray(product?.taxes) ? product.taxes : []

  let perUnitDiscount = 0
  discounts.forEach((discount) => {
    const discountUnit = String(discount?.discount_unit ?? '').trim()
    const discountValue = Number(discount?.discount_value ?? 0)
    if (!Number.isFinite(discountValue) || discountValue <= 0) {
      return
    }

    if (discountUnit === 'Percentage') {
      perUnitDiscount += normalizedUnitPrice * (discountValue / 100)
      return
    }

    if (discountUnit === 'Fixed Price') {
      perUnitDiscount += discountValue
    }
  })

  if (perUnitDiscount > normalizedUnitPrice) {
    perUnitDiscount = normalizedUnitPrice
  }

  const lineSubtotal = normalizedUnitPrice * normalizedQuantity
  const lineDiscount = perUnitDiscount * normalizedQuantity
  const taxableAmount = Math.max(lineSubtotal - lineDiscount, 0)

  let lineTax = 0
  taxes.forEach((tax) => {
    const taxUnit = String(tax?.tax_computation_unit ?? '').trim()
    const taxValue = Number(tax?.tax_computation_value ?? 0)
    if (!Number.isFinite(taxValue) || taxValue < 0) {
      return
    }

    if (taxUnit === 'Percentage') {
      lineTax += taxableAmount * (taxValue / 100)
      return
    }

    if (taxUnit === 'Fixed Price') {
      lineTax += taxValue * normalizedQuantity
    }
  })

  return {
    unitPrice: roundToTwo(normalizedUnitPrice),
    variantExtraAmount: roundToTwo(normalizedVariantExtra * normalizedQuantity),
    discountAmount: roundToTwo(lineDiscount),
    taxAmount: roundToTwo(lineTax),
    totalAmount: roundToTwo(taxableAmount + lineTax),
  }
}

function SubscriptionNewPage() {
  const { subscriptionId = '' } = useParams()
  const isEditMode = String(subscriptionId).trim() !== ''

  const [customerSearchInput, setCustomerSearchInput] = useState('')
  const [customerSearchTerm, setCustomerSearchTerm] = useState('')
  const [customerID, setCustomerID] = useState('')
  const [customerOptions, setCustomerOptions] = useState([])
  const [isCustomerDropdownOpen, setIsCustomerDropdownOpen] = useState(false)
  const [activeInfoTab, setActiveInfoTab] = useState(INFO_TAB_ORDER)
  const [nextInvoiceDate, setNextInvoiceDate] = useState('')
  const [recurringPlanID, setRecurringPlanID] = useState('')
  const [paymentTermID, setPaymentTermID] = useState('')
  const [quotationID, setQuotationID] = useState('')
  const [status, setStatus] = useState('Draft')
  const [salesPerson, setSalesPerson] = useState('')
  const [startDate, setStartDate] = useState('')
  const [paymentMethod, setPaymentMethod] = useState('')
  const [isPaymentMode, setIsPaymentMode] = useState('')
  const [recurringPlans, setRecurringPlans] = useState([])
  const [paymentTerms, setPaymentTerms] = useState([])
  const [attributeValuesByAttributeID, setAttributeValuesByAttributeID] = useState({})
  const [quotationTemplates, setQuotationTemplates] = useState([])
  const [productOptions, setProductOptions] = useState([])
  const [lineItems, setLineItems] = useState([])
  const [productDetailsByID, setProductDetailsByID] = useState({})
  const [isLoadingDependencies, setIsLoadingDependencies] = useState(true)
  const [isLoadingSubscription, setIsLoadingSubscription] = useState(false)
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(true)
  const [isLoadingTemplateProducts, setIsLoadingTemplateProducts] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const customerDropdownRef = useRef(null)

  const selectedProductIDs = useMemo(
    () => lineItems.map((lineItem) => lineItem.product_id),
    [lineItems]
  )

  const selectedRecurringPlan = useMemo(
    () => recurringPlans.find((recurringPlan) => recurringPlan.recurring_plan_id === recurringPlanID) ?? null,
    [recurringPlanID, recurringPlans]
  )

  const selectedPaymentTerm = useMemo(
    () => paymentTerms.find((paymentTerm) => paymentTerm.payment_term_id === paymentTermID) ?? null,
    [paymentTermID, paymentTerms]
  )

  const isStatusConfirmed = status === 'Confirmed'
  const isPaymentTermEnabled = status !== 'Draft'

  const lineItemsWithComputedAmounts = useMemo(() => {
    return lineItems.map((lineItem) => {
      const productDetails = productDetailsByID[lineItem.product_id]
      const variantOptions = getVariantValueOptionsForProduct(productDetails, attributeValuesByAttributeID)
      const variantOptionByID = new Map(variantOptions.map((option) => [option.value, option]))

      const normalizedSelectedVariantValueIDs = Array.isArray(lineItem.selected_variant_value_ids)
        ? lineItem.selected_variant_value_ids
          .map((variantValueID) => String(variantValueID ?? '').trim())
          .filter((variantValueID) => variantValueID !== '')
        : []

      const selectedVariants = normalizedSelectedVariantValueIDs
        .map((variantValueID) => variantOptionByID.get(variantValueID))
        .filter(Boolean)

      const perUnitVariantExtra = selectedVariants.reduce((total, variant) => total + Number(variant?.extra_price ?? 0), 0)
      const amounts = calculateLineAmounts(productDetails, lineItem.quantity, perUnitVariantExtra)

      return {
        ...lineItem,
        product_name: String(productDetails?.product_name ?? 'Unknown Product'),
        variant_options: variantOptions,
        selected_variant_value_ids: selectedVariants.map((variant) => variant.value),
        selected_variants: selectedVariants,
        unit_price: amounts.unitPrice,
        variant_extra_amount: amounts.variantExtraAmount,
        discount_amount: amounts.discountAmount,
        tax_amount: amounts.taxAmount,
        total_amount: amounts.totalAmount,
      }
    })
  }, [lineItems, productDetailsByID, attributeValuesByAttributeID])

  const grandTotal = useMemo(() => {
    return lineItemsWithComputedAmounts.reduce((total, lineItem) => total + lineItem.total_amount, 0)
  }, [lineItemsWithComputedAmounts])

  const paymentTermAmount = useMemo(
    () => calculatePaymentTermAmount(grandTotal, selectedPaymentTerm),
    [grandTotal, selectedPaymentTerm]
  )

  useEffect(() => {
    const debounceTimer = window.setTimeout(() => {
      setCustomerSearchTerm(customerSearchInput.trim())
    }, 300)

    return () => {
      window.clearTimeout(debounceTimer)
    }
  }, [customerSearchInput])

  useEffect(() => {
    let isMounted = true

    const loadDependencies = async () => {
      setIsLoadingDependencies(true)
      try {
        const [recurringPlansResponse, quotationTemplatesResponse, productsResponse, paymentTermsResponse, attributesResponse] = await Promise.all([
          listRecurringPlans('', true),
          listQuotationTemplates(''),
          listProducts(''),
          listPaymentTerms(''),
          listAttributes(''),
        ])

        if (!isMounted) {
          return
        }

        setRecurringPlans(Array.isArray(recurringPlansResponse?.recurring_plans) ? recurringPlansResponse.recurring_plans : [])
        setQuotationTemplates(Array.isArray(quotationTemplatesResponse?.quotations) ? quotationTemplatesResponse.quotations : [])
        setPaymentTerms(Array.isArray(paymentTermsResponse?.payment_terms) ? paymentTermsResponse.payment_terms : [])

        const attributes = Array.isArray(attributesResponse?.attributes) ? attributesResponse.attributes : []
        setAttributeValuesByAttributeID(buildAttributeValuesByAttributeID(attributes))

        const products = Array.isArray(productsResponse?.products) ? productsResponse.products : []
        setProductOptions(mapProductsToOptions(products))

        const initialProductDetails = {}
        products.forEach((product) => {
          const productID = String(product?.product_id ?? '').trim()
          if (!productID) {
            return
          }

          initialProductDetails[productID] = {
            ...product,
            has_full_profile: false,
          }
        })
        setProductDetailsByID(initialProductDetails)
      } catch (error) {
        if (!isMounted) {
          return
        }

        setRecurringPlans([])
        setQuotationTemplates([])
        setPaymentTerms([])
        setAttributeValuesByAttributeID({})
        setProductOptions([])
        setProductDetailsByID({})
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoadingDependencies(false)
        }
      }
    }

    loadDependencies()

    return () => {
      isMounted = false
    }
  }, [])

  useEffect(() => {
    let isMounted = true

    if (!isEditMode || isLoadingDependencies) {
      return () => {
        isMounted = false
      }
    }

    const loadSubscriptionForEdit = async () => {
      setIsLoadingSubscription(true)
      try {
        const response = await getSubscriptionById(subscriptionId)
        if (!isMounted) {
          return
        }

        const subscription = response?.subscription
        const incomingStatus = String(subscription?.status ?? 'Draft')
        const normalizedStatus = STATUS_OPTIONS.includes(incomingStatus) ? incomingStatus : 'Draft'
        const otherInfo = subscription?.other_info

        setCustomerID(String(subscription?.customer_id ?? ''))
        setCustomerSearchInput(String(subscription?.customer_name ?? ''))
        setNextInvoiceDate(String(subscription?.next_invoice_date ?? ''))
        setRecurringPlanID(String(subscription?.recurring_plan_id ?? ''))
        setPaymentTermID(String(subscription?.payment_term_id ?? ''))
        setQuotationID(String(subscription?.quotation_id ?? ''))
        setStatus(normalizedStatus)
        setSalesPerson(String(otherInfo?.sales_person ?? ''))
        setStartDate(String(otherInfo?.start_date ?? ''))
        setPaymentMethod(String(otherInfo?.payment_method ?? ''))
        setIsPaymentMode(otherInfo?.is_payment_mode === true ? 'yes' : (otherInfo?.is_payment_mode === false ? 'no' : ''))

        const subscriptionProducts = Array.isArray(subscription?.products) ? subscription.products : []
        setLineItems(subscriptionProducts.map((product) => {
          const selectedVariants = Array.isArray(product?.selected_variants) ? product.selected_variants : []
          const selectedVariantValueIDs = selectedVariants
            .map((variant) => String(variant?.attribute_value_id ?? '').trim())
            .filter((variantValueID) => variantValueID !== '')

          return {
            product_id: String(product?.product_id ?? ''),
            quantity: Number.isInteger(product?.quantity) && product.quantity > 0 ? product.quantity : 1,
            selected_variant_value_ids: Array.from(new Set(selectedVariantValueIDs)),
          }
        }).filter((lineItem) => lineItem.product_id !== ''))
      } catch (error) {
        if (!isMounted) {
          return
        }

        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoadingSubscription(false)
        }
      }
    }

    loadSubscriptionForEdit()

    return () => {
      isMounted = false
    }
  }, [isEditMode, isLoadingDependencies, subscriptionId])

  useEffect(() => {
    let isMounted = true

    const loadCustomers = async () => {
      setIsLoadingCustomers(true)
      try {
        const response = await listCustomerUsers(customerSearchTerm)
        if (!isMounted) {
          return
        }

        setCustomerOptions(Array.isArray(response?.users) ? response.users : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setCustomerOptions([])
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoadingCustomers(false)
        }
      }
    }

    loadCustomers()

    return () => {
      isMounted = false
    }
  }, [customerSearchTerm])

  useEffect(() => {
    const handleDocumentClick = (event) => {
      if (customerDropdownRef.current && !customerDropdownRef.current.contains(event.target)) {
        setIsCustomerDropdownOpen(false)
      }
    }

    document.addEventListener('mousedown', handleDocumentClick)
    return () => {
      document.removeEventListener('mousedown', handleDocumentClick)
    }
  }, [])

  useEffect(() => {
    if (status === 'Draft') {
      setPaymentTermID('')
    }

    if (status === 'Confirmed') {
      if (!startDate) {
        setStartDate(getTodayDateISO())
      }
      return
    }

    setStartDate('')
    setPaymentMethod('')
    setIsPaymentMode('')
  }, [status, startDate])

  useEffect(() => {
    let isMounted = true

    const missingProductIDs = selectedProductIDs.filter((productID) => {
      const details = productDetailsByID[productID]
      if (!details) {
        return true
      }

      return details.has_full_profile !== true
    })

    if (missingProductIDs.length === 0) {
      return () => {
        isMounted = false
      }
    }

    const loadMissingProductDetails = async () => {
      try {
        const products = await Promise.all(
          missingProductIDs.map((productID) => getProductById(productID).then((response) => response?.product).catch(() => null))
        )

        if (!isMounted) {
          return
        }

        const patch = {}
        products.forEach((product) => {
          const productID = String(product?.product_id ?? '').trim()
          if (!productID) {
            return
          }

          patch[productID] = {
            ...product,
            taxes: Array.isArray(product?.taxes) ? product.taxes : [],
            discounts: Array.isArray(product?.discounts) ? product.discounts : [],
            has_full_profile: true,
          }
        })

        if (Object.keys(patch).length > 0) {
          setProductDetailsByID((previousDetails) => ({
            ...previousDetails,
            ...patch,
          }))
        }
      } catch {
        // Ignore per-product detail failures and keep table operational with fallback values.
      }
    }

    loadMissingProductDetails()

    return () => {
      isMounted = false
    }
  }, [selectedProductIDs, productDetailsByID])

  const handleProductsChange = (productIDs) => {
    const normalizedProductIDs = Array.isArray(productIDs)
      ? productIDs.map((productID) => String(productID ?? '').trim()).filter((productID) => productID !== '')
      : []

    const lineItemByProductID = new Map(
      lineItems.map((lineItem) => [lineItem.product_id, lineItem])
    )

    const nextLineItems = normalizedProductIDs.map((productID) => ({
      product_id: productID,
      quantity: lineItemByProductID.get(productID)?.quantity ?? 1,
      selected_variant_value_ids: Array.isArray(lineItemByProductID.get(productID)?.selected_variant_value_ids)
        ? lineItemByProductID.get(productID).selected_variant_value_ids
        : [],
    }))

    setLineItems(nextLineItems)
  }

  const handleLineItemVariantChange = (productID, variantValueIDs) => {
    const normalizedVariantValueIDs = Array.isArray(variantValueIDs)
      ? Array.from(new Set(
        variantValueIDs
          .map((variantValueID) => String(variantValueID ?? '').trim())
          .filter((variantValueID) => variantValueID !== '')
      ))
      : []

    const variantOptions = getVariantValueOptionsForProduct(productDetailsByID[productID], attributeValuesByAttributeID)
    const variantOptionByID = new Map(variantOptions.map((option) => [option.value, option]))
    const selectedVariantByAttributeID = new Map()

    normalizedVariantValueIDs.forEach((variantValueID) => {
      const option = variantOptionByID.get(variantValueID)
      if (!option) {
        return
      }
      if (selectedVariantByAttributeID.has(option.attribute_id)) {
        return
      }

      selectedVariantByAttributeID.set(option.attribute_id, option)
    })

    const sanitizedVariantValueIDs = Array.from(selectedVariantByAttributeID.values()).map((option) => option.value)

    setLineItems((previousLineItems) => previousLineItems.map((lineItem) => {
      if (lineItem.product_id !== productID) {
        return lineItem
      }

      return {
        ...lineItem,
        selected_variant_value_ids: sanitizedVariantValueIDs,
      }
    }))
  }

  const handleCustomerInputChange = (event) => {
    const nextValue = event.target.value
    setCustomerSearchInput(nextValue)
    setCustomerID('')
    setIsCustomerDropdownOpen(true)
  }

  const handleCustomerSelect = (customer) => {
    const normalizedCustomerID = String(customer?.id ?? '').trim()
    if (!normalizedCustomerID) {
      return
    }

    setCustomerID(normalizedCustomerID)
    setCustomerSearchInput(buildCustomerLabel(customer))
    setIsCustomerDropdownOpen(false)
  }

  const handleQuantityChange = (productID, rawValue) => {
    const parsedValue = Number.parseInt(rawValue, 10)
    const normalizedValue = Number.isInteger(parsedValue) && parsedValue >= 1 ? parsedValue : 1

    setLineItems((previousLineItems) => previousLineItems.map((lineItem) => {
      if (lineItem.product_id !== productID) {
        return lineItem
      }

      return {
        ...lineItem,
        quantity: normalizedValue,
      }
    }))
  }

  const handleQuotationTemplateChange = async (event) => {
    const selectedQuotationID = String(event.target.value ?? '').trim()
    setQuotationID(selectedQuotationID)

    if (!selectedQuotationID) {
      setLineItems([])
      return
    }

    setIsLoadingTemplateProducts(true)
    try {
      const response = await getQuotationTemplateById(selectedQuotationID)
      const quotation = response?.quotation

      setRecurringPlanID(String(quotation?.recurring_plan_id ?? ''))

      const quotationProductIDs = Array.isArray(quotation?.products)
        ? quotation.products
          .map((product) => String(product?.product_id ?? '').trim())
          .filter((productID) => productID !== '')
        : []

      const uniqueProductIDs = Array.from(new Set(quotationProductIDs))
      setLineItems(uniqueProductIDs.map((productID) => ({
        product_id: productID,
        quantity: 1,
        selected_variant_value_ids: [],
      })))
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
      setLineItems([])
    } finally {
      setIsLoadingTemplateProducts(false)
    }
  }

  const handleSubmit = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const normalizedCustomerID = customerID.trim()
    if (!normalizedCustomerID) {
      setToastVariant('error')
      setToastMessage('Customer is required.')
      return
    }

    if (!nextInvoiceDate) {
      setToastVariant('error')
      setToastMessage('Next invoice date is required.')
      return
    }

    const normalizedRecurringPlanID = recurringPlanID.trim()
    if (!normalizedRecurringPlanID) {
      setToastVariant('error')
      setToastMessage('Recurring Plan is required.')
      return
    }

    const normalizedQuotationID = quotationID.trim()
    if (!normalizedQuotationID) {
      setToastVariant('error')
      setToastMessage('Quotation template is required.')
      return
    }

    if (lineItems.length === 0) {
      setToastVariant('error')
      setToastMessage('At least one product is required in the subscription table.')
      return
    }

    const payloadProducts = []
    for (const lineItem of lineItems) {
      const normalizedProductID = String(lineItem?.product_id ?? '').trim()
      if (!normalizedProductID) {
        setToastVariant('error')
        setToastMessage('Product ID cannot be empty.')
        return
      }

      const quantity = Number.parseInt(String(lineItem?.quantity ?? ''), 10)
      if (!Number.isInteger(quantity) || quantity < 1) {
        setToastVariant('error')
        setToastMessage('Quantity must be a whole number with minimum value 1.')
        return
      }

      const selectedVariantValueIDs = Array.isArray(lineItem?.selected_variant_value_ids)
        ? Array.from(new Set(
          lineItem.selected_variant_value_ids
            .map((variantValueID) => String(variantValueID ?? '').trim())
            .filter((variantValueID) => variantValueID !== '')
        ))
        : []

      payloadProducts.push({
        product_id: normalizedProductID,
        quantity,
        selected_variant_value_ids: selectedVariantValueIDs,
      })
    }

    setIsSubmitting(true)
    try {
      const normalizedPaymentTermID = paymentTermID.trim()
      const normalizedSalesPerson = salesPerson.trim()
      const normalizedStartDate = startDate.trim()
      const normalizedPaymentMethod = paymentMethod.trim()

      const payload = {
        customer_id: normalizedCustomerID,
        next_invoice_date: nextInvoiceDate,
        recurring_plan_id: normalizedRecurringPlanID,
        payment_term_id: isPaymentTermEnabled ? normalizedPaymentTermID : '',
        quotation_id: normalizedQuotationID,
        status,
        products: payloadProducts,
        other_info: {
          sales_person: normalizedSalesPerson,
          start_date: isStatusConfirmed ? normalizedStartDate : '',
          payment_method: isStatusConfirmed ? normalizedPaymentMethod : '',
          is_payment_mode: isStatusConfirmed
            ? (isPaymentMode === 'yes' ? true : (isPaymentMode === 'no' ? false : null))
            : null,
        },
      }

      const response = isEditMode
        ? await updateSubscription(subscriptionId, payload)
        : await createSubscription(payload)

      setToastVariant('success')
      const responseSubscriptionNumber = String(response?.subscription?.subscription_number ?? '').trim()
      setToastMessage(
        responseSubscriptionNumber
          ? `Subscription ${responseSubscriptionNumber} ${isEditMode ? 'updated' : 'created'} successfully.`
          : (response?.message ?? `Subscription ${isEditMode ? 'updated' : 'created'} successfully.`)
      )

      if (!isEditMode) {
        setCustomerSearchInput('')
        setCustomerSearchTerm('')
        setCustomerID('')
        setIsCustomerDropdownOpen(false)
        setActiveInfoTab(INFO_TAB_ORDER)
        setNextInvoiceDate('')
        setRecurringPlanID('')
        setPaymentTermID('')
        setQuotationID('')
        setStatus('Draft')
        setSalesPerson('')
        setStartDate('')
        setPaymentMethod('')
        setIsPaymentMode('')
        setLineItems([])
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">{isEditMode ? 'Edit Subscription' : 'New Subscription'}</h1>
        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => setStatus('Quotation Sent')}
            className={`inline-flex h-10 items-center rounded-lg px-4 text-sm font-semibold transition-colors duration-200 ${
              status === 'Quotation Sent'
                ? 'bg-[var(--orange)] text-white'
                : 'border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] hover:bg-[rgba(0,0,128,0.04)]'
            }`}
          >
            Send
          </button>
          <button
            type="button"
            onClick={() => setStatus('Confirmed')}
            className={`inline-flex h-10 items-center rounded-lg px-4 text-sm font-semibold transition-colors duration-200 ${
              status === 'Confirmed'
                ? 'bg-emerald-600 text-white hover:bg-emerald-700'
                : 'border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] hover:bg-[rgba(0,0,128,0.04)]'
            }`}
          >
            Confirm
          </button>
          <Link
            to="/admin/subscriptions"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
          >
            Back to Subscriptions
          </Link>
        </div>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        {isEditMode
          ? 'Update subscription details, line items, and status.'
          : 'Create a subscription with customer details, recurring plan, next invoice date, and status.'}
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      {isLoadingDependencies || (isEditMode && isLoadingSubscription) ? (
        <div className="mt-6 rounded-xl border border-[color:rgba(0,0,128,0.14)] px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
          {isEditMode
            ? 'Loading subscription details...'
            : 'Loading recurring plans, quotation templates, and products...'}
        </div>
      ) : (
        <form className="mt-6 space-y-6" onSubmit={handleSubmit} noValidate>
          <div className="space-y-2" ref={customerDropdownRef}>
            <label htmlFor="customer-search" className="block text-sm font-semibold text-[var(--navy)]">
              Customer Name
            </label>
            <div className="relative">
              <input
                id="customer-search"
                name="customer-search"
                type="search"
                value={customerSearchInput}
                onChange={handleCustomerInputChange}
                onFocus={() => setIsCustomerDropdownOpen(true)}
                placeholder="Search customer by name, email, or phone"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
                autoComplete="off"
              />

              {isCustomerDropdownOpen ? (
                <div className="absolute z-20 mt-1 w-full overflow-hidden rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] shadow-[0_12px_30px_rgba(0,0,128,0.12)]">
                  {isLoadingCustomers ? (
                    <p className="px-4 py-3 text-sm text-[color:rgba(0,0,128,0.7)]">Loading customers...</p>
                  ) : customerOptions.length === 0 ? (
                    <p className="px-4 py-3 text-sm text-[color:rgba(0,0,128,0.7)]">No matching customers found.</p>
                  ) : (
                    <ul className="max-h-64 overflow-y-auto py-1">
                      {customerOptions.map((customer) => {
                        const customerLabel = buildCustomerLabel(customer)
                        return (
                          <li key={customer.id}>
                            <button
                              type="button"
                              onClick={() => handleCustomerSelect(customer)}
                              className="w-full px-4 py-2.5 text-left text-sm text-[var(--navy)] transition-colors duration-150 hover:bg-[rgba(0,0,128,0.06)]"
                            >
                              {customerLabel}
                            </button>
                          </li>
                        )
                      })}
                    </ul>
                  )}
                </div>
              ) : null}
            </div>

            <div className="mt-2 inline-flex rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[rgba(0,0,128,0.03)] p-1">
              <button
                type="button"
                onClick={() => setActiveInfoTab(INFO_TAB_ORDER)}
                className={`rounded-md px-4 py-1.5 text-sm font-semibold transition-colors duration-150 ${
                  activeInfoTab === INFO_TAB_ORDER
                    ? 'bg-[var(--orange)] text-white'
                    : 'text-[var(--navy)] hover:bg-[rgba(0,0,128,0.08)]'
                }`}
              >
                Order Info
              </button>
              <button
                type="button"
                onClick={() => setActiveInfoTab(INFO_TAB_OTHER)}
                className={`rounded-md px-4 py-1.5 text-sm font-semibold transition-colors duration-150 ${
                  activeInfoTab === INFO_TAB_OTHER
                    ? 'bg-[var(--orange)] text-white'
                    : 'text-[var(--navy)] hover:bg-[rgba(0,0,128,0.08)]'
                }`}
              >
                Other Info
              </button>
            </div>
          </div>

          {activeInfoTab === INFO_TAB_ORDER ? (
            <>
              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <label htmlFor="next-invoice-date" className="block text-sm font-semibold text-[var(--navy)]">
                    Next Invoice Date
                  </label>
                  <input
                    id="next-invoice-date"
                    name="next-invoice-date"
                    type="date"
                    value={nextInvoiceDate}
                    onChange={(event) => setNextInvoiceDate(event.target.value)}
                    className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
                  />
                </div>

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
                      <option key={statusOption} value={statusOption}>{statusOption}</option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <label htmlFor="quotation-template" className="block text-sm font-semibold text-[var(--navy)]">
                    Quotation Template
                  </label>
                  <select
                    id="quotation-template"
                    name="quotation-template"
                    value={quotationID}
                    onChange={handleQuotationTemplateChange}
                    className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
                  >
                    <option value="">Select quotation template</option>
                    {quotationTemplates.map((quotationTemplate) => (
                      <option key={quotationTemplate.quotation_id} value={quotationTemplate.quotation_id}>
                        {buildTemplateLabel(quotationTemplate)}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-2">
                  <label htmlFor="recurring-plan" className="block text-sm font-semibold text-[var(--navy)]">
                    Recurring Plan
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
              </div>

              <div className="space-y-2">
                <label htmlFor="recurring" className="block text-sm font-semibold text-[var(--navy)]">
                  Recurring
                </label>
                <input
                  id="recurring"
                  name="recurring"
                  type="text"
                  value={selectedRecurringPlan?.billing_period ?? ''}
                  readOnly
                  placeholder="Recurring period will be shown after selecting a plan"
                  className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[rgba(0,0,128,0.04)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)]"
                />
              </div>

              <div className="space-y-2">
                <label className="block text-sm font-semibold text-[var(--navy)]">Products</label>
                <MultiSelectDropdown
                  buttonLabel="Products"
                  options={productOptions}
                  selectedValues={selectedProductIDs}
                  onChange={handleProductsChange}
                  placeholder="Select products"
                  emptyMessage="No products available."
                />
                <p className="text-xs text-[color:rgba(0,0,128,0.62)]">
                  Selected products: {selectedProductIDs.length}
                </p>
              </div>

              <div className="overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
                <div className="grid grid-cols-[1.7fr_0.7fr_1fr_1fr_1fr_1fr] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
                  <span>Product</span>
                  <span>Quantity</span>
                  <span>Unit Price</span>
                  <span>Discount</span>
                  <span>Taxes</span>
                  <span>Total Amount</span>
                </div>

                {isLoadingTemplateProducts ? (
                  <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading products from quotation template...</div>
                ) : lineItemsWithComputedAmounts.length === 0 ? (
                  <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
                    No products selected. Select a quotation template or add products manually.
                  </div>
                ) : (
                  <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
                    {lineItemsWithComputedAmounts.map((lineItem) => (
                      <div
                        key={lineItem.product_id}
                        className="grid grid-cols-[1.7fr_0.7fr_1fr_1fr_1fr_1fr] gap-4 px-4 py-3 text-sm text-[var(--navy)]"
                      >
                        <div>
                          <p className="font-semibold">{lineItem.product_name}</p>
                          <div className="mt-2">
                            <MultiSelectDropdown
                              buttonLabel="Variants"
                              options={Array.isArray(lineItem.variant_options) ? lineItem.variant_options : []}
                              selectedValues={Array.isArray(lineItem.selected_variant_value_ids) ? lineItem.selected_variant_value_ids : []}
                              onChange={(variantValueIDs) => handleLineItemVariantChange(lineItem.product_id, variantValueIDs)}
                              placeholder="Select variants (optional)"
                              emptyMessage="No variants configured for this product."
                            />
                          </div>
                          <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
                            Variant Extra: {formatCurrency(lineItem.variant_extra_amount)}
                          </p>
                        </div>
                        <div>
                          <input
                            type="number"
                            min="1"
                            step="1"
                            value={lineItem.quantity}
                            onChange={(event) => handleQuantityChange(lineItem.product_id, event.target.value)}
                            className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-2.5 py-1.5 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
                          />
                        </div>
                        <div>{formatCurrency(lineItem.unit_price)}</div>
                        <div>{formatCurrency(lineItem.discount_amount)}</div>
                        <div>{formatCurrency(lineItem.tax_amount)}</div>
                        <div className="font-semibold">{formatCurrency(lineItem.total_amount)}</div>
                      </div>
                    ))}

                    <div className="space-y-3 border-t border-[color:rgba(0,0,128,0.08)] px-4 py-3">
                      <div className="flex justify-end text-sm font-semibold text-[var(--navy)]">
                        Grand Total: {formatCurrency(grandTotal)}
                      </div>

                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <label htmlFor="payment-term" className="block text-sm font-semibold text-[var(--navy)]">
                            Payment Term (Final Invoice Amount)
                          </label>
                          <select
                            id="payment-term"
                            name="payment-term"
                            value={paymentTermID}
                            onChange={(event) => setPaymentTermID(event.target.value)}
                            disabled={!isPaymentTermEnabled}
                            className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)] disabled:cursor-not-allowed disabled:bg-[rgba(0,0,128,0.04)]"
                          >
                            <option value="">{isPaymentTermEnabled ? 'Select payment term' : 'Available after sending quotation'}</option>
                            {paymentTerms.map((paymentTerm) => (
                              <option key={paymentTerm.payment_term_id} value={paymentTerm.payment_term_id}>
                                {paymentTerm.payment_term_name}
                              </option>
                            ))}
                          </select>
                        </div>

                        <div className="rounded-lg border border-[color:rgba(0,0,128,0.16)] bg-[rgba(0,0,128,0.03)] px-4 py-3">
                          <p className="text-xs font-semibold uppercase tracking-[0.08em] text-[color:rgba(0,0,128,0.72)]">Amount Based On Payment Term</p>
                          <p className="mt-1 text-lg font-bold text-[var(--navy)]">
                            {isPaymentTermEnabled ? formatCurrency(paymentTermAmount) : '-'}
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="space-y-5 rounded-xl border border-[color:rgba(0,0,128,0.12)] bg-[rgba(0,0,128,0.02)] p-4 sm:p-5">
              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <label htmlFor="sales-person" className="block text-sm font-semibold text-[var(--navy)]">
                    Sales Person
                  </label>
                  <input
                    id="sales-person"
                    name="sales-person"
                    type="text"
                    value={salesPerson}
                    onChange={(event) => setSalesPerson(event.target.value)}
                    placeholder="Enter sales person"
                    className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
                  />
                </div>

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
                    disabled={!isStatusConfirmed}
                    className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)] disabled:cursor-not-allowed disabled:bg-[rgba(0,0,128,0.04)]"
                  />
                </div>
              </div>

              <div className="grid gap-5 sm:grid-cols-2">
                <div className="space-y-2">
                  <label htmlFor="payment-method" className="block text-sm font-semibold text-[var(--navy)]">
                    Payment Method
                  </label>
                  <input
                    id="payment-method"
                    name="payment-method"
                    type="text"
                    value={paymentMethod}
                    onChange={(event) => setPaymentMethod(event.target.value)}
                    disabled={!isStatusConfirmed}
                    placeholder={isStatusConfirmed ? 'Enter payment method' : 'Available after confirmation'}
                    className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)] disabled:cursor-not-allowed disabled:bg-[rgba(0,0,128,0.04)]"
                  />
                </div>

                <div className="space-y-2">
                  <label htmlFor="is-payment-mode" className="block text-sm font-semibold text-[var(--navy)]">
                    Is Payment Mode
                  </label>
                  <select
                    id="is-payment-mode"
                    name="is-payment-mode"
                    value={isPaymentMode}
                    onChange={(event) => setIsPaymentMode(event.target.value)}
                    disabled={!isStatusConfirmed}
                    className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)] disabled:cursor-not-allowed disabled:bg-[rgba(0,0,128,0.04)]"
                  >
                    <option value="">{isStatusConfirmed ? 'Select option' : 'Available after confirmation'}</option>
                    <option value="yes">Yes</option>
                    <option value="no">No</option>
                  </select>
                </div>
              </div>

              {!isStatusConfirmed ? (
                <p className="text-xs text-[color:rgba(0,0,128,0.66)]">
                  Start date and payment fields remain empty until the quotation is confirmed.
                </p>
              ) : null}
            </div>
          )}

          <div className="flex flex-wrap items-center justify-end gap-3">
            <Link
              to="/admin/subscriptions"
              className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
            >
              Cancel
            </Link>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
            >
              {isSubmitting ? (isEditMode ? 'Updating...' : 'Saving...') : (isEditMode ? 'Update Subscription' : 'Save Subscription')}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default SubscriptionNewPage
