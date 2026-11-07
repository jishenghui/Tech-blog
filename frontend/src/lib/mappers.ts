import { codeToHtml } from 'shiki'
import type {
  ArchitectureEdge as ApiArchEdge,
  ArchitectureNode as ApiArchNode,
  DebugStep,
  ProfileAward,
} from './types'
import type { ArchEdge, ArchNode, ArchNodeKind, Award, DebugStepView } from '../types/poc'

export async function toDebugStepViews(steps: DebugStep[] | undefined): Promise<DebugStepView[]> {
  if (!steps?.length) {
    return []
  }

  return Promise.all(
    steps.map(async (step) => {
      const lang = step.code_lang || 'text'
      let codeHtml = ''
      if (step.code) {
        try {
          codeHtml = await codeToHtml(step.code, { lang, theme: 'github-dark' })
        } catch {
          codeHtml = await codeToHtml(step.code, { lang: 'text', theme: 'github-dark' })
        }
      }

      const tagHref = step.tag_slug ? `/tags/${step.tag_slug}/` : '/tags/'
      return {
        id: step.id,
        title: step.title,
        description: step.description,
        imageUrl: step.image_url || undefined,
        videoUrl: step.video_url || undefined,
        videoPoster: step.video_poster || undefined,
        code: step.code,
        codeLang: lang,
        articleUrl: step.article_slug ? `/posts/${step.article_slug}/` : tagHref,
        tag: {
          label: step.tag_slug || '技术栈',
          href: tagHref,
        },
        codeHtml,
      }
    }),
  )
}

export function toArchNodes(nodes: ApiArchNode[]): ArchNode[] {
  return nodes.map((node) => ({
    id: node.id,
    label: node.label,
    kind: normalizeArchKind(node.kind),
    x: node.x,
    y: node.y,
    description: node.description,
    href: node.tag_slug
      ? `/tags/${node.tag_slug}/`
      : node.search_query
        ? `/search/?q=${encodeURIComponent(node.search_query)}`
        : '/tags/',
  }))
}

export function toArchEdges(edges: ApiArchEdge[]): ArchEdge[] {
  return edges.map((edge) => ({ from: edge.from, to: edge.to }))
}

export function toAwardViews(awards: ProfileAward[]): Award[] {
  return awards.map((award, index) => ({
    id: award.id || `award-${index + 1}`,
    title: award.title,
    date: award.date,
    description: award.description,
    imageUrl: award.image_url,
  }))
}

function normalizeArchKind(kind: string): ArchNodeKind {
  switch (kind) {
    case 'frontend':
    case 'backend':
    case 'data':
    case 'infra':
    case 'admin':
      return kind
    default:
      return 'infra'
  }
}
