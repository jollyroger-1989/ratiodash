import { createI18n } from 'vue-i18n'
import en from './locales/en'
import fr from './locales/fr'

// Detect browser language, fallback to 'en'
const browserLang = navigator.language.split('-')[0]
const savedLang = localStorage.getItem('lang')
const locale = savedLang ?? (browserLang === 'fr' ? 'fr' : 'en')

export type MessageSchema = typeof en

const datetimeFormats = {
  en: {
    short: {
      month: 'short' as const,
      day: 'numeric' as const,
      hour: '2-digit' as const,
      minute: '2-digit' as const,
    },
    date: {
      year: 'numeric' as const,
      month: 'short' as const,
      day: 'numeric' as const,
    },
  },
  fr: {
    short: {
      day: '2-digit' as const,
      month: '2-digit' as const,
      year: 'numeric' as const,
      hour: '2-digit' as const,
      minute: '2-digit' as const,
    },
    date: {
      day: '2-digit' as const,
      month: '2-digit' as const,
      year: 'numeric' as const,
    },
  },
}

const i18n = createI18n({
  legacy: false,
  locale,
  fallbackLocale: 'en',
  messages: { en, fr },
  datetimeFormats,
})

export default i18n
