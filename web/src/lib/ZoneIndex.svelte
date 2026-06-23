<script>
  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null

  /** Scroll the matching server-block flow section into view. */
  function jumpTo(index) {
    const el = document.getElementById(`zone-${index}`)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
</script>

{#if corefile && corefile.serverBlocks.length}
  <nav class="zone-index" aria-label="DNS server blocks">
    {#each corefile.serverBlocks as block, i}
      <button type="button" class="zone-chip" data-testid="zone-chip" on:click={() => jumpTo(i)}>
        {block.keys.join(' ')}
        {#if block.suggestions?.length}
          <span class="tip-dot" title="Has improvement tips" data-testid="zone-chip-tip" aria-label="has improvement tips">💡</span>
        {/if}
      </button>
    {/each}
  </nav>
{/if}

<style>
  .zone-index {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 0.25rem;
  }
  .zone-chip {
    appearance: none;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    font-family: ui-monospace, monospace;
    font-size: 0.8rem;
    padding: 0.25rem 0.65rem;
    border-radius: 999px;
    border: 1px solid var(--k8s-blue, #326ce5);
    background: var(--k8s-blue-soft, #e8f0fe);
    color: var(--k8s-blue-dark, #2851b8);
    cursor: pointer;
  }
  .zone-chip:hover {
    background: var(--k8s-blue, #326ce5);
    color: #fff;
  }
  .tip-dot {
    font-size: 0.75rem;
  }
</style>
