import { Links, Style } from '@current-client-bindings/core/hostlink'
import { TrackerLinks } from '@current-client-bindings/app'
import { branchIssue, branchTrackerKey, splitReferences } from './hostReferences'

const github = new Links({
  issue: 'https://github.com/o/r/issues/',
  root: 'https://github.com/',
  issuePath: '/issues/',
  style: Style.GitHub,
})
const gitlab = new Links({
  issue: 'https://gitlab.com/g/s/r/-/issues/',
  mergeRequest: 'https://gitlab.com/g/s/r/-/merge_requests/',
  root: 'https://gitlab.com/',
  issuePath: '/-/issues/',
  style: Style.GitLab,
})

describe('splitReferences', () => {
  it('links issues, cross-project issues and merge requests', () => {
    expect(splitReferences('Fix login (#12), see o/other#3 and !45', gitlab)).toEqual([
      'Fix login (',
      { text: '#12', url: 'https://gitlab.com/g/s/r/-/issues/12', provider: 'gitlab' },
      '), see ',
      { text: 'o/other#3', url: 'https://gitlab.com/o/other/-/issues/3', provider: 'gitlab' },
      ' and ',
      { text: '!45', url: 'https://gitlab.com/g/s/r/-/merge_requests/45', provider: 'gitlab' },
    ])
  })

  it('links nested groups and references at the ends of the text', () => {
    expect(splitReferences('#1 then group/sub/project#2', gitlab)).toEqual([
      { text: '#1', url: 'https://gitlab.com/g/s/r/-/issues/1', provider: 'gitlab' },
      ' then ',
      { text: 'group/sub/project#2', url: 'https://gitlab.com/group/sub/project/-/issues/2', provider: 'gitlab' },
    ])
  })

  it('leaves merge request syntax alone where the host has none', () => {
    expect(splitReferences('Merge !45 for #7', github)).toEqual([
      'Merge !45 for ',
      { text: '#7', url: 'https://github.com/o/r/issues/7', provider: 'github' },
    ])
  })

  it('ignores what only looks like a reference', () => {
    for (const text of ['color #fff', 'item#12', 'https://x.io/a/b#1', '#12abc', '# 12', '#']) {
      expect(splitReferences(text, github)).toEqual([text])
    }
  })

  it('is plain text without a host', () => {
    expect(splitReferences('Fix #12', null)).toEqual(['Fix #12'])
  })
})

describe('branchIssue', () => {
  it('reads the issue a branch was created for', () => {
    expect(branchIssue('123-fix-login', github)).toEqual({
      text: '123',
      url: 'https://github.com/o/r/issues/123',
      provider: 'github',
    })
  })

  it('needs a number and a dash at the start, and a host', () => {
    expect(branchIssue('fix-123', github)).toBeNull()
    expect(branchIssue('123', github)).toBeNull()
    expect(branchIssue('feature/123-login', github)).toBeNull()
    expect(branchIssue('123-fix-login', null)).toBeNull()
  })
})

// The patterns TrackerService.Links sends for a tracker limited to PROJ and OPS.
const jira = new TrackerLinks({
  site: 'https://example.atlassian.net/',
  kind: 'jira-cloud',
  browse: 'https://example.atlassian.net/browse/',
  keyPattern: '(?:^|[^\\w/.-])((?:PROJ|OPS)-[1-9]\\d*)\\b',
  branchKeyPattern: '(?:^|/)((?:PROJ|OPS)-[1-9]\\d*)(?:[-_/]|$)',
})

describe('tracker keys', () => {
  it('links keys beside host references, and with no host at all', () => {
    expect(splitReferences('PROJ-12: fix login (#3), see OPS-4 not UTF-8', github, jira)).toEqual([
      { text: 'PROJ-12', url: 'https://example.atlassian.net/browse/PROJ-12', provider: 'jira' },
      ': fix login (',
      { text: '#3', url: 'https://github.com/o/r/issues/3', provider: 'github' },
      '), see ',
      { text: 'OPS-4', url: 'https://example.atlassian.net/browse/OPS-4', provider: 'jira' },
      ' not UTF-8',
    ])
    expect(splitReferences('done in PROJ-7', null, jira)).toEqual([
      'done in ',
      { text: 'PROJ-7', url: 'https://example.atlassian.net/browse/PROJ-7', provider: 'jira' },
    ])
  })

  it('leaves keys inside paths and longer words alone', () => {
    expect(splitReferences('https://x/PROJ-1 and XPROJ-2', null, jira)).toEqual(['https://x/PROJ-1 and XPROJ-2'])
  })

  it('finds the key a branch is named after', () => {
    expect(branchTrackerKey('feature/PROJ-12-fix-login', jira)).toEqual({
      text: 'PROJ-12',
      url: 'https://example.atlassian.net/browse/PROJ-12',
      provider: 'jira',
      start: 8,
    })
    expect(branchTrackerKey('12-fix-login', jira)).toBeNull()
    expect(branchTrackerKey('PROJ-12', null)).toBeNull()
  })
})
