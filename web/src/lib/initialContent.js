/**
 * Resolve the editor's initial content. When served by the CLI, GET /corefile
 * returns the piped Corefile as text/plain; otherwise (standalone web app, or a
 * SPA host that returns index.html for unknown paths) we fall back to `sample`.
 *
 * @param {string} sample - fallback content
 * @param {typeof fetch} [fetchFn] - injectable fetch (for testing)
 * @returns {Promise<string>}
 */
export async function loadInitialCorefile(sample, fetchFn = fetch) {
  try {
    const resp = await fetchFn(`${import.meta.env.BASE_URL}corefile`)
    const ct = resp.headers.get('content-type') || ''
    if (resp.ok && ct.includes('text/plain')) {
      const text = await resp.text()
      if (text.length > 0) return text
    }
  } catch {
    // ignore — fall through to sample
  }
  return sample
}
