<script>
  import { selectedPlugin } from './selection.js'
  import { pluginIcon } from './pluginCategory.js'

  /** @type {import('./types.js').Corefile|null} */
  export let corefile = null
  /** @type {Record<string, {summary: string, docUrl: string}>} */
  export let catalog = {}

  /** Plain-English description for a flow step. */
  function describe(step) {
    if (catalog[step.name]) return catalog[step.name].summary
    return step.known ? '' : 'custom plugin — not a recognized CoreDNS plugin'
  }
</script>

{#if !corefile}
  <p data-testid="flow-empty">No Corefile parsed.</p>
{:else}
  <div class="flows">
    {#each corefile.serverBlocks as block}
      <section class="journey">
        <div class="endpoint entry" data-testid="flow-entry">
          <span class="endpoint-icon">🔎</span>
          <span>A DNS query for <code class="zone">{block.keys.join(' ')}</code> arrives</span>
        </div>

        <ol class="steps">
          {#each (block.flow ?? []) as step, i}
            <li class="step" class:unknown={!step.known}>
              <span class="num">{i + 1}</span>
              <span class="icon" aria-hidden="true">{pluginIcon(step.name)}</span>
              <button
                type="button"
                class="plugin {step.known ? 'known' : 'unknown'}"
                data-testid="flow-step"
                on:click={() => selectedPlugin.set(step.name)}
              >{step.name}</button>
              {#if describe(step)}
                <span class="desc">{describe(step)}</span>
              {/if}
            </li>
          {/each}
        </ol>

        <div class="endpoint exit" data-testid="flow-exit">
          <span class="endpoint-icon">✅</span>
          <span>Answer returned to the client</span>
        </div>
      </section>
    {/each}
  </div>
{/if}

<style>
  .journey {
    margin-bottom: 1.5rem;
  }

  .endpoint {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    padding: 0.4rem 0.2rem;
  }
  .endpoint .zone {
    font-family: ui-monospace, monospace;
    background: var(--k8s-blue-soft, #e8f0fe);
    padding: 0.05rem 0.35rem;
    border-radius: 4px;
  }
  .endpoint-icon {
    font-size: 1.1rem;
  }

  /* Vertical stepper with a connecting rail down the left of the number badges. */
  .steps {
    list-style: none;
    margin: 0;
    padding: 0;
    position: relative;
  }
  .steps::before {
    content: '';
    position: absolute;
    left: 0.7rem;
    top: 0.2rem;
    bottom: 0.2rem;
    width: 2px;
    background: var(--k8s-blue-soft, #e8f0fe);
  }

  .step {
    position: relative;
    display: flex;
    align-items: baseline;
    gap: 0.5rem;
    padding: 0.35rem 0;
  }

  .num {
    flex: 0 0 1.5rem;
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    background: var(--k8s-blue, #326ce5);
    color: #fff;
    font-size: 0.75rem;
    line-height: 1.5rem;
    text-align: center;
    z-index: 1;
  }
  .step.unknown .num {
    background: var(--orange, #ed8b00);
  }

  .icon {
    font-size: 1rem;
  }

  .plugin {
    appearance: none;
    background: none;
    border: none;
    padding: 0;
    font-family: ui-monospace, monospace;
    font-weight: 600;
    cursor: pointer;
    color: var(--k8s-blue-dark, #2851b8);
  }
  .plugin:hover {
    color: var(--orange, #ed8b00);
  }
  .plugin.unknown {
    color: var(--orange, #ed8b00);
  }

  .desc {
    color: #5b6b85;
    font-size: 0.85rem;
  }
</style>
