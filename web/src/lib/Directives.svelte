<script>
  import Self from './Directives.svelte'
  import { selectedPlugin } from './selection.js'
  /** @type {import('./types.js').Directive[]} */
  export let directives = []
  export let depth = 0
</script>

<ul class="directives">
  {#each directives as d}
    <li class="directive">
      {#if depth === 0}
        <button type="button" class="name" on:click={() => selectedPlugin.set(d.name)}>{d.name}</button>
      {:else}
        <span class="name">{d.name}</span>
      {/if}
      {#if d.args?.length}
        <span class="args">{d.args.join(' ')}</span>
      {/if}
      {#if d.block?.length}
        <Self directives={d.block} depth={depth + 1} />
      {/if}
    </li>
  {/each}
</ul>

<style>
  .name {
    background: none;
    border: none;
    padding: 0;
    font: inherit;
    cursor: pointer;
    text-decoration: underline dotted;
  }
</style>
