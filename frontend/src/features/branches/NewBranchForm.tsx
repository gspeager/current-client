import { useState } from 'react'
import { toBranchName } from '../../lib/branchName'
import IssuePicker from './IssuePicker'
import './NewBranchForm.scss'

export type NewBranchKind = 'branch' | 'feature' | 'release' | 'hotfix'

interface NewBranchFormProps {
  repoPath: string
  onCreate: (kind: NewBranchKind, name: string) => void
}

function NewBranchForm({ repoPath, onCreate }: NewBranchFormProps) {
  const [name, setName] = useState('')
  const [kind, setKind] = useState<NewBranchKind>('branch')
  const branch = toBranchName(name)
  return (
    <div className="branch-sidebar-new">
      <select
        aria-label="new branch kind"
        className="branch-sidebar-new-kind"
        value={kind}
        onChange={(e) => setKind(e.target.value as NewBranchKind)}
      >
        <option value="branch">Branch</option>
        <option value="feature">Start feature</option>
        <option value="release">Start release</option>
        <option value="hotfix">Start hotfix</option>
      </select>
      <div className="branch-sidebar-new-row">
        {kind !== 'branch' && <span className="branch-sidebar-new-prefix">{kind}/</span>}
        <input
          aria-label="new branch name"
          placeholder={kind === 'branch' ? 'New branch name' : 'Name'}
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
        />
        <button type="button" onClick={() => onCreate(kind, branch)} disabled={branch === ''}>
          Create
        </button>
      </div>
      {kind === 'branch' && <IssuePicker repoPath={repoPath} onPick={setName} />}
      {branch !== '' && branch !== name.trim() && (
        <p className="branch-sidebar-new-result" role="status">
          Will be created as{' '}
          <span className="branch-sidebar-new-result-name">
            {kind === 'branch' ? '' : `${kind}/`}
            {branch}
          </span>
        </p>
      )}
    </div>
  )
}

export default NewBranchForm
