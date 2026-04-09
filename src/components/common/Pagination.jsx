function buildVisiblePages(totalPages, currentPage) {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1)
  }

  const visiblePages = [1]
  const windowStart = Math.max(2, currentPage - 1)
  const windowEnd = Math.min(totalPages - 1, currentPage + 1)

  if (windowStart > 2) {
    visiblePages.push('left-ellipsis')
  }

  for (let page = windowStart; page <= windowEnd; page += 1) {
    visiblePages.push(page)
  }

  if (windowEnd < totalPages - 1) {
    visiblePages.push('right-ellipsis')
  }

  visiblePages.push(totalPages)

  return visiblePages
}

function Pagination({ currentPage, totalPages, onPageChange, isDisabled = false }) {
  if (!Number.isFinite(totalPages) || totalPages <= 1) {
    return null
  }

  const safeCurrentPage = Math.min(Math.max(Number(currentPage) || 1, 1), totalPages)
  const pageItems = buildVisiblePages(totalPages, safeCurrentPage)

  return (
    <div className="mt-6 flex flex-wrap items-center justify-between gap-3 border-t border-[color:rgba(0,0,128,0.12)] pt-4">
      <button
        type="button"
        onClick={() => onPageChange(safeCurrentPage - 1)}
        disabled={isDisabled || safeCurrentPage <= 1}
        className="inline-flex h-9 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-3 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)] disabled:cursor-not-allowed disabled:opacity-50"
      >
        Previous
      </button>

      <div className="flex max-w-full items-center gap-1 overflow-x-auto pb-1">
        {pageItems.map((pageItem) => {
          if (typeof pageItem !== 'number') {
            return (
              <span
                key={pageItem}
                className="inline-flex h-9 min-w-[2.25rem] items-center justify-center px-1 text-sm text-[color:rgba(0,0,128,0.55)]"
              >
                ...
              </span>
            )
          }

          const isActivePage = pageItem === safeCurrentPage

          return (
            <button
              key={pageItem}
              type="button"
              onClick={() => onPageChange(pageItem)}
              disabled={isDisabled || isActivePage}
              className={`inline-flex h-9 min-w-[2.25rem] items-center justify-center rounded-lg border px-2.5 text-sm font-semibold transition-colors duration-200 ${isActivePage
                ? 'border-[var(--orange)] bg-[var(--orange)] text-white'
                : 'border-[color:rgba(0,0,128,0.22)] text-[var(--navy)] hover:bg-[rgba(0,0,128,0.04)]'} disabled:cursor-not-allowed disabled:opacity-70`}
            >
              {pageItem}
            </button>
          )
        })}
      </div>

      <button
        type="button"
        onClick={() => onPageChange(safeCurrentPage + 1)}
        disabled={isDisabled || safeCurrentPage >= totalPages}
        className="inline-flex h-9 items-center rounded-lg border border-[color:rgba(0,0,128,0.22)] px-3 text-sm font-semibold text-[var(--navy)] transition-colors duration-200 hover:bg-[rgba(0,0,128,0.04)] disabled:cursor-not-allowed disabled:opacity-50"
      >
        Next
      </button>
    </div>
  )
}

export default Pagination
