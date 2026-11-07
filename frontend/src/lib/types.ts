export interface NavNode {
  id: number
  name: string
  slug: string
  kind: string
  page_key?: string | null
  children?: NavNode[]
}

export interface TagView {
  id: number
  name: string
  slug: string
  description?: string
  article_count: number
}

export interface DirectoryBrief {
  id: number
  name: string
  slug: string
  page_key?: string | null
}

export interface ArticleSummary {
  id: number
  directory_id: number | null
  title: string
  slug: string
  summary: string
  cover_url: string
  status?: string
  is_pinned: boolean
  view_count: number
  word_count: number
  reading_time: number
  source?: string
  published_at: string | null
  created_at: string
  updated_at: string
  tags: TagView[]
  directory?: DirectoryBrief
}

export interface ArticleDetail extends ArticleSummary {
  content_html: string
}

export interface Page<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface DirectoryDetail {
  id: number
  parent_id: number | null
  name: string
  slug: string
  kind: string
  page_key?: string | null
  description: string
  article_count: number
}

export interface DirectoryPage {
  directory: DirectoryDetail
  items: ArticleSummary[]
  total: number
  page: number
  page_size: number
}

export interface ProjectSummary {
  id: number
  title: string
  slug: string
  summary: string
  tech_stack: string[]
  cover_url: string
  sort_order: number
  status?: string
  updated_at: string
}

export interface DebugStep {
  id: string
  title: string
  description: string
  image_url: string
  video_url: string
  video_poster: string
  code: string
  code_lang: string
  article_slug: string
  tag_slug: string
}

export interface ArchitectureNode {
  id: string
  label: string
  kind: string
  x: number
  y: number
  description: string
  tag_slug: string
  search_query: string
}

export interface ArchitectureEdge {
  from: string
  to: string
}

export interface Architecture {
  nodes: ArchitectureNode[]
  edges: ArchitectureEdge[]
}

export interface ProjectDetail extends ProjectSummary {
  content_html: string
  repo_url: string
  demo_url: string
  started_at: string | null
  ended_at: string | null
  debug_steps: DebugStep[]
  architecture: Architecture
  created_at: string
}

export interface ProfileAward {
  id: string
  title: string
  image_url: string
  date: string
  description: string
}

export interface ProfileExperience {
  company: string
  role: string
  period: string
  description: string
}

export interface FeaturedProject {
  name: string
  summary: string
  url: string
  cover_url: string
}

export interface ProfileView {
  headline: string
  avatar_url: string
  resume_html: string
  awards: ProfileAward[]
  experiences: ProfileExperience[]
  featured_projects: FeaturedProject[]
  contacts: Record<string, string>
  updated_at: string
}

export interface TodoView {
  id: number
  content: string
  done: boolean
  sort_order: number
  due_date: string | null
  created_at: string
  updated_at: string
}
