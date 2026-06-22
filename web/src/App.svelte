<script>
  import { onMount } from 'svelte'
  import { loadWasm, analyzeCorefile } from './lib/wasm.js'
  import { loadInitialCorefile } from './lib/initialContent.js'
  import Editor from './lib/Editor.svelte'
  import StructureTree from './lib/StructureTree.svelte'
  import ValidationPanel from './lib/ValidationPanel.svelte'

  const SAMPLE = `example.org:53 {
    log
    errors
    forward . 8.8.8.8
    cache {
        success 5000
    }
}

. {
    whoami
}
`

  /** @type {import('./lib/types.js').Result|null} */
  let result = null
  let loaded = false
  /** @type {string|null} */
  let initialDoc = null
  let timer

  onMount(async () => {
    const [doc] = await Promise.all([
      loadInitialCorefile(SAMPLE),
      loadWasm().then(() => { loaded = true }),
    ])
    initialDoc = doc
    runAnalysis(doc)
  })

  function runAnalysis(text) {
    if (!loaded) return
    result = analyzeCorefile(text)
  }

  function onChange(e) {
    clearTimeout(timer)
    timer = setTimeout(() => runAnalysis(e.detail), 250)
  }
</script>

<main>
  <h1>CoreDNS Corefile Visualizer</h1>
  <div class="layout">
    {#if initialDoc !== null}
      <Editor value={initialDoc} on:change={onChange} />
    {/if}
    <section class="views">
      <StructureTree corefile={result?.corefile ?? null} />
      <ValidationPanel diagnostics={result?.diagnostics ?? []} />
    </section>
  </div>
</main>
