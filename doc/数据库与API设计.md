# 数据库与 API 设计

本文档定义后端数据模型与接口，作为 `doc/个人博客需求文档.md` 的落地补充。原则：

1. **前后台共用一套 API**：公开读接口无需登录；写接口统一 `/api/v1/admin/` 前缀。
2. **适度冗余**：正文同时存 Markdown 与预渲染 HTML，计数字段、搜索文本、CDN 地址等直接落库，避免高频联表；冗余字段由写操作在同一事务内维护。
3. 数据库为 SQLite（WAL），所有时间字段为 UTC 的 `DATETIME`。

---

## 一、数据模型

SQLite 驱动：GORM + `github.com/glebarez/sqlite`（底层 modernc.org/sqlite，纯 Go，无需 CGO），开启 WAL 模式。

### 1.1 目录树 `directories`

一级、二级目录共用一张表，用 `parent_id` 表达层级（需求文档明确要求按 `parent_id` 维护树形结构）。

```sql
CREATE TABLE directories (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  parent_id     INTEGER REFERENCES directories(id), -- NULL 表示一级目录
  name          TEXT    NOT NULL,                   -- 展示名
  slug          TEXT    NOT NULL,                   -- URL 标识，全局唯一
  kind          TEXT    NOT NULL DEFAULT 'container', -- page | container
  page_key      TEXT,        -- 固定页标识：home|projects|tech|todos|other，普通目录为 NULL
  is_fixed      INTEGER NOT NULL DEFAULT 0, -- 固定目录不可删除/改名
  visible       INTEGER NOT NULL DEFAULT 1, -- 是否出现在导航
  sort_order    INTEGER NOT NULL DEFAULT 0, -- 同层排序
  description   TEXT,
  article_count INTEGER NOT NULL DEFAULT 0, -- 冗余：已发布文章数（仅二级目录有效）
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL,
  deleted_at    DATETIME
);
CREATE UNIQUE INDEX uk_directories_slug ON directories(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_directories_parent ON directories(parent_id, sort_order);
```

要点：

- 固定一级目录：`home`(首页)、`projects`(项目)、`tech`(技术栈)、`todos`(待办)、`other`(其他)。`home/projects/tech/todos` 为 `kind='page'`，内容来自各自的专用表，**不允许有子目录**；`other` 为 `kind='container'`，用于接收解散后的目录，允许有子目录。
- 普通一级目录 `kind='container'`，其下只能挂二级目录；文章只能挂二级目录。
- 软删除保证拖拽/解散过程中外键引用不悬空。

### 1.2 文章 `articles`

```sql
CREATE TABLE articles (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  directory_id INTEGER REFERENCES directories(id), -- 所属二级目录
  title        TEXT    NOT NULL,
  slug         TEXT    NOT NULL,
  summary      TEXT,                    -- 列表摘要，可为空时取正文前 N 字
  content_md   TEXT    NOT NULL DEFAULT '',
  content_html TEXT    NOT NULL DEFAULT '', -- 冗余：预渲染结果，前台直接使用
  cover_url    TEXT,
  status       TEXT    NOT NULL DEFAULT 'draft', -- draft | published
  is_pinned    INTEGER NOT NULL DEFAULT 0,
  view_count   INTEGER NOT NULL DEFAULT 0,
  word_count   INTEGER NOT NULL DEFAULT 0,   -- 冗余
  reading_time INTEGER NOT NULL DEFAULT 0,   -- 冗余，分钟
  source       TEXT    NOT NULL DEFAULT 'editor', -- editor | zip
  source_file  TEXT,                    -- zip 导入时的原始文件名
  search_text  TEXT,                    -- 冗余：标题+纯文本正文，供 LIKE 搜索
  created_at   DATETIME NOT NULL,
  updated_at   DATETIME NOT NULL,
  published_at DATETIME,
  deleted_at   DATETIME
);
CREATE UNIQUE INDEX uk_articles_slug ON articles(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_articles_dir ON articles(directory_id, status, published_at);
```

### 1.3 标签 `tags` / `article_tags`

