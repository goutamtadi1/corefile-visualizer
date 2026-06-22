<script>
  /** @type {import('./types.js').Diagnostic[]} */
  export let diagnostics = []
</script>

<div class="validation">
  {#if diagnostics.length === 0}
    <p data-testid="validation-clean">No issues found.</p>
  {:else}
    <ul>
      {#each diagnostics as d}
        <li class="diag diag-{d.severity}" data-testid="diagnostic-row">
          <span class="sev">{d.severity}</span>
          {#if d.line > 0}<span class="loc">line {d.line}</span>{/if}
          <span class="msg">{d.message}</span>
        </li>
      {/each}
    </ul>
  {/if}
</div>
