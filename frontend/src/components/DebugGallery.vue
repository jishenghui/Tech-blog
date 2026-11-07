<script setup lang="ts">
import { computed, ref } from 'vue'
import type { DebugStepView } from '../types/poc'

const props = defineProps<{
  steps: DebugStepView[]
}>()

const activeIndex = ref(0)
const active = computed(() => props.steps[activeIndex.value] ?? props.steps[0])

function select(index: number) {
  activeIndex.value = index
}

function playVideo(event: Event, step: DebugStepView) {
  const video = event.currentTarget as HTMLVideoElement
  if (step.videoUrl && !video.src) {
    video.src = step.videoUrl
  }
  void video.play().catch(() => {
    // 移动端或浏览器策略可能阻止自动播放，保留静帧即可。
  })
}

function pauseVideo(event: Event) {
  const video = event.currentTarget as HTMLVideoElement
  video.pause()
  video.currentTime = 0
}
</script>

<template>
  <div class="debug-gallery">
    <div class="debug-list" role="tablist" aria-label="项目调试步骤">
      <button
        v-for="(step, index) in steps"
        :key="step.id"
        type="button"
        class="debug-item"
        :class="{ active: index === activeIndex }"
        role="tab"
        :aria-selected="index === activeIndex"
        @mouseenter="select(index)"
        @focus="select(index)"
        @click="select(index)"
      >
        <span class="debug-thumb">
          <video
            v-if="step.videoUrl"
            :poster="step.videoPoster"
            muted
            loop
            playsinline
            preload="none"
            :aria-label="`${step.title} 示例视频`"
            @mouseenter="playVideo($event, step)"
            @mouseleave="pauseVideo"
          ></video>
          <img v-else :src="step.imageUrl" :alt="step.title" loading="lazy" />
          <span v-if="step.videoUrl" class="video-badge">HOVER · VIDEO</span>
        </span>
        <span class="debug-title">{{ step.title }}</span>
      </button>
    </div>

    <div v-if="active" class="debug-panel" role="tabpanel">
      <div class="panel-head">
        <div>
          <p class="eyebrow">DEBUG TRACE</p>
          <h3>{{ active.title }}</h3>
        </div>
        <a class="tag" :href="active.tag.href">{{ active.tag.label }}</a>
      </div>
      <p class="panel-desc">{{ active.description }}</p>

      <div class="code-shell">
        <div class="code-bar">
          <span></span><span></span><span></span>
          <em>{{ active.codeLang }}</em>
        </div>
        <div class="code-content" v-html="active.codeHtml"></div>
      </div>

      <a class="read-link" :href="active.articleUrl">
        阅读相关技术文章 <span>→</span>
      </a>
    </div>
  </div>
</template>

<style scoped>
.debug-gallery {
  display: grid;
  grid-template-columns: minmax(260px, 0.92fr) minmax(0, 1.08fr);
  gap: 24px;
  align-items: start;
}

.debug-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.debug-item {
  display: grid;
  grid-template-columns: 148px minmax(0, 1fr);
  gap: 14px;
  align-items: center;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--panel);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    border-color 180ms ease,
    box-shadow 180ms ease,
    transform 180ms ease,
    background 180ms ease;
}

.debug-item:hover,
.debug-item:focus-visible {
  border-color: var(--border-strong);
  background: var(--panel-2);
  outline: none;
}

.debug-item.active {
  border-color: var(--accent);
  background: var(--panel-2);
  box-shadow:
    0 0 0 1px var(--accent),
    0 16px 34px rgba(126, 231, 135, 0.08);
  transform: translateY(-1px);
}

.debug-thumb {
  position: relative;
  display: block;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--code-bg);
}

.debug-thumb img,
.debug-thumb video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.video-badge {
  position: absolute;
  left: 8px;
  bottom: 8px;
  padding: 3px 7px;
  border: 1px solid var(--accent-2-soft-border);
  border-radius: 999px;
  background: rgba(13, 17, 23, 0.82);
  color: var(--accent-2);
  font-family: var(--font-mono);
  font-size: 9px;
  letter-spacing: 0.08em;
}

.debug-title {
  color: var(--text);
  font-size: 14px;
  font-weight: 650;
  line-height: 1.45;
}

.debug-panel {
  position: sticky;
  top: 24px;
  padding: 22px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.panel-head h3 {
  margin: 4px 0 0;
  font-size: 22px;
  letter-spacing: -0.02em;
}

.tag {
  flex: none;
  padding: 5px 10px;
  border: 1px solid var(--accent-2-soft-border);
  border-radius: 999px;
  color: var(--accent-2);
  font-family: var(--font-mono);
  font-size: 11px;
  text-decoration: none;
  transition: background 160ms ease;
}

.tag:hover {
  background: rgba(121, 192, 255, 0.08);
}

.panel-desc {
  margin: 16px 0 18px;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.8;
}

.code-shell {
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: #0b0f14;
}

.code-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
  background: #0f141b;
}

.code-bar span {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--border-strong);
}

.code-bar span:first-child {
  background: #ff7b72;
}

.code-bar span:nth-child(2) {
  background: #d29922;
}

.code-bar span:nth-child(3) {
  background: #3fb950;
}

.code-bar em {
  margin-left: auto;
  color: var(--muted);
  font-family: var(--font-mono);
  font-size: 10px;
  font-style: normal;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.code-content :deep(pre) {
  margin: 0;
  padding: 16px;
  overflow: auto;
  max-height: 340px;
  background: transparent !important;
  font-size: 12.5px;
  line-height: 1.65;
}

.code-content :deep(.shiki) {
  background: transparent !important;
}

.code-content :deep(code) {
  font-family: var(--font-mono);
}

.read-link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 18px;
  color: var(--accent-2);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
}

.read-link span {
  transition: transform 160ms ease;
}

.read-link:hover span {
  transform: translateX(3px);
}

@media (max-width: 860px) {
  .debug-gallery {
    grid-template-columns: 1fr;
  }

  .debug-panel {
    position: static;
  }
}

@media (max-width: 520px) {
  .debug-item {
    grid-template-columns: 110px minmax(0, 1fr);
  }
}
</style>