```sql
CREATE TABLE tags (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  name          TEXT NOT NULL,
  slug          TEXT NOT NULL,
  description   TEXT,
  article_count INTEGER NOT NULL DEFAULT 0, -- 冗余：技术栈页直接读，不聚合
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL
);
CREATE UNIQUE INDEX uk_tags_name ON tags(name);
CREATE UNIQUE INDEX uk_tags_slug ON tags(slug);

CREATE TABLE article_tags (
  article_id INTEGER NOT NULL REFERENCES articles(id),
  tag_id     INTEGER NOT NULL REFERENCES tags(id),
  created_at DATETIME NOT NULL,
  PRIMARY KEY (article_id, tag_id)
);
CREATE INDEX idx_article_tags_tag ON article_tags(tag_id);
```

### 1.4 资源 `assets`

```sql
CREATE TABLE assets (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  article_id    INTEGER REFERENCES articles(id),    -- 可为空（先上传后引用）
  directory_id  INTEGER REFERENCES directories(id), -- 归属目录，便于批量清理
  provider      TEXT NOT NULL DEFAULT 'qiniu',
  bucket        TEXT NOT NULL,
  object_key    TEXT NOT NULL,   -- 七牛云 key，如 articles/2026/11/xxx.jpg
  url           TEXT NOT NULL,   -- 冗余：完整 CDN 地址，替换 Markdown 时直接取
  original_name TEXT,
  mime          TEXT,
  size          INTEGER NOT NULL DEFAULT 0,
  sha256        TEXT,            -- 内容哈希，用于去重
  ref_count     INTEGER NOT NULL DEFAULT 0, -- 冗余：被文章引用次数
  created_at    DATETIME NOT NULL,
  deleted_at    DATETIME
);
CREATE INDEX idx_assets_sha ON assets(sha256);
CREATE INDEX idx_assets_article ON assets(article_id);
```

### 1.5 项目 `projects`（固定页面"项目"）

```sql
CREATE TABLE projects (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  title        TEXT NOT NULL,
  slug         TEXT NOT NULL,
  summary      TEXT,
  content_md   TEXT NOT NULL DEFAULT '',
  content_html TEXT NOT NULL DEFAULT '',
  tech_stack   TEXT NOT NULL DEFAULT '[]', -- JSON 数组
  repo_url     TEXT,
  demo_url     TEXT,
  cover_url    TEXT,
  sort_order   INTEGER NOT NULL DEFAULT 0, -- 左侧列表顺序
  status       TEXT NOT NULL DEFAULT 'published',
  started_at   DATETIME,
  ended_at     DATETIME,
  created_at   DATETIME NOT NULL,
  updated_at   DATETIME NOT NULL,
  deleted_at   DATETIME
);
CREATE UNIQUE INDEX uk_projects_slug ON projects(slug) WHERE deleted_at IS NULL;
```

### 1.6 首页资料 `profile`（单行）

首页内容结构差异大（简历、获奖、经历），用 JSON 列保持扩展性：

```sql
CREATE TABLE profile (
  id                INTEGER PRIMARY KEY CHECK (id = 1),
  headline          TEXT,          -- 一句话简介
  avatar_url        TEXT,
  resume_md         TEXT NOT NULL DEFAULT '',
  resume_html       TEXT NOT NULL DEFAULT '', -- 冗余
  awards            TEXT NOT NULL DEFAULT '[]', -- JSON: [{title, image_url, date, description}]
  experiences       TEXT NOT NULL DEFAULT '[]', -- JSON: [{company, role, period, description}]
  featured_projects TEXT NOT NULL DEFAULT '[]', -- JSON: [{name, summary, url, cover_url}]
  contacts          TEXT NOT NULL DEFAULT '{}', -- JSON
  updated_at        DATETIME NOT NULL
);
```

### 1.7 待办 `todos`

