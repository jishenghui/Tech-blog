import rss from '@astrojs/rss'
import type { APIContext } from 'astro'
import { fetchAllArticles } from '../lib/api'

export async function GET(context: APIContext) {
  const site = context.site ?? new URL('https://www.example.com')
  const articles = await fetchAllArticles()
  const items = articles.slice(0, 30).map((article) => ({
    title: article.title,
    link: new URL(`/posts/${article.slug}/`, site).toString(),
    pubDate: new Date(article.published_at ?? article.created_at),
    description: article.summary,
  }))

  return rss({
    title: 'Tech Blog',
    description: '个人技术博客与项目展示',
    site,
    items,
    customData: '<language>zh-cn</language>',
  })
}
