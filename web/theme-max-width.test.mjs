import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'
import postcss from 'postcss'
import tailwindcss from '@tailwindcss/postcss'

const webRoot = path.dirname(fileURLToPath(import.meta.url))

// Dialogs use these. xs/sm/md/lg/xl on the spacing scale would replace them
// with a few pixels and collapse every dialog.
const containerWidths = {
  'max-w-md': '28rem',
  'max-w-lg': '32rem',
  'max-w-2xl': '42rem',
}

function ruleBody(css, className) {
  const escaped = className.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = css.match(new RegExp(`\\.${escaped}\\s*\\{([^}]+)\\}`))
  assert.ok(match, `missing .${className} rule`)
  return match[1]
}

test('dialog max-width utilities use the container scale', async () => {
  const from = path.join(webRoot, 'src/index.css')
  const source = readFileSync(from, 'utf8')
  for (const name of ['xs', 'sm', 'md', 'lg', 'xl']) {
    assert.doesNotMatch(
      source,
      new RegExp(`--spacing-${name}\\s*:`),
      `--spacing-${name} overrides max-w-${name}`,
    )
  }
  const result = await postcss([tailwindcss()]).process(source, { from })
  const css = result.css

  for (const [className, width] of Object.entries(containerWidths)) {
    const body = ruleBody(css, className)
    assert.match(
      body,
      new RegExp(`max-width:\\s*var\\(--container-${className.slice('max-w-'.length)}\\)`),
      `${className} should be the ${width} container width, got:${body}`,
    )
    assert.doesNotMatch(
      body,
      /var\(--spacing-/,
      `${className} must not resolve to a spacing token:${body}`,
    )
  }
})