```sql
CREATE TABLE todos (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  content    TEXT NOT NULL,
  done       INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  due_date   DATETIME,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

### 1.8 导入任务 `import_jobs`

压缩包导入是异步耗时操作，记录全过程便于排查：

```sql
CREATE TABLE import_jobs (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  directory_id      INTEGER NOT NULL REFERENCES directories(id),
  original_filename TEXT NOT NULL,
  file_size         INTEGER NOT NULL DEFAULT 0,
  sha256            TEXT,
  status            TEXT NOT NULL DEFAULT 'pending', -- pending|processing|success|failed
  extracted_root    TEXT,        -- 冗余：递归下探后实际使用的根路径
  md_count          INTEGER NOT NULL DEFAULT 0,
  image_count       INTEGER NOT NULL DEFAULT 0,
  replaced_count    INTEGER NOT NULL DEFAULT 0, -- 成功替换路径的图片数
  error_message     TEXT,
  meta              TEXT NOT NULL DEFAULT '{}', -- JSON：解压树、失败文件等
  created_at        DATETIME NOT NULL,
  finished_at       DATETIME
);
```

### 1.9 其他

```sql
CREATE TABLE users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  nickname      TEXT,
  avatar_url    TEXT,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME NOT NULL,
  last_login_at DATETIME
);

CREATE TABLE settings (          -- 站点名、备案号、CDN 域名、首页开关等
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at DATETIME NOT NULL
);
```

> 单管理员场景不建 session 表：JWT 无状态，access token 2h + refresh token 7d（httpOnly Cookie）；如需"登出所有设备"，在 `users` 增加 `token_version` 字段即可。

---

## 二、冗余字段的一致性规则

| 冗余字段 | 维护时机 |
| --- | --- |
| `articles.content_html` | 保存/发布时由 Markdown 渲染器生成 |
| `articles.word_count` / `reading_time` | 同一次渲染计算 |
| `articles.search_text` | 保存时从标题与纯文本正文生成 |
| `directories.article_count` | 文章发布/取消发布/移动/删除时，在**同一事务**内对源目录与目标目录增减 |
| `tags.article_count` | 文章标签变更、发布状态变更时同事务维护 |
| `assets.ref_count` | 文章保存时解析正文中的 CDN 地址统计引用 |
| `profile.resume_html` | 保存简历时渲染 |

提供 `POST /api/v1/admin/maintenance/recalc`（或一条 CLI 命令）全量重算以上字段，用于数据修复。

---

## 三、API 设计

### 3.1 通用约定

- 前缀 `/api/v1`。
- 写接口全部位于 `/api/v1/admin/...`，由 JWT 中间件保护；公开接口只读。
- 响应信封：

```json
{ "code": 0, "message": "ok", "data": {} }
```

- 分页参数 `page`（从 1 开始）、`page_size`（默认 20，最大 100），返回：

```json
{ "code": 0, "message": "ok", "data": { "items": [], "total": 0, "page": 1, "page_size": 20 } }
```

- 鉴权：`Authorization: Bearer <access_token>`；登录失败返回 401，业务错误返回 4xx + 非 0 `code`。
- refresh token 放 httpOnly Cookie，不设置 `Domain`（仅 `admin` 子域同源使用），`SameSite=Lax`；access token 放内存，不写 localStorage。

### 3.2 认证

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/auth/login` | 用户名密码登录，返回 access/refresh token |
| POST | `/api/v1/auth/refresh` | 刷新 access token |
| POST | `/api/v1/auth/logout` | 清除 refresh Cookie |
| GET | `/api/v1/auth/me` | 当前管理员信息 |
| PUT | `/api/v1/auth/password` | 修改密码 |

### 3.3 公开读接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/nav` | 导航树：一级目录及其二级目录（仅 `visible`，按 `sort_order`） |
| GET | `/api/v1/directories/{slug}` | 目录信息 + 其下已发布文章列表（支持 `tag`、分页） |
| GET | `/api/v1/articles` | 文章列表，筛选 `directory_id`、`tag`、`q`，仅 `published` |
| GET | `/api/v1/articles/{slug}` | 文章详情（返回 `content_html`） |
| GET | `/api/v1/tags` | 全部 tag 及 `article_count`（技术栈页） |
| GET | `/api/v1/tags/{slug}/articles` | 某 tag 下文章列表 |
| GET | `/api/v1/projects` | 项目列表（左侧切换） |
| GET | `/api/v1/projects/{slug}` | 项目详情 |
| GET | `/api/v1/profile` | 首页资料 |
| GET | `/api/v1/todos` | 公开待办 |
| GET | `/api/v1/search?q=` | 基于 `search_text` 的搜索 |

