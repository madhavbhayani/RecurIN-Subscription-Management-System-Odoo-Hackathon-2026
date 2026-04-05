import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import ToastMessage from '../../../../components/common/ToastMessage'
import { deleteDiscount, listDiscounts } from '../../../../services/discountApi'

function formatAmount(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '0.00'
  }

  return numericValue.toFixed(2)
}

function formatDiscountValue(discountUnit, discountValue) {
  if (discountUnit === 'Percentage') {
    return `${formatAmount(discountValue)}%`
  }

  if (discountUnit === 'Fixed Price') {
    return `build ${formatAmount(discountValue)}`
  }

  return formatAmount(discountValue)
}

function formatPurchaseRange(minimumPurchase, maximumPurchase) {
  return `build ${formatAmount(minimumPurchase)} - build ${formatAmount(maximumPurchase)}`
}

function formatLimitUsage(discount) {
  if (!discount?.is_limit) {
    return 'Not Limited'
  }

  const limitUsers = Number(discount?.limit_users ?? 0)
  const appliedUserCount = Number(discount?.applied_user_count ?? 0)
  if (!Number.isFinite(limitUsers) || limitUsers <= 0) {
    return 'Limited'
  }

  return `${appliedUserCount}/${limitUsers}`
}

function getStatusMeta(discount) {
  const status = String(discount?.status ?? '').trim().toLowerCase()

  if (status === 'active') {
    return {
      label: 'Active',
      className: 'border-emerald-200 bg-emerald-50 text-emerald-700',
    }
  }

  const limitUsers = Number(discount?.limit_users ?? 0)
  const appliedUserCount = Number(discount?.applied_user_count ?? 0)
  const isLimitReached = discount?.is_limit && limitUsers > 0 && appliedUserCount >= limitUsers

  return {
    label: isLimitReached ? 'Inactive (Limit Reached)' : 'Inactive',
    className: 'border-red-200 bg-red-50 text-red-700',
  }
}

