package repository

const maxRecentRepositories = 10

func AddRecent(recent []string, path string) []string {
	deduped := make([]string, 0, len(recent)+1)
	deduped = append(deduped, path)
	for _, p := range recent {
		if p != path {
			deduped = append(deduped, p)
		}
	}
	if len(deduped) > maxRecentRepositories {
		deduped = deduped[:maxRecentRepositories]
	}
	return deduped
}

func RemoveRecent(recent []string, path string) []string {
	kept := make([]string, 0, len(recent))
	for _, p := range recent {
		if p != path {
			kept = append(kept, p)
		}
	}
	return kept
}
