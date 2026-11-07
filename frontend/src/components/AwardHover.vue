<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Award } from '../types/poc'

const props = defineProps<{
  awards: Award[]
}>()

const activeIndex = ref(0)
const active = computed(() => props.awards[activeIndex.value] ?? props.awards[0])

function select(index: number) {
  activeIndex.value = index
}
</script>

<template>
  <div class="awards">
    <div class="award-list">
      <button
        v-for="(award, index) in awards"
        :key="award.id"
        type="button"
        class="award-item"
        :class="{ active: index === activeIndex }"
        @mouseenter="select(index)"
        @focus="select(index)"
        @click="select(index)"
      >
        <span class="award-index">{{ String(index + 1).padStart(2, '0') }}</span>
        <span class="award-main">
          <strong>{{ award.title }}</strong>
          <span>{{ award.date }}</span>
        </span>
        <span class="award-arrow">→</span>
      </button>
      <p class="award-hint">悬停或聚焦奖项，右侧查看奖状。</p>
    </div>

    <div v-if="active" class="award-preview">
      <div class="award-image">
        <img :src="active.imageUrl" :alt="active.title" loading="lazy" />
      </div>
      <div class="award-copy">
        <p class="eyebrow">AWARD / {{ active.date }}</p>
        <h3>{{ active.title }}</h3>
        <p>{{ active.description }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.awards {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
  gap: 24px;
  align-items: stretch;
}

.award-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.award-item {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) 24px;
  gap: 14px;
  align-items: center;
  width: 100%;
  padding: 16px 18px;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--panel);
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition:
    border-color 180ms ease,
    background 180ms ease,
    transform 180ms ease;
}

.award-item:hover,
.award-item:focus-visible {
  border-color: var(--border-strong);
  background: var(--panel-2);
  outline: none;
}

.award-item.active {
  border-color: var(--accent);
  background: var(--panel-2);
  transform: translateX(4px);
  box-shadow: 0 12px 30px rgba(126, 231, 135, 0.06);
}

.award-index {
  color: var(--accent);
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.08em;
}

.award-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.award-main strong {
  font-size: 15px;
  font-weight: 650;
}

.award-main span {
  color: var(--muted);
  font-size: 12px;
}

.award-arrow {
  color: var(--muted);
  font-size: 18px;
  transition:
    color 180ms ease,
    transform 180ms ease;
}

.award-item.active .award-arrow {
  color: var(--accent);
  transform: translateX(3px);
}

.award-hint {
  margin: 8px 4px 0;
  color: var(--muted);
  font-size: 12px;
}

.award-preview {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

.award-image {
  position: relative;
  aspect-ratio: 16 / 10;
  overflow: hidden;
  background: var(--code-bg);
}

.award-image::after {
  position: absolute;
  inset: 0;
  border-bottom: 1px solid var(--border);
  background: linear-gradient(180deg, transparent 55%, rgba(7, 9, 13, 0.45));
  content: '';
  pointer-events: none;
}

.award-image img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 400ms ease;
}

.award-preview:hover .award-image img {
  transform: scale(1.025);
}

.award-copy {
  padding: 20px 22px 22px;
}

.award-copy h3 {
  margin: 4px 0 8px;
  font-size: 20px;
  letter-spacing: -0.02em;
}

.award-copy p:last-child {
  margin: 0;
  color: var(--muted);
  font-size: 14px;
  line-height: 1.75;
}

@media (max-width: 860px) {
  .awards {
    grid-template-columns: 1fr;
  }
}
</style>
