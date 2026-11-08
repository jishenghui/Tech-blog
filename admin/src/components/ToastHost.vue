<script setup lang="ts">
import { useToastStore } from '../stores/toast'

const toast = useToastStore()
</script>

<template>
  <div class="toast-host" aria-live="polite" aria-atomic="true">
    <TransitionGroup name="toast">
      <div v-for="message in toast.messages" :key="message.id" class="toast" :class="message.type">
        {{ message.text }}
      </div>
    </TransitionGroup>
  </div>
</template>
