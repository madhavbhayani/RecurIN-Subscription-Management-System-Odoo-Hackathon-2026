import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { deletePaymentTerm, listPaymentTerms } from '../../../../services/paymentTermApi'

function formatDueValue(dueUnit, dueValue) {
  const numericValue = Number(dueValue)
  const formattedValue = Number.isFinite(numericValue) ? numericValue.toFixed(2) : '0.00'

  if (dueUnit === 'Percentage') {
    return `${formattedValue}%`
  }

  if (dueUnit === 'Fixed Price') {
    return `$${formattedValue}`
  }

  return formattedValue
}

function PaymentTermPage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [paymentTerms, setPaymentTerms] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [paymentTermPendingDelete, setPaymentTermPendingDelete] = useState(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    const debounceTimer = window.setTimeout(() => {
      setSearchTerm(searchInput.trim())
    }, 300)

    return () => {
      window.clearTimeout(debounceTimer)
    }
  }, [searchInput])

  useEffect(() => {
    let isMounted = true

    const fetchPaymentTerms = async () => {
      setIsLoading(true)
      try {
        const response = await listPaymentTerms(searchTerm)
        if (!isMounted) {
          return
        }

        setPaymentTerms(Array.isArray(response?.payment_terms) ? response.payment_terms : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setPaymentTerms([])
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchPaymentTerms()

    return () => {
      isMounted = false
    }
  }, [searchTerm])

  const handleOpenDeleteDialog = (paymentTerm) => {
    setPaymentTermPendingDelete(paymentTerm)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }

    setPaymentTermPendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const paymentTermID = String(paymentTermPendingDelete?.payment_term_id ?? '').trim()
    if (!paymentTermID) {
      setPaymentTermPendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deletePaymentTerm(paymentTermID)
      setPaymentTerms((previousPaymentTerms) => previousPaymentTerms.filter((paymentTerm) => paymentTerm.payment_term_id !== paymentTermID))
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Payment term deleted successfully.')
      setPaymentTermPendingDelete(null)
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-8">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <div className="flex items-center justify-between gap-4">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Payment Term</h1>
        <Link
          to="/admin/configurations/payment-term/new"
          className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Define due amount (percentage or fixed price) and interval in days after quotation acceptance.
      </p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by payment term name or unit"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        <div className="grid grid-cols-[1.2fr_1fr_1fr_1fr_120px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
          <span>Payment Term Name</span>
          <span>Due Unit</span>
          <span>Due Value</span>
          <span>Interval (days)</span>
          <span className="text-right">Action</span>
        </div>

        {isLoading ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading payment terms...</div>
        ) : paymentTerms.length === 0 ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
            {searchTerm
              ? 'No payment terms match your search.'
              : 'No payment terms found yet. Click New to create one.'}
          </div>
        ) : (
          <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
            {paymentTerms.map((paymentTerm) => (
              <div key={paymentTerm.payment_term_id} className="grid grid-cols-[1.2fr_1fr_1fr_1fr_120px] gap-4 px-4 py-4 text-sm text-[var(--navy)]">
                <div className="font-semibold">{paymentTerm.payment_term_name}</div>
                <div>{paymentTerm.due_unit}</div>
                <div>{formatDueValue(paymentTerm.due_unit, paymentTerm.due_value)}</div>
                <div>{Number(paymentTerm.interval_days ?? 0)}</div>
                <div className="flex items-center justify-end gap-2">
                  <Link
                    to={`/admin/configurations/payment-term/${paymentTerm.payment_term_id}`}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                    title="Edit"
                    aria-label="Edit payment term"
                  >
                    <FiEdit2 className="h-4 w-4" />
                  </Link>

                  <button
                    type="button"
                    onClick={() => handleOpenDeleteDialog(paymentTerm)}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
                    title="Delete"
                    aria-label="Delete payment term"
                  >
                    <FiTrash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {paymentTermPendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Payment Term</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{paymentTermPendingDelete.payment_term_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              This action will permanently remove the payment term.
            </p>

            <div className="mt-6 flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={handleCloseDeleteDialog}
                disabled={isDeleting}
                className="inline-flex h-10 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-4 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)] disabled:cursor-not-allowed disabled:opacity-60"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmDelete}
                disabled={isDeleting}
                className="inline-flex h-10 items-center rounded-lg bg-red-600 px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-70"
              >
                {isDeleting ? 'Deleting...' : 'Yes, Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default PaymentTermPage