### 3.4 后台：目录

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/directories` | 完整树（含草稿计数、隐藏目录） |
| POST | `/api/v1/admin/directories` | 新建，`{parent_id,name,slug,sort_order}` |
| PUT | `/api/v1/admin/directories/{id}` | 改名/排序/可见性；固定目录禁止改 slug |
| POST | `/api/v1/admin/directories/{id}/dissolve` | **解散**，按需求文档语义迁移子项（见 3.8） |
| POST | `/api/v1/admin/directories/{id}/move` | 拖拽：`{parent_id, sort_order}` |
| POST | `/api/v1/admin/directories/reorder` | 批量树形排序：`{items:[{id,parent_id,sort_order}]}` |

### 3.5 后台：文章

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/articles` | 列表，支持 `directory_id`、`status`、`q` |
| GET | `/api/v1/admin/articles/{id}` | 详情（含 `content_md`、标签） |
| POST | `/api/v1/admin/articles` | 新建草稿 |
| PUT | `/api/v1/admin/articles/{id}` | 保存（服务端重新渲染 HTML、重算冗余字段） |
| DELETE | `/api/v1/admin/articles/{id}` | 软删除 |
| POST | `/api/v1/admin/articles/{id}/publish` | 发布 |
| POST | `/api/v1/admin/articles/{id}/unpublish` | 取消发布 |
| POST | `/api/v1/admin/articles/{id}/move` | 移入其他二级目录 `{directory_id}` |
| PUT | `/api/v1/admin/articles/{id}/tags` | 覆盖式设置标签 `{tag_ids}` |

### 3.6 后台：标签、资源、导入

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET/POST/PUT/DELETE | `/api/v1/admin/tags[/{id}]` | 标签管理 |
| POST | `/api/v1/admin/assets` | 图片上传（multipart），返回 CDN URL；按 `sha256` 去重 |
| DELETE | `/api/v1/admin/assets/{id}` | 删除资源（`ref_count>0` 时拒绝或仅解除引用） |
| POST | `/api/v1/admin/imports/zip` | 上传压缩包，参数 `file`、`directory_id`；同步完成或返回 job_id |
| GET | `/api/v1/admin/imports` | 导入历史 |
| GET | `/api/v1/admin/imports/{id}` | 单个导入任务状态与统计 |

### 3.7 后台：项目、首页、待办、维护

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET/POST/PUT/DELETE | `/api/v1/admin/projects[/{id}]` | 项目管理 |
| POST | `/api/v1/admin/projects/reorder` | 左侧列表排序 |
| GET/PUT | `/api/v1/admin/profile` | 首页资料读写 |
| GET/POST/PUT/DELETE | `/api/v1/admin/todos[/{id}]` | 待办管理 |
| POST | `/api/v1/admin/todos/reorder` | 排序 |
| POST | `/api/v1/admin/maintenance/recalc` | 全量重算冗余字段 |

### 3.8 解散与移动的语义

**解散一级目录 `{id}`**：
1. 校验不是固定目录；找到（或创建）`page_key='other'` 的一级目录。
2. 将 `{id}` 下所有二级目录的 `parent_id` 改为"其他"的 id，`sort_order` 追加到末尾。
3. 文章不动（仍挂在原子目录下）。
4. 软删除 `{id}`。

**解散二级目录 `{id}`**：
1. 找到（或创建）"其他"一级目录下名为"其他"的二级目录（`kind='container'`）。
2. 将该目录下所有文章 `directory_id` 改为"其他/其他"的 id。
3. 软删除 `{id}`，并同事务更新两个目录的 `article_count`。

**拖拽移动**：
- 二级目录跨一级目录：改 `parent_id`，一篇文章都不动。
- 文章跨二级目录：改 `articles.directory_id`，同事务增减两边 `article_count`。
- 批量 `reorder` 必须整批在单事务内完成，避免出现中间态循环 parent。

### 3.9 压缩包导入流程

