export const THEME_STORAGE_KEY = 'rankos-theme'
export const THEME_EVENT = 'rankos-theme'

export type Theme = 'light' | 'dark'

export function getStoredTheme(): Theme {
  try {
    return localStorage.getItem(THEME_STORAGE_KEY) === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

export function applyTheme(theme: Theme) {
  const root = document.documentElement
  root.dataset.theme = theme
  root.classList.toggle('dark', theme === 'dark')
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    /* Storage can be unavailable in private contexts. The attribute still applies. */
  }
  window.dispatchEvent(new CustomEvent(THEME_EVENT, { detail: theme }))
}

export function subscribeTheme(onChange: () => void) {
  const handler = () => onChange()
  window.addEventListener(THEME_EVENT, handler)
  return () => window.removeEventListener(THEME_EVENT, handler)
}

export function getThemeSnapshot(): Theme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
}

/** Chart.js colors from the design tokens. Primary blue, gold accent, and muted status hues. */
export function chartPalette(theme: Theme) {
  const dark = theme === 'dark'
  return {
    primary: '#3295FF',
    gold: '#FFD200',
    tooltipBg: dark ? '#191D23' : '#FFFFFF',
    tooltipBorder: dark ? '#31353A' : '#E4E4E4',
    tooltipText: dark ? '#F7FBFF' : '#1A1C1D',
    tick: dark ? '#9DA1A6' : '#2C2D2E',
    grid: dark ? '#31353A' : '#E4E4E4',
    success: dark ? '#6FCB96' : '#1F7A4D',
    error: dark ? '#F0A0A0' : '#B42318',
    warning: dark ? '#E6C15A' : '#8A5A00',
    info: dark ? '#8EBEFF' : '#1A6BB8',
    neutral: dark ? '#9DA1A6' : '#5E6368',
  }
}
