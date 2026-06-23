import { render, screen, act } from '@testing-library/svelte'
import { describe, it, expect, beforeEach } from 'vitest'
import PluginReference from '../src/lib/PluginReference.svelte'
import { selectedPlugin } from '../src/lib/selection.js'

const catalog = {
  forward: { summary: 'facilitates proxying DNS messages to upstream resolvers.', docUrl: 'https://coredns.io/plugins/forward/' },
  on: { summary: 'executes shell commands on lifecycle events.', docUrl: '' },
}

describe('PluginReference', () => {
  beforeEach(() => selectedPlugin.set(null))

  it('shows empty state when nothing selected', () => {
    render(PluginReference, { catalog })
    expect(screen.getByTestId('reference-empty')).toBeInTheDocument()
  })

  it('shows summary and doc link for a known plugin', async () => {
    render(PluginReference, { catalog })
    await act(() => selectedPlugin.set('forward'))
    expect(screen.getByText(/proxying DNS messages/)).toBeInTheDocument()
    const link = screen.getByRole('link')
    expect(link.getAttribute('href')).toBe('https://coredns.io/plugins/forward/')
  })

  it('omits the link when docUrl is empty', async () => {
    render(PluginReference, { catalog })
    await act(() => selectedPlugin.set('on'))
    expect(screen.getByText(/lifecycle events/)).toBeInTheDocument()
    expect(screen.queryByRole('link')).toBeNull()
  })

  it('shows unrecognized note for a plugin not in the catalog', async () => {
    render(PluginReference, { catalog })
    await act(() => selectedPlugin.set('bogus'))
    expect(screen.getByTestId('reference-unknown')).toBeInTheDocument()
  })
})
