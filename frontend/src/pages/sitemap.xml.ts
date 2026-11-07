import type { APIRoute } from 'astro'
import { apiGet, fetchAllArticles, pageCount } from '../lib/api'
import type { DirectoryPage, NavNode, ProjectSummary, TagView } from '../lib/types'

export const GET: APIRoute = async ({ site }) => {
  const base = (site ?? new URL('https://www.example.com')).toString().replace(/\/$/, '')
  const urls = new Set<string>(['/', '/projects/', '/tags/', '/todos/'])

  const nav = await apiGet<NavNode[]>('/api/v1/nav')
  for (const top of nav.filter((node) => node.kind === 'container')) {
    urls.add(`/${top.slug}/`)
    for (const child of top.children ?? []) {
      urls.add(`/${top.slug}/${child.slug}/`)
      const first = await apiGet<DirectoryPage>(
        `/api/v1/directories/${child.slug}?page=1&page_size=1`,
      )
      for (let page = 2; page <= pageCount(first.total, 20); page += 1) {
        urls.add(`/${top.slug}/${child.slug}/page/${page}/`)
      }
    }
  }

  const projects = await apiGet<ProjectSummary[]>('/api/v1/projects')
  for (const project of projects) {
    urls.add(`/projects/${project.slug}/`)
  }

  const tags = await apiGet<TagView[]>('/api/v1/tags')
  for (const tag of tags) {
    urls.add(`/tags/${tag.slug}/`)
    for (let page = 2; page <= pageCount(tag.article_count, 20); page += 1) {
      urls.add(`/tags/${tag.slug}/page/${page}/`)
    }
  }

  const articles = await fetchAllArticles()
  for (const article of articles) {
    urls.add(`/posts/${article.slug}/`)
  }

  const body = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ...[...urls]
      .sort()
      .map((path) => `  <url><loc>${escapeXml(base + path)}</loc></url>`),
    '</urlset>',
  ].join('\n')

  return new Response(body, {
    headers: {
      'Content-Type': 'application/xml; charset=utf-8',
    },
  })
}

function escapeXml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;')
}
