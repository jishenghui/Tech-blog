import { API_BASE } from './config'

let accessToken = ''
let refreshing: Promise<boolean> | null = null
let expiredHandler: (() => void) | null = null

export function getAccessToken(): string {
  return accessToken
}

export function setAccessToken(token: string): void {
  accessToken = token
}

export function clearAccessToken(): void {
  accessToken = ''
}

export function setSessionExpiredHandler(handler: (() => void) | null): void {
  expiredHandler = handler
}

export function notifySessionExpired(): void {
  clearAccessToken()
  expiredHandler?.()
}

async function doRefresh(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!response.ok) {
      return false
    }
    const body = (await response.json()) as {
      code: number
      data?: { access_token?: string }
    }
    if (body.code !== 0 || !body.data?.access_token) {
      return false
    }
    setAccessToken(body.data.access_token)
    return true
  } catch {
    return false
  }
}

export function refreshAccessToken(): Promise<boolean> {
  if (!refreshing) {
    refreshing = doRefresh().finally(() => {
      refreshing = null
    })
  }
  return refreshing
}
