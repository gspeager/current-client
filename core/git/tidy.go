package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

const (
	TidyKeep   = "keep"
	TidyReword = "reword"
	TidySquash = "squash"
	TidyFixup  = "fixup"
	TidyDrop   = "drop"
)

const tidyFallbackCommits = 20

type TidyCommit struct {
	SHA     string
	Subject string
	Message string
	// The action a fixup! or squash! commit starts with, beside the commit it names.
	Suggested string
}

// TidyPlan is the commits Tidy up history offers, oldest first. Base is what
// they sit on, "" when the range reaches the root.
type TidyPlan struct {
	Head     string
	Base     string
	BaseName string
	Commits  []TidyCommit
	Dirty    bool
	// A merge commit in the range: a rewrite would flatten it, so it's refused.
	HasMerges bool
}

// TidyStep is a commit in its new place. Message is only read for TidyReword.
type TidyStep struct {
	SHA     string
	Action  string
	Message string
}

// PlanTidy offers the branch's own commits: those since its upstream when it has
// unpushed work, otherwise since the default branch, otherwise the last few.
func PlanTidy(ctx context.Context, repoPath string) (TidyPlan, error) {
	head, err := ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return TidyPlan{}, err
	}
	plan := TidyPlan{Head: head}
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return plan, err
	}
	var candidates []string
	def := defaultBranch(list.branches, originHead(ctx, repoPath))
	for _, b := range list.branches {
		if b.Current && b.Upstream != "" && !list.goneUpstream[b.Name] && b.Ahead > 0 {
			candidates = append(candidates, b.Upstream)
		}
	}
	if def != "" {
		candidates = append(candidates, def)
	}
	for _, candidate := range candidates {
		base, err := MergeBase(ctx, repoPath, head, candidate)
		if err == nil && base != "" && base != head {
			plan.Base, plan.BaseName = base, candidate
			break
		}
	}
	if plan.Base == "" {
		count, err := CountCommits(ctx, repoPath, "HEAD")
		if err != nil {
			return plan, err
		}
		if count > tidyFallbackCommits {
			plan.Base = strings.TrimSpace(mustRev(ctx, repoPath, fmt.Sprintf("HEAD~%d", tidyFallbackCommits)))
			plan.BaseName = plan.Base[:min(7, len(plan.Base))]
		}
	}
	if err := fillTidyCommits(ctx, repoPath, &plan); err != nil {
		return plan, err
	}
	plan.Dirty, err = rebaseBlockedByChanges(ctx, repoPath)
	return plan, err
}

func mustRev(ctx context.Context, repoPath, rev string) string {
	result, err := runResult(ctx, repoPath, "rev-parse", "--verify", rev)
	if err != nil {
		return ""
	}
	return result.Stdout
}

func tidyRange(plan TidyPlan) string {
	if plan.Base == "" {
		return plan.Head
	}
	return plan.Base + ".." + plan.Head
}

func fillTidyCommits(ctx context.Context, repoPath string, plan *TidyPlan) error {
	result, err := runResult(ctx, repoPath, "log", "--reverse", "-z", "--format=%H%x1f%P%x1f%s%x1f%B", tidyRange(*plan))
	if err != nil {
		return err
	}
	for _, record := range strings.Split(result.Stdout, "\x00") {
		fields := strings.SplitN(record, "\x1f", 4)
		if len(fields) != 4 {
			continue
		}
		if len(strings.Fields(fields[1])) > 1 {
			plan.HasMerges = true
		}
		plan.Commits = append(plan.Commits, TidyCommit{
			SHA:     strings.TrimSpace(fields[0]),
			Subject: fields[2],
			Message: strings.TrimSpace(fields[3]),
		})
	}
	plan.Commits = arrangeAutosquash(plan.Commits)
	return nil
}

// arrangeAutosquash starts fixup! and squash! commits beside what they name, as
// git rebase --autosquash would, ready to be changed.
func arrangeAutosquash(commits []TidyCommit) []TidyCommit {
	arranged := make([]TidyCommit, 0, len(commits))
	for _, commit := range commits {
		target, action := "", ""
		if rest, ok := strings.CutPrefix(commit.Subject, "fixup! "); ok {
			target, action = rest, TidyFixup
		} else if rest, ok := strings.CutPrefix(commit.Subject, "squash! "); ok {
			target, action = rest, TidySquash
		}
		at := -1
		if target != "" {
			for i, candidate := range arranged {
				if candidate.Subject == target {
					at = i
				}
			}
		}
		if at == -1 {
			arranged = append(arranged, commit)
			continue
		}
		commit.Suggested = action
		for at+1 < len(arranged) && arranged[at+1].Suggested != "" {
			at++
		}
		arranged = append(arranged[:at+1], append([]TidyCommit{commit}, arranged[at+1:]...)...)
	}
	return arranged
}

