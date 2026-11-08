# admin

Vue 3 + Vite + Pinia 后台管理 SPA。M5 已完成：登录/刷新、文章与 Vditor 编辑、目录树拖拽/解散、标签、项目固定页 JSON、首页资料、待办、ZIP 导入、修改密码。

## 运行

要求 Node 22.23.4；后端需在 `http://127.0.0.1:8080` 运行：

```bash
cd admin
nvm use 22.23.4
npm install
npm run dev        # http://localhost:5173
npm run check      # vue-tsc
npm run build      # 产物在 admin/dist
```

开发环境通过 Vite 代理 `/api` 到 `http://127.0.0.1:8080`，因此不存在跨域问题，refresh Cookie 也会正常写入。生产环境 `admin` 子域由 Nginx 同时托管 SPA 并反代 `/api/`，`VITE_API_BASE_URL` 保持为空。

## 鉴权

- access token 只放内存（Pinia 之外由 `lib/session.ts` 保存），刷新页面通过 refresh token 重新获取。
- refresh token 由后端写入 `httpOnly` Cookie，Path 为 `/api/v1/auth`，前端不可读。
- 路由守卫在进入受保护页面前调用 `/api/v1/auth/refresh`；失败跳转登录页。
- API 请求遇到 401 时自动尝试刷新一次并重放请求。
- 侧边栏「修改密码」调用 `PUT /api/v1/auth/password`（旧密码 + 新密码，至少 8 个字符）。

## Vditor

- 编辑器使用 `mode: 'sv'`，实时预览，支持图片上传；上传走后端 `/api/v1/admin/assets`（七牛，需配置 QINIU_*）。
- Vditor 运行时资源不从外网 CDN 加载：`predev`/`prebuild` 执行 `scripts/sync-vditor.mjs`，把 `node_modules/vditor/dist` 复制到 `public/vditor/dist`（约 24MB，已被 `.gitignore` 忽略），`cdn` 指向 `/vditor`。
- 视频文件上传暂未开放，固定页视频字段填七牛 URL。
- 不设独立资源列表页：资源上传内嵌于文章编辑器、项目/资料图片字段，与 API 设计仅提供上传/删除接口保持一致。

## 页面

| 页面 | 路径 |
| --- | --- |
| 登录 | `/login` |
| 概览 | `/` |
| 文章列表 / 新建 / 编辑 | `/articles`、`/articles/new`、`/articles/:id` |
| 目录 | `/directories` |
| 标签 | `/tags` |
| 项目列表 / 编辑 | `/projects`、`/projects/new`、`/projects/:id` |
| 首页资料 | `/profile` |
| 待办 | `/todos` |
| ZIP 导入 | `/imports` |

## 固定页 JSON 编辑

- 项目编辑器分「基本信息 / 正文 / 调试流程 / 架构图」四个标签。
- 调试步骤支持增删、上下移动、图片/封面 URL 上传、代码与关联文章/标签。
- 架构图节点支持坐标、类型、说明、标签/搜索词；连线用下拉选择起终点，并有 SVG 预览。
- 首页资料支持奖项、经历、推荐项目、联系方式和简历 Markdown；奖状/封面/头像均可上传。
- 后端在保存时校验 JSON 结构，前台 Astro 构建时直接消费。

## 目录拖拽

- 二级目录可拖到其他一级容器目录（固定页面目录不可作为目标）。
- 文章 chip 可拖到其他二级目录。
- 拖拽失败或目标不合法时后端返回错误，界面会提示。
- 排序使用上/下按钮调用批量 reorder 接口，避免拖拽排序的中间态。

## 构建与部署

- `npm run build` 先做 `vue-tsc --noEmit`，再执行 Vite 构建；`dist/` 交给 Nginx 托管。
- 生产不需要 Node 运行时；后台 SPA 只由 ECS 服务，前台静态站仍走七牛 Kodo/CDN。
- 项目内 `.npmrc` 使用 npmmirror；如 CI 使用官方源，可覆盖 registry。
