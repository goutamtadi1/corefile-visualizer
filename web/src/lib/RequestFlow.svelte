<script>
  import { selectedPlugin } from './selection.js'
  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null
</script>

{#if !corefile}
  <p data-testid="flow-empty">No Corefile parsed.</p>
{:else}
  <div class="flows">
    {#each corefile.serverBlocks as block}
      <section class="flow-block">
        <span class="keys">{block.keys.join(' ')}</span>
        <div class="chain">
          <span class="endpoint">request</span>
          {#each (block.flow ?? []) as step}
            <span class="arrow">→</span>
            <button type="button" class="step {step.known ? 'known' : 'unknown'}" data-testid="flow-step" on:click={() => selectedPlugin.set(step.name)}>{step.name}</button>
          {/each}
          <span class="arrow">→</span>
          <span class="endpoint">response</span>
        </div>
      </section>
    {/each}
  </div>
{/if}

<style>
  .chain {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.4rem;
  }
  .step,
  .endpoint {
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    border: 1px solid currentColor;
    font-family: ui-monospace, monospace;
    font-size: 0.85rem;
  }
  .step {
    background: none;
    font: inherit;
    cursor: pointer;
  }
  .endpoint {
    opacity: 0.7;
  }
  .step.unknown {
    border-style: dashed;
    opacity: 0.7;
  }
  .arrow {
    opacity: 0.5;
  }
  .keys {
    font-weight: 600;
    display: block;
    margin-bottom: 0.3rem;
  }
  .flow-block {
    margin-bottom: 1rem;
  }
</style>
