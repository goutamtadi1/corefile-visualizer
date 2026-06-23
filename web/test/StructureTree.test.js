import { render, screen, fireEvent } from '@testing-library/svelte'
import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import StructureTree from '../src/lib/StructureTree.svelte'
import { selectedPlugin } from '../src/lib/selection.js'

const corefile = {
  serverBlocks: [
    {
      keys: ['example.org:53'],
      line: 1,
      directives: [
        { name: 'forward', args: ['.', '8.8.8.8'], line: 2 },
        { name: 'cache', line: 3, block: [{ name: 'success', args: ['5000'], line: 4 }] },
      ],
    },
  ],
}

describe('StructureTree', () => {
  beforeEach(() => selectedPlugin.set(null))

  it('renders server block keys', () => {
    render(StructureTree, { corefile })
    expect(screen.getByText('example.org:53')).toBeInTheDocument()
  })

  it('renders directives in order with args', () => {
    render(StructureTree, { corefile })
    expect(screen.getByText('forward')).toBeInTheDocument()
    expect(screen.getByText('. 8.8.8.8')).toBeInTheDocument()
  })

  it('renders nested block directives', () => {
    render(StructureTree, { corefile })
    expect(screen.getByText('success')).toBeInTheDocument()
  })

  it('shows an empty state when corefile is null', () => {
    render(StructureTree, { corefile: null })
    expect(screen.getByTestId('tree-empty')).toBeInTheDocument()
  })

  it('clicking a top-level plugin button updates selectedPlugin store', () => {
    render(StructureTree, { corefile })
    fireEvent.click(screen.getByText('forward'))
    expect(get(selectedPlugin)).toBe('forward')
  })
})
