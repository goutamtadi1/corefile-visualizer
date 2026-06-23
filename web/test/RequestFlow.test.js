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

  it('renders friendly query entry and answer exit endpoints', () => {
    render(RequestFlow, { corefile })
    expect(screen.getByTestId('flow-entry')).toHaveTextContent(/A DNS query for/)
    expect(screen.getByTestId('flow-entry')).toHaveTextContent('example.org:53')
    expect(screen.getByTestId('flow-exit')).toHaveTextContent(/Answer returned to the client/)
  })

  it('shows the plain-English description from the catalog', () => {
    const catalog = { errors: { summary: 'enables error logging.', docUrl: '' } }
    render(RequestFlow, { corefile, catalog })
    expect(screen.getByText('enables error logging.')).toBeInTheDocument()
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
