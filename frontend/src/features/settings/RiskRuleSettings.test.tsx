import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import RiskRuleSettings from './RiskRuleSettings'

describe('RiskRuleSettings', () => {
  it('turns a rule off and back on', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    const { rerender } = render(<RiskRuleSettings off={[]} onChange={onChange} />)

    await user.click(screen.getByRole('checkbox', { name: 'Conflict markers' }))
    expect(onChange).toHaveBeenLastCalledWith(['conflict-marker'])

    rerender(<RiskRuleSettings off={['conflict-marker']} onChange={onChange} />)
    expect(screen.getByRole('checkbox', { name: 'Conflict markers' })).not.toBeChecked()

    await user.click(screen.getByRole('checkbox', { name: 'Conflict markers' }))
    expect(onChange).toHaveBeenLastCalledWith([])
  })
})
