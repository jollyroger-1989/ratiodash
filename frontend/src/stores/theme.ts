import { ref } from 'vue'
import { defineStore } from 'pinia'

export type ThemeId = 'nebula' | 'light' | 'tokyo-night' | 'dracula'

export interface ThemeOption {
  id: ThemeId
  labelKey: string
  swatches: string[]
}

export const themes: ThemeOption[] = [
  {
    id: 'nebula',
    labelKey: 'settings.theme.nebula',
    swatches: ['#020817', '#818cf8', '#c084fc'],
  },
  {
    id: 'light',
    labelKey: 'settings.theme.light',
    swatches: ['#f3f6fb', '#4f46e5', '#ec4899'],
  },
  {
    id: 'tokyo-night',
    labelKey: 'settings.theme.tokyo-night',
    swatches: ['#1a1b26', '#7aa2f7', '#bb9af7'],
  },
  {
    id: 'dracula',
    labelKey: 'settings.theme.dracula',
    swatches: ['#282a36', '#bd93f9', '#ff79c6'],
  },
]

export const useThemeStore = defineStore('theme', () => {
  const rawStored = localStorage.getItem('theme')
  const stored = rawStored === 'darcula' ? 'dracula' : rawStored
  const currentTheme = ref<ThemeId>(
    stored === 'nebula' || stored === 'light' || stored === 'tokyo-night' || stored === 'dracula'
      ? stored
      : 'nebula'
  )

  function setTheme(id: ThemeId) {
    currentTheme.value = id
    document.documentElement.dataset.theme = id
    localStorage.setItem('theme', id)
  }

  // Sync the DOM on store init
  document.documentElement.dataset.theme = currentTheme.value

  return { currentTheme, setTheme, themes }
})
