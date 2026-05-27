<template>
  <button
    type="button"
    class="notifier-toggle"
    :class="{ selected }"
    :aria-pressed="selected"
    @click="$emit('toggle')"
  >
    <span class="notifier-toggle-check">
      <svg v-if="selected" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path d="M2 6l3 3 5-5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </span>
    <span class="notifier-toggle-name">{{ name }}</span>
    <span class="notifier-toggle-badge">{{ type }}</span>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  name: string
  type: string
  selected: boolean
}>()

defineEmits<{
  (e: 'toggle'): void
}>()
</script>

<style scoped>
.notifier-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.4rem 0.75rem 0.4rem 0.5rem;
  border-radius: 7px;
  border: 1px solid rgba(120, 140, 255, 0.18);
  background: rgba(255, 255, 255, 0.03);
  color: #8a8fb8;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  user-select: none;
  text-align: left;
}

.notifier-toggle:hover {
  background: rgba(120, 140, 255, 0.08);
  border-color: rgba(120, 140, 255, 0.35);
  color: #c8cce8;
}

.notifier-toggle.selected {
  background: rgba(99, 118, 255, 0.15);
  border-color: rgba(99, 118, 255, 0.55);
  color: #d4d8ff;
}

.notifier-toggle.selected:hover {
  background: rgba(99, 118, 255, 0.22);
  border-color: rgba(99, 118, 255, 0.75);
}

.notifier-toggle-check {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border: 1.5px solid rgba(120, 140, 255, 0.35);
  background: transparent;
  flex-shrink: 0;
  transition: background 0.15s, border-color 0.15s;
  color: #fff;
}

.notifier-toggle.selected .notifier-toggle-check {
  background: rgba(99, 118, 255, 0.7);
  border-color: rgba(99, 118, 255, 0.9);
}

.notifier-toggle-check svg {
  width: 10px;
  height: 10px;
}

.notifier-toggle-name {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.notifier-toggle-badge {
  font-size: 0.7rem;
  background: rgba(120, 140, 255, 0.1);
  color: #7a88cc;
  padding: 0.1rem 0.4rem;
  border-radius: 4px;
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}

.notifier-toggle.selected .notifier-toggle-badge {
  background: rgba(99, 118, 255, 0.2);
  color: #a0adf0;
}
</style>
