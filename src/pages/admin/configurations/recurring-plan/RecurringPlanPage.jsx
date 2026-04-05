import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import Pagination from '../../../../components/common/Pagination'
import ToastMessage from '../../../../components/common/ToastMessage'
import { deleteRecurringPlan, listRecurringPlans } from '../../../../services/recurringPlanApi'
import { buildPaginationState, createInitialPagination } from '../../../../utils/pagination'

function RecurringPlanPage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [recurringPlans, setRecurringPlans] = useState([])
  const [currentPage, setCurrentPage] = useState(1)
  const [pagination, setPagination] = useState(() => createInitialPagination())
  const [refreshKey, setRefreshKey] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [recurringPlanPendingDelete, setRecurringPlanPendingDelete] = useState(null)
  const [isDeleting, setIsDeleting] = useState(false)

  useEffect(() => {
    const debounceTimer = window.setTimeout(() => {
      setSearchTerm(searchInput.trim())
      setCurrentPage(1)
    }, 300)

    return () => {
      window.clearTimeout(debounceTimer)
    }
  }, [searchInput])

  useEffect(() => {
    let isMounted = true

    const fetchRecurringPlans = async () => {
      setIsLoading(true)
      try {
        const response = await listRecurringPlans(searchTerm, false, currentPage)
        if (!isMounted) {
          return
        }

        setRecurringPlans(Array.isArray(response?.recurring_plans) ? response.recurring_plans : [])
        const paginationState = buildPaginationState(response?.pagination, currentPage)
        setPagination(paginationState)

        if (paginationState.total_pages > 0 && currentPage > paginationState.total_pages) {
          setCurrentPage(paginationState.total_pages)
        }
      } catch (error) {
        if (!isMounted) {
          return
        }

        setRecurringPlans([])
        setPagination(createInitialPagination(currentPage))
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchRecurringPlans()

    return () => {
      isMounted = false
    }
  }, [searchTerm, currentPage, refreshKey])

  const handleOpenDeleteDialog = (recurringPlan) => {
    setRecurringPlanPendingDelete(recurringPlan)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }

    setRecurringPlanPendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const recurringPlanID = String(recurringPlanPendingDelete?.recurring_plan_id ?? '').trim()
    if (!recurringPlanID) {
      setRecurringPlanPendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deleteRecurringPlan(recurringPlanID)
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Recurring plan deleted successfully.')
      setRecurringPlanPendingDelete(null)
      setRefreshKey((previousKey) => previousKey + 1)
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Recurring Plan</h1>
        <Link
          to="/admin/configurations/recurring-plan/new"
          className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">
        Manage recurring billing periods and cycle behavior.
      </p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by recurring plan name"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        <div className="grid grid-cols-[1.6fr_1fr_1fr_120px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
          <span>Recurring Name</span>
          <span>Billing Period</span>
          <span>Products</span>
          <span className="text-right">Action</span>
        </div>

        {isLoading ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading recurring plans...</div>
        ) : recurringPlans.length === 0 ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
            {searchTerm
              ? 'No recurring plans match your search.'
              : 'No recurring plans found yet. Click New to create one.'}
          </div>
        ) : (
          <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
            {recurringPlans.map((recurringPlan) => (
              <div key={recurringPlan.recurring_plan_id} className="grid grid-cols-[1.6fr_1fr_1fr_120px] gap-4 px-4 py-4 text-sm text-[var(--navy)]">
                <div className="font-semibold">{recurringPlan.recurring_name}</div>
                <div>{recurringPlan.billing_period}</div>
                <div>{Number(recurringPlan.product_count ?? 0)}</div>
                <div className="flex items-center justify-end gap-2">
                  <Link
                    to={`/admin/configurations/recurring-plan/${recurringPlan.recurring_plan_id}`}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                    title="Edit"
                    aria-label="Edit recurring plan"
                  >
                    <FiEdit2 className="h-4 w-4" />
                  </Link>

                  <button
                    type="button"
                    onClick={() => handleOpenDeleteDialog(recurringPlan)}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
                    title="Delete"
                    aria-label="Delete recurring plan"
                  >
                    <FiTrash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <Pagination
        currentPage={currentPage}
        totalPages={pagination.total_pages}
        onPageChange={setCurrentPage}
        isDisabled={isLoading}
      />

      {recurringPlanPendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Recurring Plan</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{recurringPlanPendingDelete.recurring_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              This action will permanently remove the recurring plan.
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

export default RecurringPlanPage
