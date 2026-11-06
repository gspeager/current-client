import { matchAuthor } from './useNameIdentities'

const authors = [
  { name: 'Priya Shah', email: 'priya@example.com' },
  { name: 'Sam Ortiz', email: 'sortiz@example.com' },
  { name: 'Sam Lee', email: 'slee@example.com' },
]

describe('matchAuthor', () => {
  it('matches a full name, ignoring case', () => {
    expect(matchAuthor('priya shah', authors)?.email).toBe('priya@example.com')
  })

  it('matches the local part of an email', () => {
    expect(matchAuthor('sortiz', authors)?.name).toBe('Sam Ortiz')
  })

  it('matches a first name only one author has', () => {
    expect(matchAuthor('priya', authors)?.name).toBe('Priya Shah')
  })

  it('leaves a shared first name and a stranger unmatched', () => {
    expect(matchAuthor('sam', authors)).toBeNull()
    expect(matchAuthor('zoe', authors)).toBeNull()
  })
})
