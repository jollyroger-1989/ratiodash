<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="modal-backdrop"
      @click.self="$emit('close')"
      @keyup.esc.window="$emit('close')"
    >
      <div
        class="modal"
        :class="`modal--${size}`"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
      >
        <div class="modal-header">
          <h2 :id="titleId"><slot name="title" /></h2>
          <button class="modal-close" @click="$emit('close')" :aria-label="$t('common.cancel')">
            <font-awesome-icon :icon="['fas', 'xmark']" />
          </button>
        </div>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  modelValue: boolean
  titleId: string
  size?: 'sm' | 'md' | 'lg'
}>(), { size: 'md' })

defineEmits<{ close: [] }>()
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(2, 8, 23, 0.75);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: rgba(10, 16, 50, 0.95);
  border: 1px solid var(--border-bright);
  border-radius: 14px;
  padding: 2.5rem;
  width: min(680px, 92vw);
  max-height: 90dvh;
  overflow-y: auto;
  box-shadow: 0 8px 48px rgba(0, 0, 0, 0.7), 0 0 40px rgba(129, 140, 248, 0.12);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
}

.modal--sm { width: min(480px, 94vw); }
.modal--lg { width: min(960px, 92vw); }

@media (max-width: 640px) {
  .modal,
  .modal--sm,
  .modal--lg {
    width: 96vw;
    padding: 1.5rem;
    border-radius: 10px;
  }

  .modal-backdrop {
    align-items: flex-end;
    padding-bottom: env(safe-area-inset-bottom);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
  }

  .modal, .modal--sm, .modal--lg {
    border-bottom-left-radius: 0;
    border-bottom-right-radius: 0;
    max-height: 92dvh;
  }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text);
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  color: var(--text-muted);
  padding: 0 0.25rem;
  transition: color 0.15s;
}
.modal-close:hover { color: var(--text); }
</style>
