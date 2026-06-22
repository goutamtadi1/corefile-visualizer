import { render, screen } from '@testing-library/svelte'
import { describe, it, expect } from 'vitest'
import ValidationPanel from '../src/lib/ValidationPanel.svelte'

describe('ValidationPanel', () => {
  it('shows a clean state when there are no diagnostics', () => {
    render(ValidationPanel, { diagnostics: [] })
    expect(screen.getByTestId('validation-clean')).toBeInTheDocument()
  })

  it('renders a row per diagnostic with severity, line, and message', () => {
    render(ValidationPanel, {
      diagnostics: [
        { severity: 'warning', message: 'duplicate zone "."', line: 5 },
        { severity: 'error', message: 'missing closing brace', line: 0 },
      ],
    })
    expect(screen.getAllByTestId('diagnostic-row')).toHaveLength(2)
    expect(screen.getByText('duplicate zone "."')).toBeInTheDocument()
    expect(screen.getByText(/line 5/i)).toBeInTheDocument()
  })
})
