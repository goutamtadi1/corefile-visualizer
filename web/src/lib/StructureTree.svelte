<script>
  import Directives from './Directives.svelte'
  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null
</script>

{#if !corefile}
  <p data-testid="tree-empty">No Corefile parsed.</p>
{:else}
  <ul class="tree">
    {#each corefile.serverBlocks as block}
      <li class="block">
        <span class="keys">{block.keys.join(' ')}</span>
        <Directives directives={block.directives} />
      </li>
    {/each}
  </ul>
{/if}
