package risk

import (
	_ "embed"
	"math"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
)

//go:embed gitleaks.toml
var gitleaksRules string

// generic-api-key matches anything that looks random beside a word like "key",
// which is too noisy for a warning shown on every commit.
var skippedRules = []string{"generic-api-key"}

type allowlist struct {
	regexes   []*regexp.Regexp
	stopwords []string
	paths     []*regexp.Regexp
	// The match's whole line rather than the secret, when the rule says so.
	onLine bool
}

type secretRule struct {
	id          string
	regex       *regexp.Regexp
	path        *regexp.Regexp
	secretGroup int
	entropy     float64
	keywords    []string
	allowlists  []allowlist
}

type secretRules struct {
	rules       []secretRule
	ignoredPath []*regexp.Regexp
}

func compileAll(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, pattern := range patterns {
		if regex, err := regexp.Compile(pattern); err == nil {
			compiled = append(compiled, regex)
		}
	}
	return compiled
}

var loadSecretRules = sync.OnceValue(func() secretRules {
	var file struct {
		Allowlist struct {
			Paths []string `toml:"paths"`
		} `toml:"allowlist"`
		Rules []struct {
			ID          string   `toml:"id"`
			Regex       string   `toml:"regex"`
			Path        string   `toml:"path"`
			SecretGroup int      `toml:"secretGroup"`
			Entropy     float64  `toml:"entropy"`
			Keywords    []string `toml:"keywords"`
			Allowlists  []struct {
				Regexes     []string `toml:"regexes"`
				Stopwords   []string `toml:"stopwords"`
				Paths       []string `toml:"paths"`
				RegexTarget string   `toml:"regexTarget"`
			} `toml:"allowlists"`
		} `toml:"rules"`
	}
	if err := toml.Unmarshal([]byte(gitleaksRules), &file); err != nil {
		panic("core/risk: gitleaks.toml doesn't parse: " + err.Error())
	}
	loaded := secretRules{ignoredPath: compileAll(file.Allowlist.Paths)}
	for _, rule := range file.Rules {
		if rule.Regex == "" || slices.Contains(skippedRules, rule.ID) {
			continue
		}
		regex, err := regexp.Compile(rule.Regex)
		if err != nil {
			continue
		}
		compiled := secretRule{id: rule.ID, regex: regex, secretGroup: rule.SecretGroup, entropy: rule.Entropy}
		for _, keyword := range rule.Keywords {
			compiled.keywords = append(compiled.keywords, strings.ToLower(keyword))
		}
		if rule.Path != "" {
			compiled.path, _ = regexp.Compile(rule.Path)
		}
		for _, list := range rule.Allowlists {
			compiled.allowlists = append(compiled.allowlists, allowlist{
				regexes:   compileAll(list.Regexes),
				stopwords: list.Stopwords,
				paths:     compileAll(list.Paths),
				onLine:    list.RegexTarget == "line",
			})
		}
		loaded.rules = append(loaded.rules, compiled)
	}
	return loaded
})

func (r secretRules) pathIgnored(path string) bool {
	if fixturePath.MatchString(path) {
		return true
	}
	return slices.ContainsFunc(r.ignoredPath, func(regex *regexp.Regexp) bool { return regex.MatchString(path) })
}

// Test data holds real-looking keys on purpose.
var fixturePath = regexp.MustCompile(`(?i)(^|/)(testdata|fixtures?|__fixtures__|__snapshots__)/`)

// scanLine returns each match on a line as its rule and masked value.
func (r secretRules) scanLine(path, line string) []secretMatch {
	lower := strings.ToLower(line)
	var found []secretMatch
	for i := range r.rules {
		rule := &r.rules[i]
		if len(rule.keywords) > 0 && !slices.ContainsFunc(rule.keywords, func(keyword string) bool { return strings.Contains(lower, keyword) }) {
			continue
		}
		if rule.path != nil && !rule.path.MatchString(path) {
			continue
		}
		groups := rule.regex.FindStringSubmatch(line)
		if groups == nil {
			continue
		}
		secret := secretOf(groups, rule.secretGroup)
		if rule.entropy > 0 && entropy(secret) < rule.entropy {
			continue
		}
		if rule.allowed(path, line, secret) {
			continue
		}
		found = append(found, secretMatch{rule.id, mask(secret)})
	}
	return found
}

type secretMatch struct {
	rule   string
	masked string
}

// With no group named, a rule's secret is its first group, as in gitleaks.
func secretOf(groups []string, group int) string {
	switch {
	case group > 0 && group < len(groups):
		return groups[group]
	case group == 0 && len(groups) > 1 && groups[1] != "":
		return groups[1]
	}
	return groups[0]
}

func (rule *secretRule) allowed(path, line, secret string) bool {
	return slices.ContainsFunc(rule.allowlists, func(list allowlist) bool {
		target := secret
		if list.onLine {
			target = line
		}
		lower := strings.ToLower(secret)
		return slices.ContainsFunc(list.paths, func(regex *regexp.Regexp) bool { return regex.MatchString(path) }) ||
			slices.ContainsFunc(list.regexes, func(regex *regexp.Regexp) bool { return regex.MatchString(target) }) ||
			slices.ContainsFunc(list.stopwords, func(word string) bool { return strings.Contains(lower, word) })
	})
}

func entropy(text string) float64 {
	counts := map[rune]float64{}
	for _, char := range text {
		counts[char]++
	}
	length := float64(utf8.RuneCountInString(text))
	total := 0.0
	for _, count := range counts {
		share := count / length
		total -= share * math.Log2(share)
	}
	return total
}

// mask keeps at most the first four characters, and none of a short value.
func mask(secret string) string {
	if utf8.RuneCountInString(secret) < 12 {
		return "…"
	}
	return string([]rune(secret)[:4]) + "…"
}
