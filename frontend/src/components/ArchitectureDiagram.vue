<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ArchEdge, ArchNode } from '../types/poc'

const props = defineProps<{
  nodes: ArchNode[]
  edges: ArchEdge[]
}>()

const nodeWidth = 154
const nodeHeight = 68
const activeId = ref(props.nodes[0]?.id ?? '')
const active = computed(() => props.nodes.find((node) => node.id === activeId.value) ?? props.nodes[0])
const nodeMap = computed(() => new Map(props.nodes.map((node) => [node.id, node])))

const lines = computed(() =>
  props.edges
    .map((edge) => {
      const from = nodeMap.value.get(edge.from)
      const to = nodeMap.value.get(edge.to)
      if (!from || !to) {
        return null
      }
      return {
        x1: from.x + nodeWidth / 2,
        y1: from.y + nodeHeight / 2,
        x2: to.x + nodeWidth / 2,
        y2: to.y + nodeHeight / 2,
      }
    })
    .filter((line): line is NonNullable<typeof line> => line !== null),
)
</script>

<template>
  <div class="architecture">
    <div class="arch-canvas">
      <svg viewBox="0 0 800 470" role="group" aria-label="项目架构图">
        <defs>
          <marker
            id="arch-arrow"
            viewBox="0 0 10 10"
            refX="8"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path d="M 0 0 L 10 5 L 0 10 z" fill="var(--border-strong)" />
          </marker>
        </defs>

        <line
          v-for="(line, index) in lines"
          :key="index"
          v-bind="line"
          class="arch-edge"
          marker-end="url(#arch-arrow)"
        />

        <a
          v-for="node in nodes"
          :key="node.id"
          :href="node.href"
          class="arch-node"
          :class="[node.kind, { active: node.id === activeId }]"
          tabindex="0"
          @mouseenter="activeId = node.id"
          @focus="activeId = node.id"
        >
          <rect :x="node.x" :y="node.y" :width="nodeWidth" :height="nodeHeight" rx="16" />
          <text
            :x="node.x + nodeWidth / 2"
            :y="node.y + nodeHeight / 2"
            text-anchor="middle"
            dominant-baseline="middle"
          >
            {{ node.label }}
          </text>
        </a>
      </svg>
    </div>

    <aside v-if="active" class="arch-panel">
      <p class="eyebrow">COMPONENT</p>
      <h3>{{ active.label }}</h3>
      <p>{{ active.description }}</p>
      <a class="read-link" :href="active.href">
        查看相关技术文章 <span>→</span>
      </a>
    </aside>
  </div>
</template>

<style scoped>
.architecture {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.6fr);
  gap: 24px;
  align-items: stretch;
}

.arch-canvas {
  overflow: auto;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background:
    radial-gradient(circle at 20% 10%, var(--arch-glow), transparent 40%),
    var(--panel);
  box-shadow: var(--shadow);
}

.arch-canvas svg {
  display: block;
  width: 100%;
  min-width: 0;
  height: auto;
}

.arch-edge {
  stroke: var(--border-strong);
  stroke-width: 1.4;
  stroke-dasharray: 4 6;
}

.arch-node {
  cursor: pointer;
  outline: none;
}

.arch-node rect {
  fill: var(--panel-2);
  stroke: var(--border-strong);
  stroke-width: 1.4;
  transition:
    fill 180ms ease,
    stroke 180ms ease,
    filter 180ms ease;
}

.arch-node text {
  fill: var(--text);
  font-family: var(--font-sans);
  font-size: 15px;
  font-weight: 650;
  letter-spacing: -0.01em;
  pointer-events: none;
}

.arch-node.frontend rect {
  stroke: var(--accent-2);
}

.arch-node.backend rect {
  stroke: var(--accent);
}

.arch-node.data rect {
  stroke: var(--accent-3);
}

.arch-node.infra rect {
  stroke: #d2a8ff;
}

.arch-node.admin rect {
  stroke: #ff7b72;
}

.arch-node:hover rect,
.arch-node:focus rect,
.arch-node.active rect {
  fill: var(--panel);
  stroke-width: 2.2;
  filter: drop-shadow(0 0 16px rgba(121, 192, 255, 0.16));
}

.arch-node:focus rect {
  stroke-dasharray: 6 4;
}

.arch-panel {
  position: sticky;
  top: 24px;
  align-self: start;
  padding: 24px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

.arch-panel h3 {
  margin: 4px 0 10px;
  font-size: 22px;
  letter-spacing: -0.02em;
}

.arch-panel p:not(.eyebrow) {
  margin: 0;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.8;
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

@media (max-width: 900px) {
  .architecture {
    grid-template-columns: 1fr;
  }

  .arch-panel {
    position: static;
  }
}
</style>