func rebaseBlockedByChanges(ctx context.Context, repoPath string) (bool, error) {
	statuses, err := GetStatus(ctx, repoPath)
	if err != nil {
		return false, err
	}
	for _, status := range statuses {
		if status.WorktreeStatus != '?' {
			return configValue(ctx, repoPath, "--type=bool", "rebase.autostash") != "true", nil
		}
	}
	return false, nil
}

// validateTidy checks steps are the plan's commits, each once, with a commit to
// fold into before any squash or fixup, and something left at the end.
func validateTidy(plan TidyPlan, steps []TidyStep) error {
	known := map[string]bool{}
	for _, commit := range plan.Commits {
		known[commit.SHA] = true
	}
	if len(steps) != len(known) {
		return fmt.Errorf("the steps don't match the branch's commits")
	}
	seen, kept := map[string]bool{}, 0
	for _, step := range steps {
		if !known[step.SHA] || seen[step.SHA] {
			return fmt.Errorf("the steps don't match the branch's commits")
		}
		seen[step.SHA] = true
		switch step.Action {
		case TidyKeep, TidyReword:
			kept++
		case TidySquash, TidyFixup:
			if kept == 0 {
				return fmt.Errorf("the first commit kept has nothing to fold into")
			}
		case TidyDrop:
		default:
			return fmt.Errorf("unknown action: %s", step.Action)
		}
		if step.Action == TidyReword && strings.TrimSpace(step.Message) == "" {
			return fmt.Errorf("a reworded commit needs a message")
		}
	}
	if kept == 0 {
		return fmt.Errorf("at least one commit has to stay")
	}
	return nil
}

// TidyPreview says what running steps would do. Outcomes are in steps' order,
// with dropped commits left out; Predicted is false on Git before 2.40.
type TidyPreview struct {
	Result    []TidyResultCommit
	Replay    []ReplayStep
	Predicted bool
	ForcePush bool
	Upstream  string
}

type TidyResultCommit struct {
	Subject string
	Folded  int
}

func PreviewTidy(ctx context.Context, repoPath string, steps []TidyStep) (TidyPreview, error) {
	plan, err := PlanTidy(ctx, repoPath)
	if err != nil {
		return TidyPreview{}, err
	}
	if err := validateTidy(plan, steps); err != nil {
		return TidyPreview{}, err
	}
	preview := TidyPreview{Predicted: true}
	messages := map[string]string{}
	for _, commit := range plan.Commits {
		messages[commit.SHA] = commit.Message
	}
	var picked []string
	for _, step := range steps {
		if step.Action == TidyDrop {
			continue
		}
		picked = append(picked, step.SHA)
		switch step.Action {
		case TidyKeep, TidyReword:
			message := messages[step.SHA]
			if step.Action == TidyReword {
				message = step.Message
			}
			subject, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
			preview.Result = append(preview.Result, TidyResultCommit{Subject: subject})
		default:
			preview.Result[len(preview.Result)-1].Folded++
		}
	}
	onto := plan.Base
	if onto == "" {
		preview.Predicted = false
	} else if replay, err := ReplayCommits(ctx, repoPath, onto, picked); err != nil {
		preview.Predicted = false
	} else {
		preview.Replay = replay
	}
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return preview, err
	}
	for _, b := range list.branches {
		if b.Current && b.Upstream != "" && !list.goneUpstream[b.Name] {
			preview.Upstream = b.Upstream
		}
	}
	if preview.Upstream != "" && tidyChanges(plan, steps) {
		preview.ForcePush = firstChangedOnUpstream(ctx, repoPath, plan, steps, preview.Upstream)
	}
	return preview, nil
}

func tidyChanges(plan TidyPlan, steps []TidyStep) bool {
	for i, step := range steps {
		if step.Action != TidyKeep || step.SHA != plan.Commits[i].SHA {
			return true
		}
	}
	return false
}

