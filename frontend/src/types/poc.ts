export interface DebugStep {
  id: string
  title: string
  description: string
  imageUrl?: string
  videoUrl?: string
  videoPoster?: string
  code: string
  codeLang: string
  articleUrl: string
  tag: {
    label: string
    href: string
  }
}

export interface DebugStepView extends DebugStep {
  codeHtml: string
}

export interface Award {
  id: string
  title: string
  date: string
  description: string
  imageUrl: string
}

export type ArchNodeKind = 'frontend' | 'backend' | 'data' | 'infra' | 'admin'

export interface ArchNode {
  id: string
  label: string
  kind: ArchNodeKind
  x: number
  y: number
  description: string
  href: string
}

export interface ArchEdge {
  from: string
  to: string
}