1. 保存上传文件，计算 `sha256`，创建 `import_jobs`（`status=processing`）。
2. 解压到临时目录，**防 Zip Slip**（拒绝 `..` 与绝对路径条目）。
3. 递归下探：当根层只有一个条目且是目录时进入该目录，直到根层结构符合要求。
4. 校验：根层只允许 `.md` 文件和一个可选的 `images/` 目录；出现其他文件/目录即失败（`error_message` 记录实际结构）。
5. 逐张上传 `images/` 到七牛云，写入 `assets`（相同 `sha256` 复用已有 URL）。
6. 正则替换 md 中的 `!\[...\]\(\.?/?images/(...)\)` 为 CDN 地址；替换不了的图片计入失败清单但不中断。
7. 每个 md 创建一篇文章：标题取首个一级标题，缺省用文件名；`source='zip'`；挂到目标二级目录；默认草稿或按参数发布。
8. 更新 `import_jobs` 统计与 `finished_at`，清理临时目录。

### 3.10 前台渲染方案（已定：方案 A）

采用**全站 SSG + 发布重建**：

- 公开内容全部在构建时生成静态页：首页、项目、技术栈、目录/文章列表、**文章详情**、待办。Astro 构建脚本通过公开 API 拉取已发布内容。
- 岛屿（客户端 JS）只用于搜索、阅读数等非关键动态数据；导航下拉用 CSS 实现。
- 后台发布/取消发布/移动/解散/改 tag/改项目等写操作完成后，后端以防抖（30s）方式调用 GitHub `repository_dispatch` 触发 Actions 重建。构建失败时旧站保持不变。
- 构建完成并上传后，按受影响 URL 清单刷新 CDN（见第五节）。
- 兜底：新文章在重建完成前被访问时，静态站返回 404 页；404 页按 URL 中的 slug 调用公开 API 临时渲染正文。该页 `noindex`，不写 CDN、不承担 SEO。

---

## 四、索引与容量说明

- 单用户博客数据量小（预计文章 < 1000、资源 < 10000），上述索引足够；避免过度索引。
- 纯 Go SQLite 驱动在写密集场景慢于 CGO 版；本项目写操作集中在后台，读请求轻量，预计够用。若压测发现瓶颈，再评估 `mattn/go-sqlite3`（需要 CGO 工具链）。
- `search_text` 走 `LIKE '%q%'`；数据量增长后可启用 SQLite FTS5，接口不变。
- 七牛云免费额度为 10GB 存储 + 10GB CDN 回源，`assets.size` 与 `provider` 字段用于统计和后续迁移。

---

## 五、部署与 CDN 缓存

### 5.1 托管拓扑

**默认：Kodo 直传。** Astro 构建产物由 GitHub Actions 直接上传七牛 Kodo；主域名 CNAME 到七牛 CDN、源站为 Kodo；API 使用 `api.` 子域直连 ECS。静态流量完全不经过 ECS 的 1Mbps。

备选：ECS nginx 托管 `releases/<timestamp>/` 与 `current` symlink，七牛 CDN 回源 ECS。整站切换原子，但 CDN 冷缓存/回源会占用 1Mbps，不推荐作为默认。

Astro 使用 `build.format: 'directory'` 与统一 trailing slash，保证 URL 与 Kodo 对象键（`/xxx/index.html`）一一对应；Kodo 开启静态网站托管，配置首页文档与错误文档。后台 SPA 建议使用独立子域（如 `admin.example.com`），避免与主域的静态托管冲突。

### 5.2 缓存策略

| 对象 | Cache-Control | 是否需刷新 |
| --- | --- | --- |
| `/_astro/*`（文件名含内容哈希） | `public, max-age=31536000, immutable` | 否 |
| HTML（文章/列表/tag/首页） | `public, max-age=300` | 是（仅本发布受影响的页） |
| `sitemap.xml` / `rss.xml` | `public, max-age=600` | 是 |
| 文章图片（资源域名） | 长缓存 | 否 |

HTML 必须被 CDN 缓存，否则每次访问都会回源 Kodo，消耗 10GB/月回源额度。短 TTL 的作用：刷新负责"立即生效"，TTL 负责"最终一致"；刷新配额耗尽或刷新失败时，页面在 5 分钟内自愈。

