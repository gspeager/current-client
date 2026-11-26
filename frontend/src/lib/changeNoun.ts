// GitLab calls a change a merge request; GitHub, and a host not yet known, a pull request.
export const changeNoun = (provider: string) => (provider === 'gitlab' ? 'merge request' : 'pull request')

export const changesLabelFor = (provider: string) => (provider === 'gitlab' ? 'Merge requests' : 'Pull requests')
