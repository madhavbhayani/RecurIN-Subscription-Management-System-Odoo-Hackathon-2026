import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { createRecurringPlan } from '../../../../services/recurringPlanApi'

const BILLING_PERIODS = ['Daily', 'Weekly', 'Monthly', 'Yearly']

const BILLING_PERIOD_LIMITS = {
  Daily: { min: 1, max: 365, unit: 'day(s)' },
  Weekly: { min: 1, max: 52, unit: 'week(s)' },
  Monthly: { min: 1, max: 12, unit: 'month(s)' },
  Yearly: { min: 1, max: 10, unit: 'year(s)' },
}

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

function RecurringPlanNewPage() {
  const [recurringName, setRecurringName] = useState('')
  const [billingPeriod, setBillingPeriod] = useState('Monthly')
  const [isClosable, setIsClosable] = useState(false)
  const [automaticCloseCycles, setAutomaticCloseCycles] = useState('')
  const [isPausable, setIsPausable] = useState(false)
  const [isRenewable, setIsRenewable] = useState(true)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('info')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const periodLimit = BILLING_PERIOD_LIMITS[billingPeriod] ?? BILLING_PERIOD_LIMITS.Monthly

  const automaticCloseLabel = useMemo(() => {
    return `Automatic Close (${periodLimit.min}-${periodLimit.max} ${periodLimit.unit})`
  }, [periodLimit.max, periodLimit.min, periodLimit.unit])

  useEffect(() => {
    if (!isClosable) {
      setAutomaticCloseCycles('')
      return
    }

    if (automaticCloseCycles === '') {
      return
    }

    const numericValue = Number(automaticCloseCycles)
    if (!Number.isFinite(numericValue)) {
      setAutomaticCloseCycles('')
      return
    }

    if (numericValue < periodLimit.min) {
      setAutomaticCloseCycles(String(periodLimit.min))
      return
    }

    if (numericValue > periodLimit.max) {
      setAutomaticCloseCycles(String(periodLimit.max))
    }
  }, [automaticCloseCycles, isClosable, periodLimit.max, periodLimit.min])

  const handleSubmit = async (event) => {
    event.preventDefault()
    setToastMessage('')

    const normalizedRecurringName = recurringName.trim()
    if (!normalizedRecurringName) {
      setToastVariant('error')
      setToastMessage('Recurring Name is required.')
      return
    }

    let normalizedAutomaticCloseCycles = null
    if (isClosable) {
      const numericCycles = Number(automaticCloseCycles)
      if (!Number.isInteger(numericCycles)) {
        setToastVariant('error')
        setToastMessage('Automatic Close cycles must be a whole number.')
        return
      }
      if (numericCycles < periodLimit.min || numericCycles > periodLimit.max) {
        setToastVariant('error')
        setToastMessage(`Automatic Close cycles must be between ${periodLimit.min} and ${periodLimit.max} for ${billingPeriod}.`)
        return
      }

      normalizedAutomaticCloseCycles = numericCycles
    }

    setIsSubmitting(true)
    try {
      const response = await createRecurringPlan({
        recurring_name: normalizedRecurringName,
        billing_period: billingPeriod,
        is_closable: isClosable,
        automatic_close_cycles: normalizedAutomaticCloseCycles,
        is_pausable: isPausable,
        is_renewable: isRenewable,
        is_active: true,
      })

      setToastVariant('success')
      setToastMessage(response?.message ?? 'Recurring plan created successfully.')

      setRecurringName('')
      setBillingPeriod('Monthly')
      setIsClosable(false)
      setAutomaticCloseCycles('')
      setIsPausable(false)
      setIsRenewable(true)
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">New Recurring Plan</h1>
        <Link
          to="/admin/configurations/recurring-plan"
          className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
        >
          Back to Recurring Plans
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Create recurring cycle behavior.
      </p>

      <ToastMessage
        message={toastMessage}
        variant={toastVariant}
        onClose={() => setToastMessage('')}
      />

      <form className="mt-6 space-y-7" onSubmit={handleSubmit} noValidate>
        <div className="grid gap-5 sm:grid-cols-2">
          <div className="space-y-2">
            <label htmlFor="recurring-name" className="block text-sm font-semibold text-[var(--navy)]">
              Recurring Name
            </label>
            <input
              id="recurring-name"
              name="recurring-name"
              type="text"
              value={recurringName}
              onChange={(event) => setRecurringName(event.target.value)}
              placeholder="Enter recurring plan name"
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="billing-period" className="block text-sm font-semibold text-[var(--navy)]">
              Billing Period
            </label>
            <select
              id="billing-period"
              name="billing-period"
              value={billingPeriod}
              onChange={(event) => setBillingPeriod(event.target.value)}
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] bg-[var(--white)] px-4 py-3 text-sm text-[var(--navy)] outline-none focus:border-[var(--orange)]"
            >
              {BILLING_PERIODS.map((period) => (
                <option key={period} value={period}>{period}</option>
              ))}
            </select>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Toggle
            checked={isClosable}
            onChange={setIsClosable}
            label="Is Closable"
            description="Enable automatic closure based on cycle count."
          />

          <Toggle
            checked={isPausable}
            onChange={setIsPausable}
            label="Is Pausable"
            description="Allow the subscription to be paused."
          />

          <Toggle
            checked={isRenewable}
            onChange={setIsRenewable}
            label="Is Renewable"
            description="Allow renewal at the end of billing period."
          />

        </div>

        {isClosable ? (
          <div className="space-y-2">
            <label htmlFor="automatic-close-cycles" className="block text-sm font-semibold text-[var(--navy)]">
              {automaticCloseLabel}
            </label>
            <input
              id="automatic-close-cycles"
              name="automatic-close-cycles"
              type="number"
              inputMode="numeric"
              min={periodLimit.min}
              max={periodLimit.max}
              step="1"
              value={automaticCloseCycles}
              onChange={(event) => setAutomaticCloseCycles(event.target.value)}
              placeholder={`${periodLimit.min}-${periodLimit.max}`}
              className="w-full rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 py-3 text-base text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
            />
            <p className="text-xs text-[color:rgba(0,0,128,0.66)]">
              Example: choose 6 on Monthly billing to automatically close after 6 months.
            </p>
          </div>
        ) : null}

        <div className="rounded-xl border border-[color:rgba(0,0,128,0.14)] p-4">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold text-[var(--navy)]">Subscription Products</h2>
            <span className="rounded-full border border-[color:rgba(0,0,128,0.2)] bg-[rgba(0,0,128,0.04)] px-3 py-1 text-xs font-semibold text-[var(--navy)]">
              Product Count: 0
            </span>
          </div>

          <div className="mt-4 overflow-hidden rounded-lg border border-red-300">
            <div className="grid grid-cols-[1.4fr_1fr_1fr_1fr] border-b border-dashed border-red-300 bg-red-50/40 text-sm font-semibold text-red-600">
              <div className="border-r border-dashed border-red-300 px-4 py-3">Product</div>
              <div className="border-r border-dashed border-red-300 px-4 py-3">Product Type</div>
              <div className="border-r border-dashed border-red-300 px-4 py-3">Price</div>
              <div className="px-4 py-3">Min Qty.</div>
            </div>

            <div className="grid grid-cols-[1.4fr_1fr_1fr_1fr] text-sm text-red-500">
              <div className="border-r border-dashed border-red-300 px-4 py-4">No products</div>
              <div className="border-r border-dashed border-red-300 px-4 py-4">-</div>
              <div className="border-r border-dashed border-red-300 px-4 py-4">-</div>
              <div className="px-4 py-4">-</div>
            </div>
          </div>

          <p className="mt-2 text-xs text-[color:rgba(0,0,128,0.62)]">
            Products will appear after mappings are available for this recurring plan.
          </p>
        </div>

        <div className="flex flex-wrap items-center justify-end gap-3">
          <Link
            to="/admin/configurations/recurring-plan"
            className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)]"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={isSubmitting}
            className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] disabled:cursor-not-allowed disabled:opacity-70"
          >
            {isSubmitting ? 'Saving...' : 'Save Recurring Plan'}
          </button>
        </div>
      </form>
    </div>
  )
}

export default RecurringPlanNewPage
