import { describe, it, expect } from 'vitest'
import { loadInitialCorefile } from '../src/lib/initialContent.js'

const SAMPLE = '. {\n  whoami\n}\n'

function resp({ ok = true, ct = 'text/plain; charset=utf-8', body = '' }) {
  return Promise.resolve({
    ok,
    headers: { get: (h) => (h.toLowerCase() === 'content-type' ? ct : null) },
    text: () => Promise.resolve(body),
  })
}

describe('loadInitialCorefile', () => {
  it('uses /corefile text when ok and content-type is text/plain', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ body: 'piped-corefile' }))
    expect(got).toBe('piped-corefile')
  })

  it('falls back to sample when content-type is html (SPA fallback)', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ ct: 'text/html', body: '<html>' }))
    expect(got).toBe(SAMPLE)
  })

  it('falls back to sample on non-ok response', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ ok: false, body: 'nope' }))
    expect(got).toBe(SAMPLE)
  })

  it('falls back to sample on empty body', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => resp({ body: '' }))
    expect(got).toBe(SAMPLE)
  })

  it('falls back to sample when fetch throws', async () => {
    const got = await loadInitialCorefile(SAMPLE, () => Promise.reject(new Error('network')))
    expect(got).toBe(SAMPLE)
  })
})
