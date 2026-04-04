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

export function createPayPalCheckoutOrder() {
  return requestWithAuth('/api/v1/payments/paypal/create-order', {
    method: 'POST',
  })
}

export function capturePayPalCheckoutOrder(capturePayload) {
  const normalizedPayload = typeof capturePayload === 'string'
    ? { order_id: capturePayload }
    : (capturePayload && typeof capturePayload === 'object' ? capturePayload : {})

  return requestWithAuth('/api/v1/payments/paypal/capture-order', {
    method: 'POST',
    payload: normalizedPayload,
  })
}
