package risk

import (
	"regexp"

	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/git"
)

var pathRules = []struct {
	rule  string
	what  string
	paths *regexp.Regexp
}{
	{RuleCI, "CI file", regexp.MustCompile(`^\.(github|gitea|forgejo)/(workflows|actions)/|^\.gitlab-ci\.ya?ml$|^\.gitlab/ci/|^\.(circleci|buildkite|woodpecker|teamcity)/` +
		`|^\.(travis|drone|woodpecker|appveyor|cirrus)\.ya?ml$|^(appveyor|azure-pipelines[^/]*|bitbucket-pipelines|cloudbuild|codemagic)\.ya?ml$|(^|/)Jenkinsfile[^/]*$`)},
	{RuleAgentInstructions, "Agent instructions", regexp.MustCompile(`(^|/)(AGENTS|CLAUDE|CLAUDE\.local|GEMINI)\.md$|(^|/)\.(cursorrules|windsurfrules|clinerules|roorules)$` +
		`|(^|/)\.(cursor|claude|codex|gemini|clinerules|windsurf|junie|roo)/|(^|/)\.mcp\.json$|^\.github/(copilot-instructions\.md$|instructions/|prompts/|agents/|chatmodes/)|(^|/)\.aider\.conf\.ya?ml$`)},
	{RuleMigration, "Database migration", regexp.MustCompile(`(?i)(^|/)(migrations?|migrate)/(.*/)?([^/]+\.sql|\d{3}[^/]*\.(rb|py|[cm]?[jt]s|go|php|cs|java|kt|exs?))$` +
		`|(^|/)(alembic|migrations)/versions/[^/]+\.py$|(^|/)db/changelog/(.*/)?[^/]+\.(sql|xml|ya?ml|json)$|(^|/)[VU]\d[\w.]*__[^/]+\.(sql|java|kt)$`)},
}

var statusWord = map[string]string{"A": "added", "D": "deleted"}

func pathFindings(files []git.ChangedFile) []Finding {
	var findings []Finding
	for _, rule := range pathRules {
		for _, file := range files {
			if !rule.paths.MatchString(file.Path) || (rule.rule == RuleMigration && filekind.Of(file.Path) != filekind.Code) {
				continue
			}
			word := statusWord[file.Status]
			if word == "" {
				word = "changed"
			}
			findings = append(findings, Finding{Rule: rule.rule, Path: file.Path, Message: rule.what + " " + word + "."})
		}
	}
	return findings
}
