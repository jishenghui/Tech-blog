import type { ArchEdge, ArchNode, Award, DebugStep } from '../types/poc'

export const debugSteps: DebugStep[] = [
  {
    id: 'reproduce',
    title: '01 · 复现 500',
    description:
      '先用最小请求复现发布接口的 500，确认不是偶发网络问题。响应信封里的 50001 指向事务提交失败，而不是参数校验。',
    imageUrl: '/poc/debug-reproduce.svg',
    code: `$ curl -s -X POST http://127.0.0.1:8080/api/v1/admin/articles/42/publish
HTTP/1.1 500 Internal Server Error

{"code":50001,"message":"发布文章失败"}`,
    codeLang: 'bash',
    articleUrl: '/posts/sqlite-wal-lock/',
    tag: { label: 'SQLite', href: '/tags/sqlite/' },
  },
  {
    id: 'trace',
    title: '02 · 日志与锁',
    description:
      '打开 GORM 的慢查询与错误日志后，发现同一事务里既更新文章、又扫描文章表重算资源引用，写锁被自己长时间占用。示例视频为占位素材。',
    videoUrl: '/poc/debug-demo.mp4',
    videoPoster: '/poc/video-poster.svg',
    code: `// service/article.go
err := s.db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Model(&article).Updates(updates).Error; err != nil {
        return err
    }
    return syncAssetRefs(tx, oldContent, content)
})
// log: database is locked (5) (SQLITE_BUSY)`,
    codeLang: 'go',
    articleUrl: '/posts/sqlite-busy-timeout/',
    tag: { label: 'SQLite', href: '/tags/sqlite/' },
  },
  {
    id: 'fix',
    title: '03 · 修复与验证',
    description:
      '把资源引用重算放到文章落库之后，并让每篇文章只在事务内做必要写入；补上 busy_timeout 与回归测试后，发布接口恢复 200。',
    imageUrl: '/poc/debug-fix.svg',
    code: `// 同一事务内完成文章更新与引用计数
if err := tx.Model(&article).Updates(updates).Error; err != nil {
    return err
}
if err := syncAssetRefs(tx, oldContent, content); err != nil {
    return err
}
// 200 OK · 回归测试通过`,
    codeLang: 'go',
    articleUrl: '/posts/transaction-retry/',
    tag: { label: 'Go', href: '/tags/go/' },
  },
]

export const awards: Award[] = [
  {
    id: 'scholarship',
    title: '国家奖学金',
    date: '2024 · 教育部',
    description: '综合成绩与项目实践排名前列，获国家奖学金。',
    imageUrl: '/poc/award-1.svg',
  },
  {
    id: 'lanqiao',
    title: '蓝桥杯 · 省一等奖',
    date: '2024 · 蓝桥杯组委会',
    description: 'Java 软件开发大学 A 组省级一等奖。',
    imageUrl: '/poc/award-2.svg',
  },
  {
    id: 'graduate',
    title: '校级优秀毕业生',
    date: '2025 · 学校',
    description: '毕业设计与企业实践表现优秀，获评校级优秀毕业生。',
    imageUrl: '/poc/award-3.svg',
  },
]

export const architectureNodes: ArchNode[] = [
  {
    id: 'astro',
    label: 'Astro SSG',
    kind: 'frontend',
    x: 60,
    y: 50,
    description:
      '前台在 GitHub Actions 中构建，通过公开 API 拉取内容，输出全站静态 HTML；页面不依赖 Node 运行时。',
    href: '/tags/astro/',
  },
  {
    id: 'cdn',
    label: 'Qiniu CDN',
    kind: 'infra',
    x: 340,
    y: 50,
    description:
      '承载静态 HTML、文章图片与固定页素材。主域 CNAME 到 CDN，静态流量不经过 1Mbps 的 ECS。',
    href: '/tags/nginx/',
  },
  {
    id: 'actions',
    label: 'Actions',
    kind: 'infra',
    x: 620,
    y: 50,
    description:
      '发布写操作触发 repository_dispatch；构建成功后上传 Kodo 并刷新受影响 URL，失败时旧站继续服务。',
    href: '/tags/github-actions/',
  },
  {
    id: 'browser',
    label: 'Browser',
    kind: 'frontend',
    x: 60,
    y: 210,
    description:
      '访客浏览器。静态页从 CDN 加载；搜索、悬浮卡片、视频播放等交互由少量岛屿脚本接管。',
    href: '/tags/web/',
  },
  {
    id: 'api',
    label: 'Go API',
    kind: 'backend',
    x: 340,
    y: 210,
    description:
      'Gin + GORM 单二进制服务，常驻 2G ECS。公开读接口无鉴权，后台写接口走 JWT，只服务 API 与后台 SPA。',
    href: '/tags/go/',
  },
  {
    id: 'db',
    label: 'SQLite',
    kind: 'data',
    x: 620,
    y: 210,
    description:
      'WAL 模式的单文件数据库，配合纯 Go 驱动，不引入 CGO 与额外数据库进程，把内存留给系统与缓存。',
    href: '/tags/sqlite/',
  },
  {
    id: 'admin',
    label: 'Vue Admin',
    kind: 'admin',
    x: 60,
    y: 350,
    description:
      'Vue 3 + Vite 后台 SPA，负责文章、目录、资源和固定页 JSON 的可视化编辑；通过 admin 子域同源反代 API。',
    href: '/tags/vue/',
  },
]

export const architectureEdges: ArchEdge[] = [
  { from: 'astro', to: 'cdn' },
  { from: 'actions', to: 'cdn' },
  { from: 'browser', to: 'cdn' },
  { from: 'browser', to: 'api' },
  { from: 'admin', to: 'api' },
  { from: 'api', to: 'db' },
]
