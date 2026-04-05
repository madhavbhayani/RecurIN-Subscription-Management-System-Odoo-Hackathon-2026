import { useEffect, useState } from 'react'
import { FiEdit2, FiTrash2 } from 'react-icons/fi'
import { Link } from 'react-router-dom'
import Pagination from '../../../components/common/Pagination'
import ToastMessage from '../../../components/common/ToastMessage'
import { deleteProduct, listProducts } from '../../../services/productApi'
import { buildPaginationState, createInitialPagination } from '../../../utils/pagination'

const CURRENCY_SYMBOL = '$'

function formatPrice(value) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue)) {
    return `${CURRENCY_SYMBOL}0.00`
  }

  return `${CURRENCY_SYMBOL}${numericValue.toFixed(2)}`
}

function ProductsPage() {
  const [searchInput, setSearchInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [products, setProducts] = useState([])
  const [currentPage, setCurrentPage] = useState(1)
  const [pagination, setPagination] = useState(() => createInitialPagination())
  const [refreshKey, setRefreshKey] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVariant, setToastVariant] = useState('error')
  const [productPendingDelete, setProductPendingDelete] = useState(null)
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

    const fetchProducts = async () => {
      setIsLoading(true)
      try {
        const response = await listProducts(searchTerm, currentPage)
        if (!isMounted) {
          return
        }

        setProducts(Array.isArray(response?.products) ? response.products : [])
        const paginationState = buildPaginationState(response?.pagination, currentPage)
        setPagination(paginationState)

        if (paginationState.total_pages > 0 && currentPage > paginationState.total_pages) {
          setCurrentPage(paginationState.total_pages)
        }
      } catch (error) {
        if (!isMounted) {
          return
        }

        setProducts([])
        setPagination(createInitialPagination(currentPage))
        setToastVariant('error')
        setToastMessage(error.message)
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    fetchProducts()

    return () => {
      isMounted = false
    }
  }, [searchTerm, currentPage, refreshKey])

  const handleOpenDeleteDialog = (product) => {
    setProductPendingDelete(product)
  }

  const handleCloseDeleteDialog = () => {
    if (isDeleting) {
      return
    }

    setProductPendingDelete(null)
  }

  const handleConfirmDelete = async () => {
    const productID = String(productPendingDelete?.product_id ?? '').trim()
    if (!productID) {
      setProductPendingDelete(null)
      return
    }

    setIsDeleting(true)
    try {
      const response = await deleteProduct(productID)
      setToastVariant('success')
      setToastMessage(response?.message ?? 'Product deleted successfully.')
      setProductPendingDelete(null)
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
        <h1 className="[font-family:var(--font-display)] text-3xl font-bold text-[var(--navy)]">Products</h1>
        <Link
          to="/admin/products/new"
          className="inline-flex h-10 items-center rounded-lg bg-[var(--orange)] px-5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-[#e56000]"
        >
          New
        </Link>
      </div>

      <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.76)]">Create and manage products with taxes and variants.</p>

      <div className="mt-5">
        <input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder="Search by product name"
          className="w-full rounded-lg border border-[color:rgba(0,0,128,0.2)] px-4 py-2.5 text-sm text-[var(--navy)] outline-none placeholder:text-[color:rgba(0,0,128,0.45)] focus:border-[var(--orange)]"
        />
      </div>

      <div className="mt-6 overflow-hidden rounded-xl border border-[color:rgba(0,0,128,0.12)]">
        <div className="grid grid-cols-[1.25fr_1fr_1fr_1.1fr_120px] gap-4 bg-[rgba(0,0,128,0.04)] px-4 py-3 text-xs font-semibold uppercase tracking-[0.08em] text-[var(--navy)]">
          <span>Product Name</span>
          <span>Sales Price</span>
          <span>Cost</span>
          <span>Recurring Plan</span>
          <span className="text-right">Action</span>
        </div>

        {isLoading ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">Loading products...</div>
        ) : products.length === 0 ? (
          <div className="px-4 py-8 text-sm text-[color:rgba(0,0,128,0.66)]">
            {searchTerm
              ? 'No products match your search.'
              : 'No products found yet. Click New to create your first product.'}
          </div>
        ) : (
          <div className="divide-y divide-[color:rgba(0,0,128,0.08)]">
            {products.map((product) => (
              <div key={product.product_id} className="grid grid-cols-[1.25fr_1fr_1fr_1.1fr_120px] gap-4 px-4 py-4 text-sm text-[var(--navy)]">
                <div className="font-semibold">{product.product_name}</div>
                <div>{formatPrice(product.sales_price)}</div>
                <div>{formatPrice(product.cost_price)}</div>
                <div>
                  {String(product?.recurring_name ?? '').trim()
                    ? `${product.recurring_name}${String(product?.billing_period ?? '').trim() ? ` (${product.billing_period})` : ''}`
                    : '-'}
                </div>
                <div className="flex items-center justify-end gap-2">
                  <Link
                    to={`/admin/products/${product.product_id}`}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[color:rgba(0,0,128,0.2)] text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.05)]"
                    title="Edit"
                    aria-label="Edit product"
                  >
                    <FiEdit2 className="h-4 w-4" />
                  </Link>

                  <button
                    type="button"
                    onClick={() => handleOpenDeleteDialog(product)}
                    className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-red-300 text-red-600 transition-colors duration-200 hover:bg-red-50"
                    title="Delete"
                    aria-label="Delete product"
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

      {productPendingDelete && (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/35 p-4">
          <div className="w-full max-w-md rounded-xl border border-[color:rgba(0,0,128,0.16)] bg-[var(--white)] p-6 shadow-[0_16px_40px_rgba(0,0,0,0.2)]">
            <h2 className="[font-family:var(--font-display)] text-xl font-bold text-[var(--navy)]">Delete Product</h2>
            <p className="mt-2 text-sm text-[color:rgba(0,0,128,0.78)]">
              Are you sure you want to delete{' '}
              <span className="font-semibold text-[var(--navy)]">{productPendingDelete.product_name}</span>
              ?
            </p>
            <p className="mt-1 text-xs text-[color:rgba(0,0,128,0.62)]">
              This action will permanently remove the product and its related mapping data.
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

export default ProductsPage
