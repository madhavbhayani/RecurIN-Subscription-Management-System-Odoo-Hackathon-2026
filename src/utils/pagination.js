export const ADMIN_PAGE_SIZE = 30

export function createInitialPagination(page = 1) {
  return {
    page,
    per_page: ADMIN_PAGE_SIZE,
    total_records: 0,
    total_pages: 0,
  }
}

function toPositiveInteger(value, fallback) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue) || numericValue < 1) {
    return fallback
  }

  return Math.trunc(numericValue)
}

function toNonNegativeInteger(value, fallback) {
  const numericValue = Number(value)
  if (!Number.isFinite(numericValue) || numericValue < 0) {
    return fallback
  }

  return Math.trunc(numericValue)
}

export function buildPaginationState(rawPagination, fallbackPage = 1) {
  const page = toPositiveInteger(rawPagination?.page, fallbackPage)
  const perPage = toPositiveInteger(rawPagination?.per_page, ADMIN_PAGE_SIZE)
  const totalRecords = toNonNegativeInteger(rawPagination?.total_records, 0)
  const derivedTotalPages = totalRecords > 0 ? Math.ceil(totalRecords / perPage) : 0
  const totalPages = toNonNegativeInteger(rawPagination?.total_pages, derivedTotalPages)

  return {
    page,
    per_page: perPage,
    total_records: totalRecords,
    total_pages: totalPages,
  }
}
