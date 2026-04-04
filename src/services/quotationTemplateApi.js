import { getAuthSession } from './session'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

async function requestWithAuth(path, options = {}) {
  const {
    method = 'GET',
    payload,
    queryParams = null,
  } = options

  const session = getAuthSession()
  if (!session?.token) {
    throw new Error('Your session has expired. Please log in again.')
  }

  const url = new URL(`${API_BASE_URL}${path}`)
  if (queryParams && typeof queryParams === 'object') {
    Object.entries(queryParams).forEach(([key, value]) => {
      const normalizedValue = String(value ?? '').trim()
      if (normalizedValue !== '') {
        url.searchParams.set(key, normalizedValue)
      }
    })
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
    response = await fetch(url.toString(), fetchOptions)
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

export function listQuotationTemplates(search = '') {
  return requestWithAuth('/api/v1/admin/quotations', {
    method: 'GET',
    queryParams: { search },
  })
}

export function createQuotationTemplate(payload) {
  return requestWithAuth('/api/v1/admin/quotations', {
    method: 'POST',
    payload,
  })
}

export function getQuotationTemplateById(quotationId) {
  return requestWithAuth(`/api/v1/admin/quotations/${encodeURIComponent(quotationId)}`, {
    method: 'GET',
  })
}

export function updateQuotationTemplate(quotationId, payload) {
  return requestWithAuth(`/api/v1/admin/quotations/${encodeURIComponent(quotationId)}`, {
    method: 'PUT',
    payload,
  })
}

export function deleteQuotationTemplate(quotationId) {
  return requestWithAuth(`/api/v1/admin/quotations/${encodeURIComponent(quotationId)}`, {
    method: 'DELETE',
  })
}
