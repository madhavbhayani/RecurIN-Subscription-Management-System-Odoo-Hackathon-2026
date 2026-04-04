import { useState } from 'react'
import { Link } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { createTax } from '../../../../services/taxApi'

const TAX_COMPUTATION_UNITS = [
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

function TaxesNewPage() {
  const [taxName, setTaxName] = useState('')
  const [taxComputationUnit, setTaxComputationUnit] = useState('Fixed Price')
  const [taxComputationValue, setTaxComputationValue] = useState('')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectedTaxUnit = TAX_COMPUTATION_UNITS.find((unit) => unit.value === taxComputationUnit)
  const currentTaxUnitSymbol = selectedTaxUnit?.symbol ?? ''

  const handleSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    const normalizedTaxName = taxName.trim()
    if (!normalizedTaxName) {
      setToastVariant('error')
      setToastMessage('Tax name is required.')
      return
    }

    const normalizedValueText = String(taxComputationValue).trim()
    const normalizedValue = normalizedValueText === '' ? 0 : Number(normalizedValueText)
    if (!Number.isFinite(normalizedValue) || normalizedValue < 0) {
      setToastVariant('error')
      setToastMessage('Tax computation value must be a non-negative number.')
      return
    }
    if (taxComputationUnit === 'Percentage' && normalizedValue > 100) {
      setToastVariant('error')
      setToastMessage('Percentage tax value cannot be greater than 100.')
      return
    }

    setIsSubmitting(true)
    try {
      const response = await createTax({
        tax_name: normalizedTaxName,
        tax_computation_unit: taxComputationUnit,
        tax_computation_value: Number(normalizedValue.toFixed(2)),
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Tax created successfully.')

      setTaxName('')
      setTaxComputationUnit('Fixed Price')
      setTaxComputationValue('')
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">
          New Tax
        </h1>
        <Link
          to="/admin/configurations/taxes"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Taxes
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Create a tax configuration with computation unit and value.
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      <form className="mt-6 space-y-6" onSubmit={handleSubmit} noValidate>
        <div className="space-y-2">
          <label htmlFor="tax-name" className="block text-sm font-semibold text-[var(--navy)]">
            Tax Name
          </label>
          <input
            id="tax-name"
            name="tax-name"
            type="text"
            value={taxName}
            onChange={(event) => setTaxName(event.target.value)}
            placeholder="Enter tax name"
            className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
          />
        </div>

        <div className="grid gap-5 sm:grid-cols-2">
          <div className="space-y-2">
            <label htmlFor="tax-computation-unit" className="block text-sm font-semibold text-[var(--navy)]">
              Tax Computation Unit
            </label>
            <select
              id="tax-computation-unit"
              name="tax-computation-unit"
              value={taxComputationUnit}
              onChange={(event) => setTaxComputationUnit(event.target.value)}
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
            >
              {TAX_COMPUTATION_UNITS.map((unit) => (
                <option key={unit.value} value={unit.value}>{unit.label}</option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <label htmlFor="tax-computation-value" className="block text-sm font-semibold text-[var(--navy)]">
              Tax Computation Value
            </label>
            <div className="relative">
              <input
                id="tax-computation-value"
                name="tax-computation-value"
                type="number"
                inputMode="decimal"
                min="0"
                max={taxComputationUnit === 'Percentage' ? 100 : undefined}
                step="0.01"
                value={taxComputationValue}
                onChange={(event) => setTaxComputationValue(event.target.value)}
                placeholder="0.00"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 pr-10 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
              <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-[color:rgba(0,0,128,0.7)]">
                {currentTaxUnitSymbol}
              </span>
            </div>
          </div>
        </div>

        <div className="flex flex-wrap items-center justify-end gap-3">
          <Link
            to="/admin/configurations/taxes"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
          >
            {isSubmitting ? 'Saving...' : 'Save Tax'}
          </button>
        </div>
      </form>
    </div>
  )
}

export default TaxesNewPage
