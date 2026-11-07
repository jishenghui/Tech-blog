# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目现状

**M1/M2 已完成（2026-11-07）。** `backend/` 已包含 Gin + GORM(SQLite)、JWT 鉴权、目录/文章/标签 CRUD、Markdown 渲染与公开读接口；`frontend/`、`admin/` 尚未创建。开始任何工作前先读：

- `README.md` — 项目概览与架构图
- `doc/个人博客需求文档.md` — PRD、技术选型、实施步骤
- `doc/数据库与API设计.md` — 表结构、API、发布/CDN 细节
- `doc/开发计划.md` — 里程碑、环境、配置项、开放问题

Git：`main` 分支，remote `origin` = `https://github.com/jishenghui/Tech-blog.git`；本仓库提交身份为 `jishenghui <jishenghui@users.noreply.github.com>`。

## Git 提交规范

- **所有 commit message 必须使用中文描述**，标题与正文均是；类型前缀（`feat:`、`fix:`、`docs:`、`chore:` 等）可以保留英文，但冒号后的描述必须为中文。
- 示例：`docs: 更新迁移后开发机环境审计`、`feat: 搭建后端骨架与 JWT 鉴权`。
- 不要使用纯英文句子作为提交说明。
- **任务完成并验证通过后直接提交**，无需再次询问；默认提交到当前分支（`main`）。
- 提交前确认测试通过，且 `.env`、SQLite 数据库、编译产物等未进入版本库。

## 项目目标与目标架构

个人技术博客（单管理员），部署于 2 核 2G / 1Mbps 阿里云 ECS。选型的核心约束是**内存与带宽**，任何设计都要优先服从这一点。

计划中的结构：

```
backend/    Go + Gin + GORM(SQLite)，编译为单二进制 blog-api
frontend/   Astro 前台（SSG）
admin/      Vue 3 + Vite 后台 SPA
deploy/     nginx.conf、systemd 等部署模板
doc/        需求、设计、开发计划
```

**关键架构决策（不要轻易改变）：**
- 前台 Astro **全站 SSG（含文章详情）**，构建产物直传七牛 Kodo，主域 CNAME 到七牛 CDN；ECS 的 1Mbps 只服务 API、后台 SPA 和构建取数。若把前台资源留在 ECS，100 并发会打满带宽——这是整个方案的前提。
- 发布类写操作由后端防抖触发 GitHub Actions `repository_dispatch` 重建；构建成功才上传 Kodo 并刷新受影响 URL，失败保留旧站。
- 新文章在重建完成前由 Kodo `404.html` 按 slug 调公开 API 临时渲染，仅兜底：`noindex`、不写 CDN。
- 数据库用 SQLite（WAL）+ GORM + `github.com/glebarez/sqlite` 纯 Go 驱动，**不使用 CGO**；后端用 Go 而非 Node，都是为了 2G 内存。
- 前后台共用一套 API：公开读接口无鉴权；写接口统一 `/api/v1/admin/` + JWT。
- 部署：`www`/`cdn` 走七牛 CDN + Kodo；`api`/`admin` 子域指向 ECS，Nginx 仅服务这两个子域。

## 领域规则（来自需求文档，实现时必须遵守）

**目录模型**：固定一级目录为 首页、项目、技术栈、待办、其他。首页/项目/技术栈/待办 是 `kind='page'`，**没有二级目录**；`其他` 是容器目录（`kind='container'`），允许二级目录，用于承接解散后的目录。树形结构用 SQLite 的 `parent_id` 维护。

**解散语义**（容易搞错，按原文实现）：
- 解散一级目录：其下的二级目录整体归入一级目录"其他"，文件保留在各自原二级目录中。
- 解散二级目录：其下文件归入"其他"一级目录下的"其他"二级目录。该兜底目录 slug 固定为 `other-other`（slug 全局唯一），由解散操作自动创建，不能被解散。

