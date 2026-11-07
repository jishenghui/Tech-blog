import type { ArticleSummary, Page } from './types'

const API_BASE = (import.meta.env.PUBLIC_API_BASE_URL || 'http://localhost:8080').replace(/\/+$/, '')

export const apiBase = API_BASE

interface Envelope<T> {
  code: number
  message: string
  data: T
}

const cache = new Map<string, Promise<unknown>>()

export function apiGet<T>(path: string): Promise<T> {
  const url = `${API_BASE}${path}`
  const cached = cache.get(url)
  if (cached) {
    return cached as Promise<T>
  }

  const promise = (async () => {
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 15000)
    try {
      const response = await fetch(url, { signal: controller.signal })
      if (!response.ok) {
        throw new Error(`API 请求失败 ${response.status}: ${url}`)
      }
      const body = (await response.json()) as Envelope<T>
      if (body.code !== 0) {
        throw new Error(`API 业务错误 ${body.code}: ${body.message}`)
      }
      return body.data
    } finally {
      clearTimeout(timer)
    }
  })()

  cache.set(url, promise)
  return promise
}

export function apiUrl(path: string): string {
  return `${API_BASE}${path}`
}

export async function fetchAllArticles(pageSize = 100): Promise<ArticleSummary[]> {
  const first = await apiGet<Page<ArticleSummary>>(`/api/v1/articles?page=1&page_size=${pageSize}`)
  const pageCount = Math.ceil(first.total / pageSize)
  if (pageCount <= 1) {
    return first.items
  }

  const rest = await Promise.all(
    Array.from({ length: pageCount - 1 }, (_, index) =>
      apiGet<Page<ArticleSummary>>(`/api/v1/articles?page=${index + 2}&page_size=${pageSize}`),
    ),
  )
  return first.items.concat(...rest.map((page) => page.items))
}

export function pageCount(total: number, pageSize = 20): number {
  return Math.max(1, Math.ceil(total / pageSize))
}
