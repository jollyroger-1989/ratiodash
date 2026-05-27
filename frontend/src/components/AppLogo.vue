<template>
  <RouterLink to="/" class="app-logo">
    <img src="/favicon.svg" :width="size" :height="size" aria-hidden="true" />
    <span class="app-logo__name">RatioDash</span>
  </RouterLink>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router'

withDefaults(defineProps<{ size?: number }>(), { size: 28 })
</script>

<style scoped>
.app-logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  text-decoration: none;
  color: var(--text);
  font-weight: 700;
  font-size: 1.05rem;
  letter-spacing: 0.01em;
}

.app-logo:hover .app-logo__name {
  color: var(--accent);
}

.app-logo img {
  flex-shrink: 0;
}
</style>
