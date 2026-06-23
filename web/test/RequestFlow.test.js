import { render, screen, fireEvent } from '@testing-library/svelte'
import { describe, it, expect, beforeEach } from 'vitest'
import { get } from 'svelte/store'
import RequestFlow from '../src/lib/RequestFlow.svelte'
import { selectedPlugin } from '../src/lib/selection.js'

const corefile = {
  serverBlocks: [
    {
      keys: ['example.org:53'],
      line: 1,
      directives: [],
      flow: [
        { name: 'errors', known: true },
        { name: 'log', known: true },
        { name: 'customplugin', known: false },
      ],
    },
  ],
}

describe('RequestFlow', () => {
  beforeEach(() => selectedPlugin.set(null))

  it('renders flow steps in order', () => {
    render(RequestFlow, { corefile })
    const steps = screen.getAllByTestId('flow-step')
    expect(steps.map((s) => s.textContent.trim())).toEqual(['errors', 'log', 'customplugin'])
  })

  it('marks unknown steps', () => {
    render(RequestFlow, { corefile })
    const unknown = screen.getByText('customplugin')
    expect(unknown.classList.contains('unknown')).toBe(true)
  })

  it('renders request and response endpoints', () => {
    render(RequestFlow, { corefile })
    expect(screen.getByText('request')).toBeInTheDocument()
    expect(screen.getByText('response')).toBeInTheDocument()
  })

  it('shows an empty state when corefile is null', () => {
    render(RequestFlow, { corefile: null })
    expect(screen.getByTestId('flow-empty')).toBeInTheDocument()
  })

  it('clicking a flow step button updates selectedPlugin store', () => {
    render(RequestFlow, { corefile })
    fireEvent.click(screen.getByText('errors'))
    expect(get(selectedPlugin)).toBe('errors')
  })
})
