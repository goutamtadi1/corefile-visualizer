import { render, screen, fireEvent } from '@testing-library/svelte'
import { describe, it, expect } from 'vitest'
import ZoneIndex from '../src/lib/ZoneIndex.svelte'

describe('ZoneIndex', () => {
  it('renders a chip per server block with its zone keys', () => {
    const corefile = {
      serverBlocks: [
        { keys: ['cluster.local:53'], suggestions: [] },
        { keys: ['.'], suggestions: ['x'] },
      ],
    }
    render(ZoneIndex, { corefile })
    const chips = screen.getAllByTestId('zone-chip')
    expect(chips).toHaveLength(2)
    expect(chips[0]).toHaveTextContent('cluster.local:53')
  })

  it('shows a tip indicator only for blocks with suggestions', () => {
    const corefile = {
      serverBlocks: [
        { keys: ['a:53'], suggestions: [] },
        { keys: ['b:53'], suggestions: ['add cache'] },
      ],
    }
    render(ZoneIndex, { corefile })
    expect(screen.getAllByTestId('zone-chip-tip')).toHaveLength(1)
  })

  it('renders nothing when corefile is null', () => {
    const { container } = render(ZoneIndex, { corefile: null })
    expect(container.querySelector('.zone-index')).toBeNull()
  })

  it('clicking a chip does not throw when the target section is absent', async () => {
    const corefile = { serverBlocks: [{ keys: ['.'], suggestions: [] }] }
    render(ZoneIndex, { corefile })
    await fireEvent.click(screen.getByTestId('zone-chip'))
  })
})