// A rewrite keeps commits before the first change, so only what follows it counts.
func firstChangedOnUpstream(ctx context.Context, repoPath string, plan TidyPlan, steps []TidyStep, upstream string) bool {
	for i, step := range steps {
		if step.Action != TidyKeep || step.SHA != plan.Commits[i].SHA {
			return isAncestor(ctx, repoPath, plan.Commits[i].SHA, upstream)
		}
	}
	return false
}

// tidyTodo writes the rebase's todo list. Messages are applied by the list itself,
// from files in the Git directory, and never through an editor: a stop on a
// conflict can come before them, and Continue, possibly after a restart, has no
// editor to answer. Every commit's line is present, so rebase.missingCommitsCheck
// has nothing to object to.
func tidyTodo(plan TidyPlan, steps []TidyStep, messageDir string) (string, map[string]string, error) {
	messages := map[string]string{}
	for _, commit := range plan.Commits {
		messages[commit.SHA] = commit.Message
	}
	var todo strings.Builder
	files := map[string]string{}
	flush := func(group []TidyStep) {
		if len(group) == 0 {
			return
		}
		message, custom := messages[group[0].SHA], group[0].Action == TidyReword
		if custom {
			message = group[0].Message
		}
		for _, step := range group[1:] {
			if step.Action == TidySquash {
				message += "\n\n" + messages[step.SHA]
				custom = true
			}
		}
		if custom {
			name := filepath.Join(messageDir, group[0].SHA+".msg")
			files[name] = message + "\n"
			fmt.Fprintf(&todo, "exec git commit --amend --no-verify --allow-empty --cleanup=strip -F %s\n", shellQuote(filepath.ToSlash(name)))
		}
	}
	var group []TidyStep
	for _, step := range steps {
		switch step.Action {
		case TidyDrop:
			fmt.Fprintf(&todo, "drop %s\n", step.SHA)
		case TidySquash, TidyFixup:
			fmt.Fprintf(&todo, "fixup %s\n", step.SHA)
			group = append(group, step)
		default:
			flush(group)
			fmt.Fprintf(&todo, "pick %s\n", step.SHA)
			group = []TidyStep{step}
		}
	}
	flush(group)
	return todo.String(), files, nil
}

func shellQuote(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

// RunTidy rewrites the branch as steps say. A conflict stops it like any rebase,
// with Continue and Abort, and Undo takes a finished one back.
func RunTidy(ctx context.Context, repoPath, expectedHead string, steps []TidyStep) error {
	plan, err := PlanTidy(ctx, repoPath)
	if err != nil {
		return err
	}
	if plan.Head != expectedHead {
		return fmt.Errorf("the branch has moved since this was opened. Open it again")
	}
	if plan.HasMerges {
		return fmt.Errorf("these commits include a merge, which a rewrite would flatten")
	}
	if plan.Dirty {
		return fmt.Errorf("commit or stash the changes in the working tree first")
	}
	if err := validateTidy(plan, steps); err != nil {
		return err
	}
	gitDir, err := GitDir(ctx, repoPath)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(repoPath, gitDir)
	}
	messageDir := filepath.Join(gitDir, "current-client-tidy")
	if err := os.RemoveAll(messageDir); err != nil {
		return err
	}
	if err := os.MkdirAll(messageDir, 0o755); err != nil {
		return err
	}
	todo, files, err := tidyTodo(plan, steps, messageDir)
	if err != nil {
		return err
	}
	for name, content := range files {
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			return err
		}
	}
	todoFile := filepath.Join(messageDir, "todo")
	if err := os.WriteFile(todoFile, []byte(todo), 0o644); err != nil {
		return err
	}
	args := []string{"rebase", "-i", "--empty=keep", "--no-autosquash"}
	if plan.Base == "" {
		args = append(args, "--root")
	} else {
		args = append(args, plan.Base)
	}
	_, err = runStoppable(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: args,
		Env: []string{
			"GIT_SEQUENCE_EDITOR=cp " + shellQuote(filepath.ToSlash(todoFile)),
			"GIT_EDITOR=true",
		},
	})
	return err
}

// FixupInto commits what's staged as a fixup of sha, for Tidy up history to fold in.
func FixupInto(ctx context.Context, repoPath, sha string) error {
	_, err := runResult(ctx, repoPath, "commit", "--no-edit", "--fixup="+sha)
	return err
}
