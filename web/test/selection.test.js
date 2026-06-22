import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import { selectedPlugin } from '../src/lib/selection.js'

describe('selectedPlugin store', () => {
  beforeEach(() => selectedPlugin.set(null))

  it('defaults to null', () => {
    expect(get(selectedPlugin)).toBe(null)
  })

  it('updates when set', () => {
    selectedPlugin.set('forward')
    expect(get(selectedPlugin)).toBe('forward')
  })
})
