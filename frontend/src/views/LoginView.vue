<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1 class="auth-title">{{ $t('auth.login.title') }}</h1>
      <p class="auth-subtitle">{{ $t('auth.login.subtitle') }}</p>

      <form class="auth-form" @submit.prevent="submit">
        <div class="field">
          <label for="username">{{ $t('auth.username') }}</label>
          <input
            id="username"
            v-model="username"
            type="text"
            autocomplete="username"
            :placeholder="$t('auth.username')"
            required
          />
        </div>

        <div class="field">
          <label for="password">{{ $t('auth.password') }}</label>
          <input
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            :placeholder="$t('auth.password')"
            required
          />
        </div>

        <p v-if="error" class="auth-error">{{ error }}</p>

        <button type="submit" class="auth-btn" :disabled="loading">
          {{ loading ? $t('auth.login.loading') : $t('auth.login.submit') }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function submit() {
  error.value = ''
  loading.value = true
  try {
    await authStore.login(username.value, password.value)
    router.push('/')
  } catch {
    error.value = 'Invalid username or password.'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.auth-card {
  width: 100%;
  max-width: 400px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 2.5rem 2rem;
}

.auth-title {
  margin: 0 0 0.25rem;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text);
}

.auth-subtitle {
  margin: 0 0 2rem;
  font-size: 0.9rem;
  color: var(--text-muted, #9ca3af);
}

.auth-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.field label {
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text);
}

.field input {
  padding: 0.6rem 0.75rem;
  border-radius: 7px;
  border: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.06);
  color: var(--text);
  font-size: 0.95rem;
  outline: none;
  transition: border-color 0.15s;
}

.field input:focus {
  border-color: var(--accent, #6366f1);
}

.auth-error {
  margin: 0;
  font-size: 0.85rem;
  color: #f87171;
}

.auth-btn {
  margin-top: 0.5rem;
  padding: 0.65rem 1rem;
  border-radius: 7px;
  border: none;
  background: var(--accent, #6366f1);
  color: #fff;
  font-size: 0.95rem;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s;
}

.auth-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.auth-btn:not(:disabled):hover {
  opacity: 0.88;
}
</style>
