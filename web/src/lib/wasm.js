/** @typedef {import('./types.js').Result} Result */

let loadingPromise = null

/** Loads wasm_exec.js and instantiates the engine, registering global analyze(). Idempotent and safe under concurrent calls. */
export function loadWasm() {
  if (loadingPromise) return loadingPromise
  loadingPromise = (async () => {
    // wasm_exec.js defines globalThis.Go and is plain (non-module) script text.
    // new Function(...) runs it at global scope; safe as of Go 1.26 because
    // wasm_exec.js assigns to globalThis (not `this`).
    const execSrc = await (await fetch(`${import.meta.env.BASE_URL}wasm/wasm_exec.js`)).text()
    // eslint-disable-next-line no-new-func
    new Function(execSrc)()
    const go = new globalThis.Go()
    const resp = await fetch(`${import.meta.env.BASE_URL}wasm/main.wasm`)
    const { instance } = await WebAssembly.instantiateStreaming(resp, go.importObject)
    go.run(instance) // runs forever (select{}); registers globalThis.analyze
  })()
  return loadingPromise
}

/**
 * Analyzes Corefile text via the WASM engine.
 * @param {string} text
 * @returns {Result}
 */
export function analyzeCorefile(text) {
  if (typeof globalThis.analyze !== 'function') {
    throw new Error('WASM engine not loaded; call loadWasm() first')
  }
  return JSON.parse(globalThis.analyze(text))
}

/**
 * Returns the static plugin catalog from the WASM engine.
 * @returns {Record<string, {summary: string, docUrl: string}>}
 */
export function loadPluginCatalog() {
  if (typeof globalThis.pluginCatalog !== 'function') {
    throw new Error('WASM engine not loaded; call loadWasm() first')
  }
  return JSON.parse(globalThis.pluginCatalog())
}
