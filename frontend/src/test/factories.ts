import { BranchInfo } from '@current-client-bindings/app'

// The repository most tests open.
export const REPO = '/repos/app'

export const branchInfo = (fields: Partial<BranchInfo> & { name: string }) => new BranchInfo(fields)
