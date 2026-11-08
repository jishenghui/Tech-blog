import { API_BASE } from './config'
import { getAccessToken, notifySessionExpired, refreshAccessToken } from './session'

export class ApiError extends Error {
  status: number
  code?: number

  constructor(message: string, status: number, code?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  formData?: FormData
  retry?: boolean
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = {}
  const token = getAccessToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  let body: BodyInit | undefined
  if (options.formData) {
    body = options.formData
  } else if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(options.body)
  }

  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      method: options.method ?? 'GET',
      headers,
      body,
      credentials: 'include',
    })
  } catch {
    throw new ApiError('无法连接后端 API，请确认服务已启动', 0)
  }

  if (response.status === 401 && options.retry !== false && getAccessToken()) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      return request<T>(path, { ...options, retry: false })
    }
    notifySessionExpired()
  }

  const text = await response.text()
  let payload: { code?: number; message?: string; data?: unknown } | null = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      // 非 JSON 响应，按 HTTP 状态处理。
    }
  }

  if (!response.ok) {
    throw new ApiError(
      payload?.message || `请求失败（HTTP ${response.status}）`,
      response.status,
      payload?.code,
    )
  }
  if (payload && payload.code !== 0) {
    throw new ApiError(payload.message || '请求失败', response.status, payload.code)
  }
  return (payload?.data ?? null) as T
}

export const api = {
  get<T>(path: string): Promise<T> {
    return request<T>(path)
  },
  post<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, { method: 'POST', body })
  },
  put<T>(path: string, body?: unknown): Promise<T> {
    return request<T>(path, { method: 'PUT', body })
  },
  del<T>(path: string): Promise<T> {
    return request<T>(path, { method: 'DELETE' })
  },
  upload<T>(path: string, formData: FormData): Promise<T> {
    return request<T>(path, { method: 'POST', formData })
  },
}
