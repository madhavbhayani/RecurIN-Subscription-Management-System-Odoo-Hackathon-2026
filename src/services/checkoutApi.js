import { getAuthSession } from './session'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

async function requestWithAuth(path, options = {}) {
  const {
    method = 'GET',
    payload,
  } = options

  const session = getAuthSession()
  if (!session?.token) {
    throw new Error('Please log in to continue.')
  }

  const headers = {
    Authorization: `Bearer ${session.token}`,
  }

  const fetchOptions = {
    method,
    headers,
  }

  if (payload !== undefined) {
    headers['Content-Type'] = 'application/json'
    fetchOptions.body = JSON.stringify(payload)
  }

  let response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, fetchOptions)
  } catch {
    throw new Error(`Unable to connect to backend at ${API_BASE_URL}. Please make sure the Go server is running.`)
  }

  const rawText = await response.text()
  const normalizedText = rawText.trim()

  let data = null
  try {
    data = normalizedText ? JSON.parse(normalizedText) : null
  } catch {
    data = null
  }

  if (!response.ok) {
    const message = data?.message ?? data?.error ?? normalizedText ?? 'Request failed'
    throw new Error(message)
  }

  return data
}

function parseDownloadFileName(contentDispositionHeader) {
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

  const plainMatch = headerValue.match(/filename="?([^";]+)"?/i)
  if (plainMatch && plainMatch[1]) {
    return String(plainMatch[1]).trim()
  }

  return ''
}

function triggerBrowserDownload(blob, fileName) {
  const downloadBlobURL = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = downloadBlobURL
  anchor.download = fileName
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(downloadBlobURL)
}

async function requestPDFWithAuth(path, fallbackFileName) {
  const session = getAuthSession()
  if (!session?.token) {
    throw new Error('Please log in to continue.')
  }

  const headers = {
    Authorization: `Bearer ${session.token}`,
  }

  let response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: 'GET',
      headers,
    })
  } catch {
    throw new Error(`Unable to connect to backend at ${API_BASE_URL}. Please make sure the Go server is running.`)
  }

  if (!response.ok) {
    let message = 'Request failed'
    try {
      const contentType = String(response.headers.get('content-type') ?? '').toLowerCase()
      if (contentType.includes('application/json')) {
        const data = await response.json()
        message = data?.message ?? data?.error ?? message
      } else {
        const textPayload = (await response.text()).trim()
        if (textPayload) {
          message = textPayload
        }
      }
    } catch {
      // Ignore parse failures and keep fallback error message.
    }

    throw new Error(message)
  }

  const documentBlob = await response.blob()
  const resolvedFileName = parseDownloadFileName(response.headers.get('content-disposition')) || fallbackFileName
  triggerBrowserDownload(documentBlob, resolvedFileName)
}

export function getMyCheckoutProfile() {
  return requestWithAuth('/api/v1/users/me', {
    method: 'GET',
  })
}

export function updateMyCheckoutAddress(address) {
  return requestWithAuth('/api/v1/users/me/address', {
    method: 'PATCH',
    payload: {
      address,
    },
  })
}

export function createPayPalCheckoutOrder(options = {}) {
  const normalizedSubscriptionID = String(options?.subscription_id ?? '').trim()

  const payload = normalizedSubscriptionID
    ? { subscription_id: normalizedSubscriptionID }
    : undefined

  return requestWithAuth('/api/v1/payments/paypal/create-order', {
    method: 'POST',
    payload,
  })
}

export function listMySubscriptions() {
  return requestWithAuth('/api/v1/users/me/subscriptions', {
    method: 'GET',
  })
}

export function respondToQuotation(subscriptionID, action) {
  const normalizedSubscriptionID = String(subscriptionID ?? '').trim()
  if (!normalizedSubscriptionID) {
    throw new Error('Subscription ID is required.')
  }

  const normalizedAction = String(action ?? '').trim().toLowerCase()
  if (normalizedAction !== 'accept' && normalizedAction !== 'reject') {
    throw new Error('Action must be either accept or reject.')
  }

  return requestWithAuth(`/api/v1/users/me/subscriptions/${encodeURIComponent(normalizedSubscriptionID)}/respond`, {
    method: 'PATCH',
    payload: {
      action: normalizedAction,
    },
  })
}

export function downloadMySubscriptionInvoicePdf(subscriptionID, fallbackFileName = 'Invoice-RecurIN.pdf') {
  const normalizedSubscriptionID = String(subscriptionID ?? '').trim()
  if (!normalizedSubscriptionID) {
    throw new Error('Subscription ID is required.')
  }

  return requestPDFWithAuth(
    `/api/v1/users/me/subscriptions/${encodeURIComponent(normalizedSubscriptionID)}/invoice`,
    fallbackFileName,
  )
}

export function downloadMySubscriptionQuotationPdf(subscriptionID, fallbackFileName = 'Quotation-RecurIN.pdf') {
  const normalizedSubscriptionID = String(subscriptionID ?? '').trim()
  if (!normalizedSubscriptionID) {
    throw new Error('Subscription ID is required.')
  }

  return requestPDFWithAuth(
    `/api/v1/users/me/subscriptions/${encodeURIComponent(normalizedSubscriptionID)}/quotation`,
    fallbackFileName,
  )
}

export function capturePayPalCheckoutOrder(capturePayload) {
  const normalizedPayload = typeof capturePayload === 'string'
    ? { order_id: capturePayload }
    : (capturePayload && typeof capturePayload === 'object' ? capturePayload : {})

  const normalizedSubscriptionID = String(normalizedPayload.subscription_id ?? '').trim()
  if (normalizedSubscriptionID) {
    normalizedPayload.subscription_id = normalizedSubscriptionID
  } else {
    delete normalizedPayload.subscription_id
  }

  return requestWithAuth('/api/v1/payments/paypal/capture-order', {
    method: 'POST',
    payload: normalizedPayload,
  })
}
