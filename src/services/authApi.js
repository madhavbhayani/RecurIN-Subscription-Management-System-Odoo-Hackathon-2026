const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

async function request(path, payload) {
  let response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    })
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

export function signupUser(payload) {
  return request('/api/v1/auth/signup', payload)
}

export function loginUser(payload) {
  return request('/api/v1/auth/login', payload)
}
