<script>
  import Self from './Directives.svelte'
  /** @type {import('./types.js').Directive[]} */
  export let directives = []
</script>

<ul class="directives">
  {#each directives as d}
    <li class="directive">
      <span class="name">{d.name}</span>
      {#if d.args?.length}
        <span class="args">{d.args.join(' ')}</span>
      {/if}
      {#if d.block?.length}
        <Self directives={d.block} />
      {/if}
    </li>
  {/each}
</ul>
