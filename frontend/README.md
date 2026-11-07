# frontend

Astro 前台。M4 已完成：全站 SSG、构建时通过公开 API 取数、文章渲染、搜索、404 兜底、sitemap 与 RSS。

## 运行

要求 Node 22.23.4（与 Astro 7 的 engines 要求一致）：

```bash
nvm use 22.23.4
npm install
npm run dev        # http://localhost:4321/
npm run build      # 需要后端 API 可用，产物在 dist/
npm run preview    # 预览构建产物
npm run check      # Astro/TypeScript 检查
```

本地 `.env`：

```dotenv
PUBLIC_API_BASE_URL=http://localhost:8080
SITE_URL=http://localhost:4321
```

模板见 `deploy/env/frontend.env.example`。`npm run build` 会在构建时请求 `PUBLIC_API_BASE_URL`，因此构建前必须先启动 `backend/`。

项目内 `.npmrc` 指向 `https://registry.npmmirror.com`；当前机器的全局清华 npm 镜像对 Astro 相关包返回 404，所以使用 npmmirror。若部署环境更适合官方源，可在 CI 中覆盖 registry。

## 页面

| 页面 | 路径 |
| --- | --- |
| 首页 / 简历 | `/` |
| 项目列表 | `/projects/` |
| 项目详情 | `/projects/{slug}/` |
| 技术栈 | `/tags/` |
| 标签文章 | `/tags/{slug}/`、`/tags/{slug}/page/{n}/` |
| 一级目录 | `/{category}/` |
| 二级目录 | `/{category}/{subcategory}/`、`/page/{n}/` |
| 文章详情 | `/posts/{slug}/` |
| 待办 | `/todos/` |
| 搜索 | `/search/?q=`（客户端调公开搜索 API） |
| 404 | `/404.html`（按文章 slug 调公开 API 兜底） |
| sitemap / RSS | `/sitemap.xml`、`/rss.xml` |
| POC | `/poc/`（保留的交互原型，不进入 sitemap） |

## 实现要点

- 全站 SSG：`getStaticPaths` 在构建时拉取目录、标签、项目、文章和固定页 JSON；产物为纯静态 HTML。
- 文章页：
  - 后端 `content_html` 直接输出；Chroma CSS 在 `src/styles/chroma.css`。
  - 数学公式使用 KaTeX auto-render，Mermaid 使用动态 `import('mermaid')`；只有页面包含对应标记时才加载。
  - `article-prose` 隐藏正文重复的首个 h1（页面标题已单独展示）。
- 固定页：
  - 项目页集成 `DebugGallery.vue`（构建时 Shiki 高亮代码、视频悬停播放）与 `ArchitectureDiagram.vue`（内联 SVG）。
  - 首页集成 `AwardHover.vue`（奖项悬浮奖状）。
- 搜索：静态页面 + 小脚本调用 `/api/v1/search`；这是少数客户端取数场景之一。
- 404：静态 404 页读取 URL 中的文章 slug，调 `/api/v1/articles/{slug}` 临时渲染；`noindex`，不承担 SEO。
- sitemap 与 RSS 使用构建时数据生成；sitemap 排除 `/poc/`、搜索页与 404。
- 交互岛屿使用 `client:visible`，共享 Vue runtime；普通文章/列表页不加载 Vue。

## 主题

- 默认 `auto`：按站点坐标计算日出/日落，白天浅色、夜间深色；内联脚本在首屏渲染前设置 `data-theme`，避免闪烁。
- 页头按钮循环 `自动 → 浅色 → 深色 → 自动`，手动选择写入 `localStorage`（`blog-theme-mode`）；清除后恢复自动。
- 坐标来自 `PUBLIC_SITE_LAT` / `PUBLIC_SITE_LNG`，默认上海（31.2304 / 121.4737）。换城市时更新 `frontend/.env` 与 `deploy/env/frontend.env.example`。
- 全部在浏览器完成，不调用 API、不增加服务器开销；坐标非法时回退 `prefers-color-scheme`。
- 代码高亮：`chroma.css`（深色）+ `chroma-light.css`（浅色，作用域为 `[data-theme='light']`）；Mermaid 按页面初始主题选择 `default` / `dark`。
- 切换主题后，已经渲染的 Mermaid 图不会自动重绘；下次加载会按当前主题渲染。`/poc/` 原型页固定深色。

## 性能备注

- 页面主体为静态 HTML；文章页只有搜索/404 之外的必要脚本，KaTeX/Mermaid 按需加载。
- Mermaid 和 KaTeX 的动态 chunk 较大，Vite 会给出 >500KB 提示；这是按需加载的预期结果，不影响首屏。
- `public/poc/debug-demo.mp4` 是 1.1MB 的 CC0 示例视频，仅用于 `/poc/` 演示；正式站点不需要时可删除。
