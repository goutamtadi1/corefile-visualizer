<script>
  import Self from './Directives.svelte'
  import { selectedPlugin } from './selection.js'
  /** @type {import('./types.js').Directive[]} */
  export let directives = []
</script>

<ul class="directives">
  {#each directives as d}
    <li class="directive">
      <button type="button" class="name" on:click={() => selectedPlugin.set(d.name)}>{d.name}</button>
      {#if d.args?.length}
        <span class="args">{d.args.join(' ')}</span>
      {/if}
      {#if d.block?.length}
        <Self directives={d.block} />
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
    color: inherit;
    cursor: pointer;
    text-decoration: underline dotted;
  }
</style>
