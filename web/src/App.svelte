<script>
  import { onMount } from 'svelte'
  import { loadWasm, analyzeCorefile } from './lib/wasm.js'
  import Editor from './lib/Editor.svelte'

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
  let timer

  onMount(async () => {
    await loadWasm()
    loaded = true
    runAnalysis(SAMPLE)
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
    <Editor value={SAMPLE} on:change={onChange} />
    <section data-testid="result-summary">
      {#if result?.corefile}
        {result.corefile.serverBlocks.length} server block(s),
        {result.diagnostics.length} diagnostic(s)
      {:else if result}
        parse error
      {:else}
        analyzing…
      {/if}
    </section>
  </div>
</main>
