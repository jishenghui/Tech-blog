# Tech-blog

个人技术博客与项目展示平台。**M1–M5 已完成；下一步 M6（发布流水线与 ECS 部署）。**

## 文档

| 文档 | 内容 |
| --- | --- |
| [需求文档](doc/个人博客需求文档.md) | PRD、技术选型、实施步骤、部署与 CDN 配置 |
| [数据库与 API 设计](doc/数据库与API设计.md) | 表结构、接口清单、冗余字段规则、发布流程 |
| [开发计划](doc/开发计划.md) | 里程碑、环境要求、配置项、开放问题、下次会话起点 |
| [CLAUDE.md](CLAUDE.md) | 给 Claude Code 的项目约定与架构决策 |

## 架构概览

```
浏览器
  │
  ├── www.example.com ──► 七牛 CDN ──► Kodo(blog-static)   Astro SSG 全站静态页
  ├── cdn.example.com ──► 七牛 CDN ──► Kodo(blog-assets)   文章图片
  ├── api.example.com ──► ECS Nginx ──► Go API (Gin + SQLite)
  └── admin.example.com ► ECS Nginx ──► Vue 3 后台 SPA（/api/ 同源反代）
```

- **前台**：Astro 全站 SSG（含文章详情），构建时通过公开 API 取数。
- **发布**：后台写操作 → 后端防抖触发 GitHub Actions `repository_dispatch` → Actions 重建 → qshell 上传 Kodo → 刷新受影响 CDN URL。构建失败保留旧站。
- **兜底**：新文章在重建完成前被访问时，`404.html` 按 slug 调公开 API 临时渲染（`noindex`，不写 CDN）。
- **后台**：Vue 3 + Vite + Pinia + Vditor；与前台共用 `/api/v1`，写接口统一 `/api/v1/admin` + JWT。
- **数据库**：SQLite（WAL）+ GORM + `github.com/glebarez/sqlite` 纯 Go 驱动，无需 CGO。

## 目录结构

```
backend/    Go + Gin + GORM(SQLite)，鉴权/目录/文章/标签/资源/导入/公开 API，单二进制 blog-api
frontend/   Astro 前台（SSG，M4 已完成）
admin/      Vue 3 后台 SPA（M5 已完成）
deploy/     nginx.conf、systemd 等部署模板
doc/        需求、设计、开发计划
```

## 环境要求

- Go：当前稳定版（≥1.23）
- Node.js：22 LTS + npm
- Docker + Docker Compose（本地试运行）
- 无需 GCC（纯 Go SQLite）；qshell 仅在 CI 中使用
- GitHub 认证（PAT 或 Git Credential Manager）

## 快速开始（后端）

```bash
cd backend
cp ../deploy/env/backend.env.example .env    # 首次；填写 JWT_SECRET
go run . -hash-password 你的密码             # 把输出填入 .env 的 ADMIN_PASSWORD_HASH
go run .
```

首次启动会自动建表、创建唯一管理员与固定一级目录，并在 `./data/blog.db` 创建 SQLite（WAL）数据库。接口与后续里程碑见 [开发计划](doc/开发计划.md)；本地环境审计见开发计划第 4 节。

## 快速开始（前台）

先启动后端（见上），确保 `http://localhost:8080` 可用，然后：

```bash
cd frontend
nvm use 22.23.4
npm install
# 首次：cp ../deploy/env/frontend.env.example .env
npm run dev        # http://localhost:4321/
npm run build      # 构建时通过 API 取数，产物在 dist/
```

前台已实现首页/简历、项目、技术栈、目录与文章、待办、搜索、404 兜底、sitemap 与 RSS，并支持浅色/深色与按日出日落自动切换的主题。三类固定页交互已集成到首页与项目页；交互原型仍可在 `/poc/` 查看。详细说明见 [frontend/README.md](frontend/README.md)。

## 快速开始（后台）

先启动后端，然后：

```bash
cd admin
nvm use 22.23.4
npm install
npm run dev        # http://localhost:5173
```

后台通过 Vite 代理 `/api` 到 `http://127.0.0.1:8080`，支持登录/刷新、文章与 Vditor 编辑、目录树拖拽/解散、项目固定页 JSON、首页资料、待办、ZIP 导入和修改密码。详细说明见 [admin/README.md](admin/README.md)。

## 部署

详见 [需求文档 · 阶段三至阶段六](doc/个人博客需求文档.md) 与 [数据库与 API 设计 · 第五节](doc/数据库与API设计.md)。
