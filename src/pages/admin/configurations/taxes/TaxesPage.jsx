import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { deleteTax, listTaxes } from '../../../../services/taxApi'

const TAX_UNIT_META = {
  'Fixed Price': {
    label: 'Fixed Price (\u20b9)',
    symbol: '\u20b9',
  },
  Percentage: {
    label: 'Percentage (%)',
    symbol: '%',
  },
}

function formatTaxValue(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '0.00'
  }

  return numericValue.toFixed(2)
}

function getTaxUnitMeta(taxComputationUnit) {
  return TAX_UNIT_META[taxComputationUnit] ?? {
    label: String(taxComputationUnit ?? '-'),
    symbol: '',
  }
}

function formatTaxValueWithUnit(value, taxComputationUnit) {
  const formattedValue = formatTaxValue(value)
  const { symbol } = getTaxUnitMeta(taxComputationUnit)

  if (symbol === '\u20b9') {
    return `${symbol} ${formattedValue}`
  }
  if (symbol === '%') {
    return `${formattedValue}${symbol}`
  }

  return formattedValue
}

function TaxesPage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [taxes, setTaxes] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [taxPendingDelete, setTaxPendingDelete] = useState(null)
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

    const fetchTaxes = async () => {
      setIsLoading(true)
      try {
        const response = await listTaxes(searchTerm)
        if (!isMounted) {
          return
        }

        setTaxes(Array.isArray(response?.taxes) ? response.taxes : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setTaxes([])
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchTaxes()

    return () => {
      isMounted = false
    }
  }, [searchTerm])

  const handleOpenDeleteDialog = (tax) => {
    setTaxPendingDelete(tax)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }

    setTaxPendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const taxID = String(taxPendingDelete?.tax_id ?? '').trim()
    if (!taxID) {
      setTaxPendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deleteTax(taxID)
      setTaxes((previousTaxes) => previousTaxes.filter((tax) => tax.tax_id !== taxID))
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Tax deleted successfully.')
      setTaxPendingDelete(null)
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Taxes</h1>
        <Link
          to="/admin/configurations/taxes/new"
          className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">Manage tax structures and computation methods.</p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by tax name or computation unit"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        <div className="grid grid-cols-[1fr_1fr_1fr_120px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
          <span>Tax Name</span>
          <span>Tax Computation Unit</span>
          <span>Tax Computation Value</span>
          <span className="text-right">Action</span>
        </div>

        {isLoading ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading taxes...</div>
        ) : taxes.length === 0 ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
            {searchTerm
              ? 'No taxes match your search.'
              : 'No taxes found yet. Click New to create a tax configuration.'}
          </div>
        ) : (
          <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
            {taxes.map((tax) => {
              const taxUnitMeta = getTaxUnitMeta(tax.tax_computation_unit)

              return (
                <div key={tax.tax_id} className="grid grid-cols-[1fr_1fr_1fr_120px] gap-4 px-4 py-4 text-sm text-[var(--navy)]">
                  <div className="font-semibold">{tax.tax_name}</div>
                  <div>{taxUnitMeta.label}</div>
                  <div>{formatTaxValueWithUnit(tax.tax_computation_value, tax.tax_computation_unit)}</div>
                  <div className="flex items-center justify-end gap-2">
                    <Link
                      to={`/admin/configurations/taxes/${tax.tax_id}`}
                      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                      title="Edit"
                      aria-label="Edit tax"
                    >
                      <FiEdit2 className="h-4 w-4" />
                    </Link>

                    <button
                      type="button"
                      onClick={() => handleOpenDeleteDialog(tax)}
                      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
                      title="Delete"
                      aria-label="Delete tax"
                    >
                      <FiTrash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {taxPendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Tax</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{taxPendingDelete.tax_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              This action will permanently remove the tax data.
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

export default TaxesPage
