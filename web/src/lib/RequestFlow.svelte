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

  /**
   * Improvement tip for a server block, or null when there are no suggestions
   * (in which case no tooltip is shown).
   */
  function improveTip(block) {
    const tips = block.suggestions
    if (!tips || tips.length === 0) return null
    return '💡 Improve this block:\n' + tips.map((t) => '• ' + t).join('\n')
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
          <span>A DNS query for
            {#if improveTip(block)}
              <code class="zone tip" data-tip={improveTip(block)} data-testid="zone-tip" tabindex="0">{block.keys.join(' ')}</code>
            {:else}
              <code class="zone">{block.keys.join(' ')}</code>
            {/if}
            arrives</span>
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

  /* Zone with improvement suggestions: hoverable, with a CSS-only tooltip. */
  .zone.tip {
    position: relative;
    cursor: help;
    border-bottom: 1px dashed var(--k8s-blue, #326ce5);
  }
  .zone.tip[data-tip]:hover::after,
  .zone.tip[data-tip]:focus-visible::after {
    content: attr(data-tip);
    position: absolute;
    left: 0;
    top: calc(100% + 6px);
    z-index: 10;
    width: max-content;
    max-width: 320px;
    padding: 0.5rem 0.65rem;
    border-radius: 6px;
    background: var(--ink, #1a2233);
    color: #fff;
    font-family: system-ui, -apple-system, sans-serif;
    font-size: 0.78rem;
    font-weight: 400;
    line-height: 1.4;
    white-space: pre-line;
    text-align: left;
    box-shadow: 0 2px 10px rgba(26, 34, 51, 0.25);
    pointer-events: none;
  }
</style>
