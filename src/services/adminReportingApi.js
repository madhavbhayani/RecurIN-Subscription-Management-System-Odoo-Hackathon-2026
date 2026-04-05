import { getAuthSession } from './session'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

function normalizeReportingMonths(months) {
  const parsedMonths = Number(months)
  if (!Number.isInteger(parsedMonths) || parsedMonths <= 0) {
    return 6
  }

  return Math.min(parsedMonths, 24)
}

async function requestWithAuth(path, options = {}) {
  const {
    method = 'GET',
    queryParams = null,
  } = options

  const session = getAuthSession()
  if (!session?.token) {
    throw new Error('Your session has expired. Please log in again.')
  }

  const requestURL = new URL(`${API_BASE_URL}${path}`)
  if (queryParams && typeof queryParams === 'object') {
    Object.entries(queryParams).forEach(([key, value]) => {
      const normalizedValue = String(value ?? '').trim()
      if (normalizedValue !== '') {
        requestURL.searchParams.set(key, normalizedValue)
      }
    })
  }

  let response
  try {
    response = await fetch(requestURL.toString(), {
      method,
      headers: {
        Authorization: `Bearer ${session.token}`,
      },
    })
  } catch {
    throw new Error(`Unable to connect to backend at ${API_BASE_URL}. Please make sure the Go server is running.`)
  }

  const rawResponseText = await response.text()
  const normalizedResponseText = rawResponseText.trim()

  let responseData = null
  try {
    responseData = normalizedResponseText ? JSON.parse(normalizedResponseText) : null
  } catch {
    responseData = null
  }

  if (!response.ok) {
    const errorMessage = responseData?.message ?? responseData?.error ?? normalizedResponseText ?? 'Request failed'
    throw new Error(errorMessage)
  }

  return responseData
}

function parseFileNameFromHeader(contentDispositionHeader) {
  const headerValue = String(contentDispositionHeader ?? '').trim()
  if (!headerValue) {
    return ''
  }

  const utfMatch = headerValue.match(/filename\*=UTF-8''([^;]+)/i)
  if (utfMatch && utfMatch[1]) {
    try {
      return decodeURIComponent(utfMatch[1]).trim()
    } catch {
      return String(utfMatch[1]).trim()
    }
  }

  const fallbackMatch = headerValue.match(/filename="?([^";]+)"?/i)
  if (fallbackMatch && fallbackMatch[1]) {
    return String(fallbackMatch[1]).trim()
  }

  return ''
}

function triggerFileDownload(blobPayload, fileName) {
  const fileURL = URL.createObjectURL(blobPayload)
  const anchor = document.createElement('a')
  anchor.href = fileURL
  anchor.download = fileName
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(fileURL)
}

async function downloadPdfWithAuth(path, fallbackFileName, queryParams = null) {
  const session = getAuthSession()
  if (!session?.token) {
    throw new Error('Your session has expired. Please log in again.')
  }

  const requestURL = new URL(`${API_BASE_URL}${path}`)
  if (queryParams && typeof queryParams === 'object') {
    Object.entries(queryParams).forEach(([key, value]) => {
      const normalizedValue = String(value ?? '').trim()
      if (normalizedValue !== '') {
        requestURL.searchParams.set(key, normalizedValue)
      }
    })
  }

  let response
  try {
    response = await fetch(requestURL.toString(), {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${session.token}`,
      },
    })
  } catch {
    throw new Error(`Unable to connect to backend at ${API_BASE_URL}. Please make sure the Go server is running.`)
  }

  if (!response.ok) {
    let errorMessage = 'Request failed'
    try {
      const contentType = String(response.headers.get('content-type') ?? '').toLowerCase()
      if (contentType.includes('application/json')) {
        const errorPayload = await response.json()
        errorMessage = errorPayload?.message ?? errorPayload?.error ?? errorMessage
      } else {
        const errorText = (await response.text()).trim()
        if (errorText) {
          errorMessage = errorText
        }
      }
    } catch {
      // Ignore parse errors and use fallback message.
    }

    throw new Error(errorMessage)
  }

  const pdfBlob = await response.blob()
  const resolvedFileName = parseFileNameFromHeader(response.headers.get('content-disposition')) || fallbackFileName
  triggerFileDownload(pdfBlob, resolvedFileName)
}

export function getAdminReportingDashboard(months = 6) {
  const normalizedMonths = normalizeReportingMonths(months)
  return requestWithAuth('/api/v1/admin/reporting/dashboard', {
    method: 'GET',
    queryParams: {
      months: normalizedMonths,
    },
  })
}

export function downloadAdminSalesReportPdf(months = 6) {
  const normalizedMonths = normalizeReportingMonths(months)
  return downloadPdfWithAuth('/api/v1/admin/reporting/reports/sales', 'Sales-Report.pdf', {
    months: normalizedMonths,
  })
}

export function downloadAdminModuleFrequencyReportPdf(months = 6) {
  const normalizedMonths = normalizeReportingMonths(months)
  return downloadPdfWithAuth('/api/v1/admin/reporting/reports/modules', 'Module-Frequency-Report.pdf', {
    months: normalizedMonths,
  })
}