**压缩包导入**：
- 解压后根层级只允许一个 `images/` 子目录加若干 `.md`；出现其他文件/目录即失败。
- 若根层级只有单一文件夹（重复压缩），需**递归下探**直到满足上述结构或触底报错（不能只剥一层）。
- `images/` 中图片上传七牛云；md 中的 `![](./images/xxx.jpg)` 需替换为 CDN 绝对地址。
- 必须防 Zip Slip（拒绝 `..` 与绝对路径条目）。

**技术栈页**：按文章 tag 聚合，列出全部 tag 及每个 tag 下的文章数量，可筛选。

**Markdown 渲染**：后端 goldmark + Chroma 生成 `content_html`；数学公式输出 MathJax/KaTeX 兼容的 `<span class="math ...">\(...\)</span>`，Mermaid 输出 `<pre class="mermaid">`，由前台按需加载 KaTeX/Mermaid 与 Chroma CSS 完成最终展示。`search_text` 与摘要不包含代码块。

## 构建与运行

后端（已可运行）：

```bash
cd backend
go run .                              # 本地开发，读取 backend/.env
go run . -hash-password 你的密码      # 生成 ADMIN_PASSWORD_HASH
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o blog-api .
```

首次运行前从 `deploy/env/backend.env.example` 复制出 `backend/.env`，填写 `JWT_SECRET` 与 `ADMIN_PASSWORD_HASH`。首次启动会自动建表、创建管理员与固定一级目录。

前台与后台（待 M4/M5）：

```bash
cd frontend && npm install && npm run build
cd admin && npm install && npm run build
```

本地试运行规划为 Docker Compose（`backend` / `frontend` / `nginx` 三个服务），仅用于联调；生产前台由 Kodo + CDN 承载。仓库中目前还没有 `Dockerfile`、`docker-compose.yml`，需要时按需求文档阶段二创建。

## 环境

本机审计（2026-11-07，迁移后的新开发机）：Go 1.27.3（winget 安装，`GOPROXY=https://goproxy.cn,direct`）、Node v24.13.0（nvm 管理，另装有 22.23.4）、npm 11.6.2、Docker 29.9.2、Compose v5.6.1、Git 2.55.0、winget 1.29.380。GCC、make、gh、qshell 未安装：纯 Go SQLite 不需要 GCC；make/qshell 本项目不需要（qshell 只在 CI 用）；gh 可选（`winget install --id GitHub.cli -e`）。本地开发建议 `nvm use 22.23.4` 对齐 CI 的 Node 22。详见 `doc/开发计划.md` 第 4 节。

## 安全与限流（部署层约定）

- Nginx `limit_req_zone` 按 IP 限制 10 r/s，超限返回 429；fail2ban 匹配 429 日志，封禁 1 分钟。
- 七牛云开启 Referer 白名单防盗链；免费额度有限（约 10GB 存储 + 10GB CDN 回源/下载流量，以控制台为准），HTML 必须被 CDN 缓存。
- 上线环境需创建 2G Swap 防 OOM。

## 已确认的设计决策

- 接口路径与数据库表结构由项目自行定义，详见 `doc/数据库与API设计.md`；实现时以该文档为准，不要另起一套。
- **前后台共用一套 API**：公开读接口无需鉴权；所有写接口统一挂 `/api/v1/admin` 前缀，由 JWT 中间件保护。
- 数据库表设计允许并鼓励冗余（预渲染 `content_html`、各类计数字段、`search_text`、资源 URL 等），目的是减少联表与后续迁移；代价是所有冗余字段必须在同一事务内随写操作更新。
- 前台渲染最终采用**方案 A（全站 SSG + 发布重建）**，不做正文客户端拉取；404 兜底是唯一例外。
- CDN 缓存、发布顺序、受影响 URL 清单与刷新配额见 `doc/数据库与API设计.md` 第五节，不要在实现中绕开。

## 下一步

M1/M2 已完成（`backend/`）。下一步按 `doc/开发计划.md` 的 **M3** 实现七牛云图片上传与 ZIP 导入；不要同时铺开前台/后台。任何架构变更先更新设计文档与 CLAUDE.md，再写代码。
