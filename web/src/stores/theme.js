import { defineStore } from 'pinia'
import { ref } from 'vue'

// 三套主题：warm(暖沙米·默认) / cool(冷蓝) / dark(暗黑)
export const THEMES = [
  { value: 'warm', label: '暖沙米', emoji: '🌞' },
  { value: 'cool', label: '冷蓝', emoji: '🌊' },
  { value: 'dark', label: '暗黑', emoji: '🌙' }
]

const VALID = THEMES.map(t => t.value)

const LEGACY_MAP = {
  light: 'warm',
  blue: 'cool'
}

function resolveInitialTheme() {
  let theme = localStorage.getItem('flyiam_theme') || localStorage.getItem('theme')
  if (theme && LEGACY_MAP[theme]) {
    theme = LEGACY_MAP[theme]
  }
  if (!theme || !VALID.includes(theme)) {
    theme = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'warm'
  }
  return theme
}

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref(resolveInitialTheme())

  const setTheme = (theme) => {
    if (!VALID.includes(theme)) {
      theme = 'warm'
    }
    currentTheme.value = theme
    localStorage.setItem('flyiam_theme', theme)
    document.documentElement.setAttribute('data-theme', theme)
    const meta = document.querySelector('meta[name="theme-color"]')
    if (meta) {
      const bg = getComputedStyle(document.body).backgroundColor
      meta.setAttribute('content', bg)
    }
  }

  // 初始化
  setTheme(currentTheme.value)

  return {
    currentTheme,
    setTheme,
    themes: THEMES
  }
})
