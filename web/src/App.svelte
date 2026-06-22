<script>
  import { onMount } from 'svelte'
  import { loadWasm, analyzeCorefile, loadPluginCatalog } from './lib/wasm.js'
  import { loadInitialCorefile } from './lib/initialContent.js'
  import Editor from './lib/Editor.svelte'
  import StructureTree from './lib/StructureTree.svelte'
  import ValidationPanel from './lib/ValidationPanel.svelte'
  import RequestFlow from './lib/RequestFlow.svelte'
  import PluginReference from './lib/PluginReference.svelte'

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
  /** @type {Record<string, {summary: string, docUrl: string}>} */
  let catalog = {}

  onMount(async () => {
    const wasmReady = loadWasm()
      .then(() => {
        loaded = true
        try { catalog = loadPluginCatalog() } catch { /* catalog optional */ }
      })
      .catch(() => { /* wasm failed; editor still renders, analysis unavailable */ })
    initialDoc = await loadInitialCorefile(SAMPLE)
    await wasmReady
    if (loaded) runAnalysis(initialDoc)
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
    <section class="pane pane-source" aria-label="Corefile source">
      <h2>Corefile</h2>
      {#if initialDoc !== null}
        <Editor value={initialDoc} on:change={onChange} />
      {/if}
    </section>
    <section class="pane pane-views" aria-label="Visualization">
      <h2>Visualization</h2>
      <div class="views">
        <StructureTree corefile={result?.corefile ?? null} />
        <ValidationPanel diagnostics={result?.diagnostics ?? []} />
        <RequestFlow corefile={result?.corefile ?? null} />
        <PluginReference {catalog} />
      </div>
    </section>
  </div>
</main>

<style>
  main {
    max-width: 1200px;
    margin: 0 auto;
    padding: 1rem;
  }

  /* Two panes: source (Corefile) on the left, generated views on the right. */
  .layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
    align-items: start;
  }

  .pane {
    min-width: 0; /* allow children to shrink instead of overflowing the grid */
  }

  .pane h2 {
    font-size: 0.95rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    opacity: 0.6;
    margin: 0 0 0.5rem;
  }

  /* Right pane scrolls independently of the editor on tall content. */
  .pane-views .views {
    max-height: calc(100vh - 8rem);
    overflow: auto;
  }

  /* Stack vertically (editor on top) on narrow screens. */
  @media (max-width: 768px) {
    .layout {
      grid-template-columns: 1fr;
    }
    .pane-views .views {
      max-height: none;
    }
  }
</style>
