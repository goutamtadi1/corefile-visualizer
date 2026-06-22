import { render, screen } from '@testing-library/svelte'
import { describe, it, expect } from 'vitest'
import RequestFlow from '../src/lib/RequestFlow.svelte'

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
  it('renders flow steps in order', () => {
    render(RequestFlow, { corefile })
    const steps = screen.getAllByTestId('flow-step')
    expect(steps.map((s) => s.textContent.trim())).toEqual(['errors', 'log', 'customplugin'])
  })

  it('marks unknown steps', () => {
    render(RequestFlow, { corefile })
    const unknown = screen.getByText('customplugin')
    expect(unknown.className).toContain('unknown')
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
})
