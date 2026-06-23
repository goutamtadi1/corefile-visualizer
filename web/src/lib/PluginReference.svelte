<script>
  import { selectedPlugin } from './selection.js'
  /** @type {Record<string, {summary: string, docUrl: string}>} */
  export let catalog = {}
</script>

<div class="reference">
  <h3>Reference</h3>
  {#if $selectedPlugin === null}
    <p data-testid="reference-empty">Select a plugin in the tree or flow to see its reference.</p>
  {:else if catalog[$selectedPlugin]}
    <h3 class="plugin-name">{$selectedPlugin}</h3>
    <p class="summary">{catalog[$selectedPlugin].summary}</p>
    {#if catalog[$selectedPlugin].docUrl}
      <a class="doclink" href={catalog[$selectedPlugin].docUrl} target="_blank" rel="noopener noreferrer">Documentation ↗</a>
    {/if}
  {:else}
    <h3 class="plugin-name">{$selectedPlugin}</h3>
    <p class="unknown" data-testid="reference-unknown">Unrecognized plugin — not a known CoreDNS plugin.</p>
  {/if}
</div>

<style>
  .plugin-name {
    font-family: ui-monospace, monospace;
    margin: 0.2rem 0;
  }
  .doclink {
    font-size: 0.9rem;
  }
</style>
