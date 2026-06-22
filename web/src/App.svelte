<script>
  import { onMount } from 'svelte'
  import { loadWasm, analyzeCorefile } from './lib/wasm.js'

  let status = 'loading…'
  onMount(async () => {
    try {
      await loadWasm()
      const r = analyzeCorefile('. {\n    whoami\n}\n')
      status = `engine ready — ${r.corefile.serverBlocks.length} block(s)`
    } catch (e) {
      status = `error: ${e.message}`
    }
  })
</script>

<main>
  <h1>CoreDNS Corefile Visualizer</h1>
  <p data-testid="status">{status}</p>
</main>
