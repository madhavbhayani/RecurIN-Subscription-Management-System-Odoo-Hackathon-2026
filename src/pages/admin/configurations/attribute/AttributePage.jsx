import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import Pagination from '../../../../components/common/Pagination'
import ToastMessage from '../../../../components/common/ToastMessage'
import { deleteAttribute, listAttributes } from '../../../../services/attributeApi'
import { buildPaginationState, createInitialPagination } from '../../../../utils/pagination'

function formatDefaultPrice(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return '0.00'
  }

  return numericValue.toFixed(2)
}

function AttributePage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [attributes, setAttributes] = useState([])
  const [currentPage, setCurrentPage] = useState(1)
  const [pagination, setPagination] = useState(() => createInitialPagination())
  const [refreshKey, setRefreshKey] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [attributePendingDelete, setAttributePendingDelete] = useState(null)
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

    const fetchAttributes = async () => {
      setIsLoading(true)
      try {
        const response = await listAttributes(searchTerm, currentPage)
        if (!isMounted) {
          return
        }

        setAttributes(Array.isArray(response?.attributes) ? response.attributes : [])
        const paginationState = buildPaginationState(response?.pagination, currentPage)
        setPagination(paginationState)

        if (paginationState.total_pages > 0 && currentPage > paginationState.total_pages) {
          setCurrentPage(paginationState.total_pages)
        }
      } catch (error) {
        if (!isMounted) {
          return
        }

        setAttributes([])
        setPagination(createInitialPagination(currentPage))
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchAttributes()

    return () => {
      isMounted = false
    }
  }, [searchTerm, currentPage, refreshKey])

  const handleOpenDeleteDialog = (attribute) => {
    setAttributePendingDelete(attribute)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }
    setAttributePendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const attributeID = String(attributePendingDelete?.attribute_id ?? '').trim()
    if (!attributeID) {
      setAttributePendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deleteAttribute(attributeID)
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Attribute deleted successfully.')
      setAttributePendingDelete(null)
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Attribute</h1>
        <Link
          to="/admin/configurations/attribute/new"
          className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">Manage product and subscription attributes.</p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by attribute name or value"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        <div className="grid grid-cols-[1fr_1fr_1.6fr_120px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
          <span>Attribute Name</span>
          <span>Attribute Values</span>
          <span className="whitespace-nowrap">Attribute Default Extra Pricing</span>
          <span className="text-right">Action</span>
        </div>

        {isLoading ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading attributes...</div>
        ) : attributes.length === 0 ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
            {searchTerm
              ? 'No attributes match your search.'
              : 'No attributes found yet. Click New to create an attribute with one or more values and default extra prices.'}
          </div>
        ) : (
          <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
            {attributes.map((attribute) => {
              const values = Array.isArray(attribute?.values) ? attribute.values : []

              return (
                <div key={attribute.attribute_id} className="grid grid-cols-[1fr_1fr_1.6fr_120px] gap-4 px-4 py-4 text-sm text-[var(--navy)]">
                  <div className="font-semibold">{attribute.attribute_name}</div>
                  <div className="space-y-1">
                    {values.length > 0 ? (
                      values.map((value) => (
                        <div key={value.attribute_value_id ?? `${attribute.attribute_id}-${value.attribute_value}`}>
                          {value.attribute_value}
                        </div>
                      ))
                    ) : (
                      <span>-</span>
                    )}
                  </div>
                  <div className="space-y-1">
                    {values.length > 0 ? (
                      values.map((value) => (
                        <div key={value.attribute_value_id ?? `${attribute.attribute_id}-${value.attribute_value}-price`}>
                          {formatDefaultPrice(value.default_extra_price)}
                        </div>
                      ))
                    ) : (
                      <span>-</span>
                    )}
                  </div>
                  <div className="flex items-center justify-end gap-2">
                    <Link
                      to={`/admin/configurations/attribute/${attribute.attribute_id}`}
                      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                      title="Edit"
                      aria-label="Edit attribute"
                    >
                      <FiEdit2 className="h-4 w-4" />
                    </Link>

                    <button
                      type="button"
                      onClick={() => handleOpenDeleteDialog(attribute)}
                      className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
                      title="Delete"
                      aria-label="Delete attribute"
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

      <Pagination
        currentPage={currentPage}
        totalPages={pagination.total_pages}
        onPageChange={setCurrentPage}
        isDisabled={isLoading}
      />

      {attributePendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Attribute</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{attributePendingDelete.attribute_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              This will permanently remove the attribute and its values.
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

export default AttributePage
