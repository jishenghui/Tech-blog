<script setup lang="ts">
import AppIcon from './AppIcon.vue'

defineProps<{
  open: boolean
  title: string
  wide?: boolean
}>()

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
      <section class="modal" :class="{ wide }" role="dialog" aria-modal="true">
        <header class="modal-head">
          <div>
            <p class="eyebrow">ADMIN</p>
            <h3>{{ title }}</h3>
          </div>
          <button class="button-icon" type="button" aria-label="关闭" @click="emit('close')">
            <AppIcon name="close" />
          </button>
        </header>
        <div class="modal-body">
          <slot />
        </div>
        <footer v-if="$slots.actions" class="modal-actions">
          <slot name="actions" />
        </footer>
      </section>
    </div>
  </Teleport>
</template>
