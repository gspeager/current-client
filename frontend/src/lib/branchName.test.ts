import { describe, expect, it } from 'vitest'
import { issueBranchName, toBranchName } from './branchName'

describe('toBranchName', () => {
  it.each([
    ['my new branch', 'my-new-branch'],
    ['  padded  ', 'padded'],
    ['tabs\tand  runs   of space', 'tabs-and-runs-of-space'],
    ['feature/add login', 'feature/add-login'],
    ['fix: the ~thing^ ?*[x]', 'fix-the-thing-x]'],
    ['back\\slash', 'back-slash'],
    ['bell\u0007and\u007fdel', 'bell-and-del'],
    ['a..b', 'a-b'],
    ['at@{sign', 'at-sign'],
    ['-leading-dash', 'leading-dash'],
    ['.hidden/.part', 'hidden/part'],
    ['ends.', 'ends'],
    ['name.lock', 'name'],
    ['name.lock.lock.', 'name'],
    ['ends-.', 'ends'],
    ['ends.-', 'ends'],
    ['a//b/', 'a/b'],
    ['@', ''],
    ['   ', ''],
    ['already-valid/name_1.2', 'already-valid/name_1.2'],
  ])('%j → %j', (input, expected) => {
    expect(toBranchName(input)).toBe(expected)
  })
})

describe('issueBranchName', () => {
  it.each([
    [12, 'Login fails on Windows', '12-login-fails-on-windows'],
    [7, 'Fix: the `--force` flag (again)!', '7-fix-the-force-flag-again'],
    [3, 'One two three four five six seven eight nine ten', '3-one-two-three-four-five-six-seven-eight'],
    [9, 'Überprüfung für Ärger', '9-überprüfung-für-ärger'],
    [5, '???', '5-issue'],
  ])('%j %j → %j', (number, title, expected) => {
    expect(issueBranchName(number, title)).toBe(expected)
  })
})
