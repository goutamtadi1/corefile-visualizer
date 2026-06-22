/** @typedef {import('./types.js').Result} Result */

let ready = false

/** Loads wasm_exec.js and instantiates the engine, registering global analyze(). */
export async function loadWasm() {
  if (ready) return
  // wasm_exec.js defines globalThis.Go and is plain (non-module) script text.
  const execSrc = await (await fetch(`${import.meta.env.BASE_URL}wasm/wasm_exec.js`)).text()
  // eslint-disable-next-line no-new-func
  new Function(execSrc)()
  const go = new globalThis.Go()
  const resp = await fetch(`${import.meta.env.BASE_URL}wasm/main.wasm`)
  const { instance } = await WebAssembly.instantiateStreaming(resp, go.importObject)
  go.run(instance) // runs forever (select{}); registers globalThis.analyze
  ready = true
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