### 5.3 发布流程（顺序固定）

1. Actions 构建；校验 dist（关键 HTML/资源存在、页面数量在预期范围）。
2. 上传新的 hashed 静态资源（文件名含哈希，不与旧版本冲突）。
3. 全部资源上传成功后，再上传 HTML、`sitemap.xml`、`rss.xml`（覆盖同名对象）。
4. 全部上传完成后，调用七牛刷新 API，按受影响 URL 清单批量刷新；失败重试，仍失败则依赖短 TTL。
5. 记录 release 元数据（commit、受影响 URL、刷新配额余量）。
6. 不删除旧 hashed 资源，至少保留 7 天；清理任务按当前 manifest 删除更早的资源。

构建或上传任一步失败：**不刷新、不切换**，旧站继续服务。回滚 = 重新部署上一个 commit 的构建产物（保留 Actions artifact）。

Kodo 直传没有原子目录切换，靠"资源先传、HTML 后传、旧资源不删"保证任一时刻单页自洽；跨页 nav 不一致窗口只有几秒。若采用 ECS + nginx 备选，则用 release 目录 + symlink 原子切换，禁止对 live 目录执行 `rsync --delete`。

### 5.4 受影响 URL 清单

由后端在写操作成功后计算，去重后随 `repository_dispatch` 的 `client_payload` 传给 Actions；防抖窗口内的多次发布合并处理。

| 变更 | 需刷新 |
| --- | --- |
| 文章新增/修改/删除 | 文章页、所属列表页及其分页、首页（若展示最新）、该文章涉及的 tag 页、tag 索引、`sitemap.xml`、`rss.xml` |
| 文章移动 | 源/目标列表页及分页、文章页、`sitemap.xml` |
| 目录新增/改名/解散/拖拽 | **所有页面**（nav 内嵌在每页 HTML 中），按每次 60 个 URL 分批 |
| 项目新增/修改/排序 | 项目页、首页 |
| 首页资料修改 | 首页 |
| 待办修改 | 待办页 |

目录结构变更会导致全站 HTML 变化。若站点页面数超过 500，一天内无法全部刷新，策略是优先刷新首页/列表/最近文章，其余依赖 5 分钟 TTL。这也是不要轻易调整目录结构的运维原因。

### 5.5 刷新配额与调用

- 文件刷新默认 **500 URL/天**，每日 0 点重置；控制台单次最多 20 个 URL，OpenAPI `POST /v2/tune/refresh` 单次最多 60 个 URL，qshell `cdnrefresh` 默认每批 50 个。超限错误码 `400034`。
- 目录刷新默认约 **10–11 次/天**（官方文档与 FAQ 数字不一致，部分账号需开通），控制台单次 5 个目录，API 单次 10 个；不作为常规手段。
- 预取默认 100 个/天，可用于发布后预热首页/列表。
- 响应返回 `urlQuotaDay` / `urlSurplusDay`（目录同理），后台记录余量；余量不足时按"文章页 > 列表/首页 > tag > sitemap/rss"降级刷新。
- 单次发布通常 20–40 个 URL；个人博客正常频率下 500/天足够。

### 5.6 404 兜底页

- Kodo 静态托管配置错误文档 `/404.html`（ECS 备选方案用 `error_page 404 /404.html`）。
- 404 页读取 `location.pathname`，识别文章路径后调用 `GET https://api.example.com/api/v1/articles/{slug}`；成功则渲染正文，失败显示"文章不存在或尚未发布"。
- 页面带 `<meta name="robots" content="noindex">`；不写 CDN、不做 SEO。
- 公开 API 仅允许站点域名（`https://www.example.com`）的 CORS，供 404 兜底页调用；后台 SPA 由 `admin.` 子域 Nginx 同时反代 `/api/`，与 API 同源，不走 CORS。
- 新文章 URL 在兜底期间可能被 CDN 缓存为 404，发布时把新文章 URL 一并加入刷新清单。

> 配额数据来自七牛官方文档，控制台实际额度可能不同；上线前以控制台显示为准。
