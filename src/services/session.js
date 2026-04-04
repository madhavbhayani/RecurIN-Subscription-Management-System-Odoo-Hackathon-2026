const ACCESS_TOKEN_COOKIE = 'recurin_access_token'
const USER_COOKIE = 'recurin_user'
const SESSION_MAX_AGE_SECONDS = 60 * 60

export function saveAuthSession(authResponse) {
  const token = String(authResponse?.access_token ?? '').trim()
  if (!token) {
    return
  }

  setCookie(ACCESS_TOKEN_COOKIE, token, SESSION_MAX_AGE_SECONDS)

  if (authResponse?.user) {
    setCookie(USER_COOKIE, JSON.stringify(authResponse.user), SESSION_MAX_AGE_SECONDS)
  }

  // Clean up legacy localStorage keys from previous implementations.
  localStorage.removeItem(ACCESS_TOKEN_COOKIE)
  localStorage.removeItem(USER_COOKIE)
}

export function clearAuthSession() {
  deleteCookie(ACCESS_TOKEN_COOKIE)
  deleteCookie(USER_COOKIE)

  // Clean up legacy localStorage keys from previous implementations.
  localStorage.removeItem(ACCESS_TOKEN_COOKIE)
  localStorage.removeItem(USER_COOKIE)
}

export function getAuthSession() {
  const token = getCookie(ACCESS_TOKEN_COOKIE)
  if (!token) {
    return null
  }

  const payload = parseJwtPayload(token)
  if (!payload || isTokenExpired(payload)) {
    clearAuthSession()
    return null
  }

  const user = readStoredUser() ?? userFromPayload(payload)

  return {
    token,
    user,
  }
}

function setCookie(name, value, maxAgeSeconds) {
  document.cookie = `${name}=${encodeURIComponent(value)}; Max-Age=${maxAgeSeconds}; Path=/; SameSite=Lax`
}

function getCookie(name) {
  const pattern = new RegExp(`(?:^|; )${name}=([^;]*)`)
  const match = document.cookie.match(pattern)
  if (!match) {
    return null
  }

  return decodeURIComponent(match[1])
}

function deleteCookie(name) {
  document.cookie = `${name}=; Max-Age=0; Path=/; SameSite=Lax`
}

function readStoredUser() {
  const userText = getCookie(USER_COOKIE)
  if (!userText) {
    return null
  }

  try {
    return JSON.parse(userText)
  } catch {
    return null
  }
}

function userFromPayload(payload) {
  return {
    id: payload.uid ?? payload.sub ?? '',
    role: normalizeRole(payload.role),
  }
}

function parseJwtPayload(token) {
  const tokenParts = token.split('.')
  if (tokenParts.length < 2) {
    return null
  }

  try {
    const payloadText = atob(base64UrlToBase64(tokenParts[1]))
    return JSON.parse(payloadText)
  } catch {
    return null
  }
}

function base64UrlToBase64(base64Url) {
  const normalized = base64Url.replace(/-/g, '+').replace(/_/g, '/')
  const padding = '='.repeat((4 - (normalized.length % 4)) % 4)
  return normalized + padding
}

function isTokenExpired(payload) {
  const expiryEpochSeconds = Number(payload?.exp)
  if (!Number.isFinite(expiryEpochSeconds) || expiryEpochSeconds <= 0) {
    return true
  }

  return Date.now() >= expiryEpochSeconds * 1000
}

function normalizeRole(role) {
  const normalizedRole = String(role ?? '').trim().toLowerCase()

  if (normalizedRole === 'admin') {
    return 'Admin'
  }
  if (normalizedRole === 'internal-user' || normalizedRole === 'internal') {
    return 'Internal'
  }

  return 'User'
}
