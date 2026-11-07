# frontend

Astro 前台。当前处于 **M4 POC 阶段**：先验证固定页交互，正式页面、公开 API 接入与 404 兜底尚未实现。

## 运行

要求 Node 22.23.4（与 Astro 7 的 engines 要求一致）：

```bash
nvm use 22.23.4
npm install
npm run dev        # http://localhost:4321/poc/
npm run build      # 产物在 dist/
npm run preview    # 预览构建产物
npm run check      # Astro/TypeScript 检查
```

项目内 `.npmrc` 指向 `https://registry.npmmirror.com`；当前机器的全局清华 npm 镜像对 Astro 相关包返回 404，所以使用 npmmirror。若部署环境更适合官方源，可在 CI 中覆盖 registry。

## POC 内容

`/poc/` 验证三类固定页交互：

1. **项目调试流程**：截图/视频封面列表；悬停切换右侧说明与代码；视频封面悬停播放；点击跳转技术文章（当前为 mock 链接）。
2. **简历奖项**：悬停/聚焦奖项，右侧切换奖状大图与说明。
3. **项目架构图**：内联 SVG 节点；悬停/聚焦显示组件说明；点击跳转技术栈/文章。

实现要点：

- Astro 7 构建时生成静态 HTML；`@astrojs/vue` 岛屿用 `client:visible` 按需水合。
- 代码高亮用 Shiki 在构建时完成，浏览器不下载高亮库。
- 视频 `preload="none"`，悬停时才设置 `src` 并播放；`muted` 保证浏览器允许播放。
- 奖状/截图懒加载；架构图使用内联 SVG，不依赖图片热区。
- 悬停交互同时支持键盘聚焦；移动端点击切换，视频提供静帧兜底。

已用 Playwright + 本机 Edge 验证：调试步骤切换、视频播放（`readyState=4`）、奖项切换、架构节点切换均正常，无控制台错误。

## 文件结构

```text
src/
  components/
    DebugGallery.vue          # 调试流程岛屿
    AwardHover.vue            # 奖项岛屿
    ArchitectureDiagram.vue   # 架构图岛屿
  data/poc.ts                 # POC mock 数据
  types/poc.ts                # 数据类型
  pages/
    index.astro               # 临时入口
    poc.astro                 # 交互 POC
  styles/global.css
public/
  favicon.svg
  poc/                        # 截图、奖状、占位视频
```

`public/poc/debug-demo.mp4` 是 1.1MB 的 CC0 示例视频，仅用于演示悬停播放；正式实现会替换为项目自己的排查录屏，并放到七牛云。

## 与后台数据的关系

POC 数据目前来自 `src/data/poc.ts`。正式设计（见 `doc/数据库与API设计.md` 1.5.1）：

- `projects.debug_steps`：调试流程步骤 JSON；
- `projects.architecture`：架构图节点/边 JSON；
- `profile.awards`：奖状 JSON（已有字段）。

这些字段由 M5 后台可视化编辑，Astro 构建时通过公开 API 读取。视频文件建议放七牛云，数据库只存 URL。

## 性能备注

POC 构建产物中，页面 HTML 约 32KB；Vue runtime 约 68KB（未压缩），三个岛屿组件各约 4KB，共享同一份 runtime。正式实现时可评估：

- 奖项、架构图等简单交互改用原生 TS/CSS，进一步去掉 Vue runtime；
- 调试流程保留 Vue 岛屿，代码面板改为 Astro 静态渲染 + 轻量切换脚本，避免代码 HTML 在 island props 中重复。