function DiscountPage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [discounts, setDiscounts] = useState([])
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [discountPendingDelete, setDiscountPendingDelete] = useState(null)
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

    const fetchDiscounts = async () => {
      setIsLoading(true)
      try {
        const response = await listDiscounts(searchTerm)
        if (!isMounted) {
          return
        }

        setDiscounts(Array.isArray(response?.discounts) ? response.discounts : [])
      } catch (error) {
        if (!isMounted) {
          return
        }

        setDiscounts([])
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchDiscounts()

    return () => {
      isMounted = false
    }
  }, [searchTerm])

  const handleOpenDeleteDialog = (discount) => {
    setDiscountPendingDelete(discount)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }

    setDiscountPendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const discountID = String(discountPendingDelete?.discount_id ?? '').trim()
    if (!discountID) {
      setDiscountPendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deleteDiscount(discountID)
      setDiscounts((previousDiscounts) => previousDiscounts.filter((discount) => discount.discount_id !== discountID))
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Discount deleted successfully.')
      setDiscountPendingDelete(null)
    } catch (error) {
      setToastVariant('error')
      setToastMessage(error.message)
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <div className="rounded-2xl border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-5 sm:p-8">
      <ToastMessage message={toastMessage} variant={toastVariant} onClose={() => setToastMessage('')} />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Discount</h1>
        <Link
          to="/admin/configurations/discount/new"
          className="inline-flex h-10 w-full items-center justify-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000] sm:w-auto"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Define discount value, product applicability, purchase range, date range, user limits, and active status.
      </p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by discount name or unit"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        {isLoading ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading discounts...</div>
        ) : discounts.length === 0 ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
            {searchTerm
              ? 'No discounts match your search.'
              : 'No discounts found yet. Click New to create one.'}
          </div>
        ) : (
          <>
            <div className="hidden grid-cols-[1.35fr_1fr_1.2fr_1fr_0.9fr_1fr_110px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)] md:grid">
              <span>Discount Name</span>
              <span>Unit / Value</span>
              <span>Purchase Range</span>
              <span>Validity</span>
              <span>Limit Users</span>
              <span>Status</span>
              <span className="text-right">Action</span>
            </div>

            <div className="hidden divide-y divide-[color:rgba(0,0,128,0.08)] md:block">
              {discounts.map((discount) => {
                const statusMeta = getStatusMeta(discount)

                return (
                  <div
                    key={discount.discount_id}
                    className="grid grid-cols-[1.35fr_1fr_1.2fr_1fr_0.9fr_1fr_110px] gap-4 px-4 py-4 text-sm text-[var(--navy)]"
                  >
                    <div>
                      <p className="font-semibold">{discount.discount_name}</p>
                      <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.64)]">
                        Products: {Number(discount.product_count ?? 0)}
                      </p>
                    </div>
                    <div>
                      <p>{discount.discount_unit}</p>
                      <p className="text-xs text-[color:rgba(0,0,128,0.64)]">
                        {formatDiscountValue(discount.discount_unit, discount.discount_value)}
                      </p>
                    </div>
                    <div>{formatPurchaseRange(discount.minimum_purchase, discount.maximum_purchase)}</div>
                    <div>
                      <p>{discount.start_date}</p>
                      <p className="text-xs text-[color:rgba(0,0,128,0.64)]">to {discount.end_date}</p>
                    </div>
                    <div>{formatLimitUsage(discount)}</div>
                    <div>
                      <span className={`inline-flex rounded-full border px-2.5 py-1 text-xs font-semibold ${statusMeta.className}`}>
                        {statusMeta.label}
                      </span>
                    </div>
                    <div className="flex items-center justify-end gap-2">
                      <Link
                        to={`/admin/configurations/discount/${discount.discount_id}`}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                        title="Edit"
                        aria-label="Edit discount"
                      >
                        <FiEdit2 className="h-4 w-4" />
                      </Link>

                      <button
                        type="button"
                        onClick={() => handleOpenDeleteDialog(discount)}
                        className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
                        title="Delete"
                        aria-label="Delete discount"
                      >
                        <FiTrash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>

            <div className="space-y-3 p-3 md:hidden">
              {discounts.map((discount) => {
                const statusMeta = getStatusMeta(discount)

                return (
                  <div
                    key={`mobile-${discount.discount_id}`}
                    className="rounded-lg border border-[color:rgba(0,0,128,0.14)] bg-[var(--white)] p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="text-sm font-semibold text-[var(--navy)]">{discount.discount_name}</p>
                        <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.64)]">
                          {discount.discount_unit} • {formatDiscountValue(discount.discount_unit, discount.discount_value)}
                        </p>
                      </div>
                      <span className={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-semibold ${statusMeta.className}`}>
                        {statusMeta.label}
                      </span>
                    </div>

                    <div className="mt-3 grid grid-cols-1 gap-2 text-xs text-[color:rgba(0,0,128,0.8)]">
                      <p><span className="font-semibold text-[var(--navy)]">Purchase:</span> {formatPurchaseRange(discount.minimum_purchase, discount.maximum_purchase)}</p>
                      <p><span className="font-semibold text-[var(--navy)]">Validity:</span> {discount.start_date} to {discount.end_date}</p>
                      <p><span className="font-semibold text-[var(--navy)]">Limit Users:</span> {formatLimitUsage(discount)}</p>
                      <p><span className="font-semibold text-[var(--navy)]">Products:</span> {Number(discount.product_count ?? 0)}</p>
                    </div>

                    <div className="mt-4 flex items-center gap-2">
                      <Link
                        to={`/admin/configurations/discount/${discount.discount_id}`}
                        className="inline-flex h-9 flex-1 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                        title="Edit"
                        aria-label="Edit discount"
                      >
                        <FiEdit2 className="mr-2 h-4 w-4" />
                        Edit
                      </Link>

                      <button
                        type="button"
                        onClick={() => handleOpenDeleteDialog(discount)}
                        className="inline-flex h-9 flex-1 items-center justify-center rounded-lg border border-red-300 text-sm font-semibold text-red-600 transition-colors duration-200 hover:bg-red-50"
                        title="Delete"
                        aria-label="Delete discount"
                      >
                        <FiTrash2 className="mr-2 h-4 w-4" />
                        Delete
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          </>
        )}
      </div>

      {discountPendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Discount</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{discountPendingDelete.discount_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              This action will permanently remove the discount and all related product mappings.
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

export default DiscountPage
