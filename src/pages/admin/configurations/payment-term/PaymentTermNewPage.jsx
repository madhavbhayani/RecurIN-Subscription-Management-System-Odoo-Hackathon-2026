import { useState } from 'react'
import { Link } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { createPaymentTerm } from '../../../../services/paymentTermApi'

const DUE_UNITS = [
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

function PaymentTermNewPage() {
  const [paymentTermName, setPaymentTermName] = useState('')
  const [dueUnit, setDueUnit] = useState('Percentage')
  const [dueValue, setDueValue] = useState('')
  const [intervalDays, setIntervalDays] = useState('')
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectedDueUnit = DUE_UNITS.find((unit) => unit.value === dueUnit)
  const dueUnitSymbol = selectedDueUnit?.symbol ?? ''

  const handleSubmit = async (event) => {
    event.preventDefault()

    setToastMessage('')

    const normalizedPaymentTermName = paymentTermName.trim()
    if (!normalizedPaymentTermName) {
      setToastVariant('error')
      setToastMessage('Payment term name is required.')
      return
    }

    const parsedDueValue = Number(dueValue)
    if (!Number.isFinite(parsedDueValue) || parsedDueValue <= 0) {
      setToastVariant('error')
      setToastMessage('Due value must be a number greater than zero.')
      return
    }
    if (dueUnit === 'Percentage' && parsedDueValue > 100) {
      setToastVariant('error')
      setToastMessage('Percentage due value cannot be greater than 100.')
      return
    }

    const parsedIntervalDays = Number(intervalDays)
    if (!Number.isInteger(parsedIntervalDays) || parsedIntervalDays <= 0) {
      setToastVariant('error')
      setToastMessage('Interval (in days) must be a whole number greater than zero.')
      return
    }

    setIsSubmitting(true)
    try {
      const response = await createPaymentTerm({
        payment_term_name: normalizedPaymentTermName,
        due_unit: dueUnit,
        due_value: Number(parsedDueValue.toFixed(2)),
        interval_days: parsedIntervalDays,
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Payment term created successfully.')

      setPaymentTermName('')
      setDueUnit('Percentage')
      setDueValue('')
      setIntervalDays('')
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">New Payment Term</h1>
        <Link
          to="/admin/configurations/payment-term"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Payment Terms
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Define due unit, amount, and interval after quotation acceptance.
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      <form className="mt-6 space-y-6" onSubmit={handleSubmit} noValidate>
        <div className="space-y-2">
          <label htmlFor="payment-term-name" className="block text-sm font-semibold text-[var(--navy)]">
            Payment Term Name
          </label>
          <input
            id="payment-term-name"
            name="payment-term-name"
            type="text"
            value={paymentTermName}
            onChange={(event) => setPaymentTermName(event.target.value)}
            placeholder="Enter payment term name"
            className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
          />
        </div>

        <div className="grid gap-5 sm:grid-cols-2">
          <div className="space-y-2">
            <label htmlFor="due-unit" className="block text-sm font-semibold text-[var(--navy)]">
              Due Unit
            </label>
            <select
              id="due-unit"
              name="due-unit"
              value={dueUnit}
              onChange={(event) => setDueUnit(event.target.value)}
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
            >
              {DUE_UNITS.map((unit) => (
                <option key={unit.value} value={unit.value}>{unit.label}</option>
              ))}
            </select>
          </div>

          <div className="space-y-2">
            <label htmlFor="due-value" className="block text-sm font-semibold text-[var(--navy)]">
              Due Value
            </label>
            <div className="relative">
              <input
                id="due-value"
                name="due-value"
                type="number"
                inputMode="decimal"
                min="0"
                max={dueUnit === 'Percentage' ? 100 : undefined}
                step="0.01"
                value={dueValue}
                onChange={(event) => setDueValue(event.target.value)}
                placeholder="0.00"
                className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 pr-10 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
              />
              <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-[color:rgba(0,0,128,0.7)]">
                {dueUnitSymbol}
              </span>
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <label htmlFor="interval-days" className="block text-sm font-semibold text-[var(--navy)]">
            Interval (in days)
          </label>
          <input
            id="interval-days"
            name="interval-days"
            type="number"
            inputMode="numeric"
            min="1"
            step="1"
            value={intervalDays}
            onChange={(event) => setIntervalDays(event.target.value)}
            placeholder="Enter interval in days"
            className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
          />
        </div>

        <div className="flex flex-wrap items-center justify-end gap-3">
          <Link
            to="/admin/configurations/payment-term"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
          >
            {isSubmitting ? 'Saving...' : 'Save Payment Term'}
          </button>
        </div>
      </form>
    </div>
  )
}

export default PaymentTermNewPage
