<script setup lang="ts">
import { computed } from 'vue'
import type { Architecture } from '../lib/types'

const props = defineProps<{
  architecture: Architecture
}>()

const nodeWidth = 154
const nodeHeight = 68
const nodeMap = computed(() => new Map(props.architecture.nodes.map((node) => [node.id, node])))

const lines = computed(() =>
  props.architecture.edges
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
  <div class="arch-preview">
    <svg viewBox="0 0 800 470" role="img" aria-label="架构图预览">
      <defs>
        <marker
          id="admin-arch-arrow"
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
        class="arch-preview-edge"
        marker-end="url(#admin-arch-arrow)"
      />
      <g
        v-for="node in architecture.nodes"
        :key="node.id"
        class="arch-preview-node"
        :class="node.kind"
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
      </g>
    </svg>
    <p v-if="architecture.nodes.length === 0" class="tag-picker-empty">还没有节点。</p>
  </div>
</template>
