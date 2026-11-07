package app

import (
	"context"
	"errors"
	"slices"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/config"
)

type StackLayerInfo struct {
	Branch  string `json:"branch"`
	Parent  string `json:"parent"`
	Depth   int    `json:"depth"`
	Commits int    `json:"commits"`
	Missing bool   `json:"missing"`
	Merged  bool   `json:"merged"`
	Behind  bool   `json:"behind"`
}

type StackInfo struct {
	Root         string           `json:"root"`
	Base         string           `json:"base"`
	Layers       []StackLayerInfo `json:"layers"`
	NeedsRestack bool             `json:"needsRestack"`
}

type StackListInfo struct {
	Stacks  []StackInfo `json:"stacks"`
	CanUndo bool        `json:"canUndo"`
}

type RestackPreviewLayerInfo struct {
	Branch        string                    `json:"branch"`
	Onto          string                    `json:"onto"`
	Commits       []RebasePreviewCommitInfo `json:"commits"`
	DroppedMerges int                       `json:"droppedMerges"`
	Missing       bool                      `json:"missing"`
	Merged        bool                      `json:"merged"`
	Moves         bool                      `json:"moves"`
}

type RestackPreviewInfo struct {
	Layers    []RestackPreviewLayerInfo `json:"layers"`
	Predicted bool                      `json:"predicted"`
	Dirty     bool                      `json:"dirty"`
	Blocked   string                    `json:"blocked"`
}

type RestackResultInfo struct {
	Stopped string   `json:"stopped"`
	Left    []string `json:"left"`
	Moved   int      `json:"moved"`
}

type StackService struct{}

func linksOf(repoPath string) []git.StackLink {
	return mapSlice(configOrDefault().Stacks[repoPath], func(link config.StackLink) git.StackLink { return git.StackLink(link) })
}

func saveLinks(repoPath string, links []git.StackLink) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.Stacks == nil {
			cfg.Stacks = map[string][]config.StackLink{}
		}
		cfg.Stacks[repoPath] = mapSlice(links, func(link git.StackLink) config.StackLink { return config.StackLink(link) })
	})
}

func renameInStacks(repoPath, oldName, newName string) error {
	links := linksOf(repoPath)
	if len(links) == 0 {
		return nil
	}
	for i := range links {
		if links[i].Branch == oldName {
			links[i].Branch = newName
		}
		if links[i].Parent == oldName {
			links[i].Parent = newName
		}
	}
	return saveLinks(repoPath, links)
}

// stackOf is the layers from root upwards, which is what one Restack or Push all covers.
func stackOf(links []git.StackLink, root string) []git.StackLink {
	in := map[string]bool{root: true}
	var stack []git.StackLink
	for _, link := range git.StackOrder(links) {
		if link.Branch == root || in[link.Parent] {
			in[link.Branch] = true
			stack = append(stack, link)
		}
	}
	return stack
}

func (s *StackService) List(repoPath string) (StackListInfo, error) {
	ctx := context.Background()
	list := StackListInfo{Stacks: []StackInfo{}}
	cfg := configOrDefault()
	_, list.CanUndo = cfg.StackUndo[repoPath]
	links := linksOf(repoPath)
	if len(links) == 0 {
		return list, nil
	}
	if state, err := git.DetectConflictState(ctx, repoPath); err == nil && state.Operation == git.ConflictNone {
		if refreshed, changed := git.RefreshStackLinks(ctx, repoPath, links); changed {
			links = refreshed
			if err := saveLinks(repoPath, links); err != nil {
				return list, err
			}
		}
	}
	layers, err := git.StackLayers(ctx, repoPath, links)
	if err != nil {
		return list, err
	}
	depth := map[string]int{}
	root := map[string]int{}
	for _, layer := range layers {
		at, stacked := root[layer.Parent]
		if !stacked {
			list.Stacks = append(list.Stacks, StackInfo{Root: layer.Branch, Base: layer.Parent, Layers: []StackLayerInfo{}})
			at = len(list.Stacks) - 1
		} else {
			depth[layer.Branch] = depth[layer.Parent] + 1
		}
		root[layer.Branch] = at
		stack := &list.Stacks[at]
		stack.Layers = append(stack.Layers, StackLayerInfo{
			Branch:  layer.Branch,
			Parent:  layer.Parent,
			Depth:   depth[layer.Branch],
			Commits: layer.Commits,
			Missing: layer.Missing,
			Merged:  layer.Merged,
			Behind:  layer.Behind,
		})
		stack.NeedsRestack = stack.NeedsRestack || layer.Behind || layer.Missing || layer.Merged
	}
	return list, nil
}

// AddLayer records branch as built on parent, where their histories meet.
func (s *StackService) AddLayer(repoPath, branch, parent string) error {
	links := linksOf(repoPath)
	parentOf := map[string]string{}
	for _, link := range links {
		parentOf[link.Branch] = link.Parent
	}
	for at, seen := parent, 0; at != ""; at, seen = parentOf[at], seen+1 {
		if at == branch || seen > len(links) {
			return errors.New("a branch can't be stacked on itself or on a branch above it")
		}
	}
	base, err := git.MergeBase(context.Background(), repoPath, parent, branch)
	if err != nil {
		return err
	}
	links = slices.DeleteFunc(links, func(link git.StackLink) bool { return link.Branch == branch })
	return saveLinks(repoPath, append(links, git.StackLink{Branch: branch, Parent: parent, ParentTip: base}))
}

func (s *StackService) NewLayer(repoPath, name, parent string) error {
	ctx := context.Background()
	if err := git.CreateBranchAt(ctx, repoPath, name, parent); err != nil {
		return err
	}
	if err := s.AddLayer(repoPath, name, parent); err != nil {
		return err
	}
	return git.CheckoutBranch(ctx, repoPath, name)
}

// RemoveLayer forgets a layer; what's above it is now built on what's below it.
func (s *StackService) RemoveLayer(repoPath, branch string) error {
	links := linksOf(repoPath)
	parent := ""
	for _, link := range links {
		if link.Branch == branch {
			parent = link.Parent
		}
	}
	links = slices.DeleteFunc(links, func(link git.StackLink) bool { return link.Branch == branch })
	for i := range links {
		if links[i].Parent == branch {
			links[i].Parent = parent
		}
	}
	return saveLinks(repoPath, links)
}

func (s *StackService) Preview(repoPath, root string) (RestackPreviewInfo, error) {
	preview, err := git.PreviewRestack(context.Background(), repoPath, stackOf(linksOf(repoPath), root))
	if err != nil {
		return RestackPreviewInfo{}, err
	}
	return RestackPreviewInfo{
		Layers: mapSlice(preview.Layers, func(layer git.RestackPreviewLayer) RestackPreviewLayerInfo {
			return RestackPreviewLayerInfo{
				Branch: layer.Branch,
				Onto:   layer.Onto,
				Commits: mapSlice(layer.Commits, func(commit git.PreviewCommit) RebasePreviewCommitInfo {
					return RebasePreviewCommitInfo{commit.SHA, commit.Subject, string(commit.Outcome), append([]string{}, commit.Files...)}
				}),
				DroppedMerges: layer.DroppedMerges,
				Missing:       layer.Missing,
				Merged:        layer.Merged,
				Moves:         layer.Moves,
			}
		}),
		Predicted: preview.Predicted,
		Dirty:     preview.Dirty,
		Blocked:   preview.Blocked,
	}, nil
}

func (s *StackService) Restack(repoPath, root string) (RestackResultInfo, error) {
	all := linksOf(repoPath)
	stack := stackOf(all, root)
	result, err := git.Restack(context.Background(), repoPath, stack)
	for _, move := range result.Moves {
		snapshotTip(repoPath, "restack", move.Branch, move.Before)
	}
	if len(result.Links) > 0 || len(result.Left) > 0 {
		kept := slices.DeleteFunc(slices.Clone(all), func(link git.StackLink) bool {
			return slices.ContainsFunc(stack, func(in git.StackLink) bool { return in.Branch == link.Branch })
		})
		if saveErr := saveLinks(repoPath, append(kept, result.Links...)); saveErr != nil && err == nil {
			err = saveErr
		}
	}
	if len(result.Moves) > 0 {
		if saveErr := saveUndo(repoPath, all, result); saveErr != nil && err == nil {
			err = saveErr
		}
	}
	return RestackResultInfo{Stopped: result.Stopped, Left: result.Left, Moved: len(result.Moves)}, err
}

// saveUndo keeps the state from before the first of a restack's attempts, since
// carrying on after a conflict is still the same restack to the user.
func saveUndo(repoPath string, links []git.StackLink, result git.RestackResult) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.StackUndo == nil {
			cfg.StackUndo = map[string]config.StackUndo{}
		}
		undo := config.StackUndo{
			Links:   mapSlice(links, func(link git.StackLink) config.StackLink { return config.StackLink(link) }),
			Moves:   mapSlice(result.Moves, func(move git.StackMove) config.StackMove { return config.StackMove(move) }),
			Stopped: result.Stopped != "",
		}
		if previous, ok := cfg.StackUndo[repoPath]; ok && previous.Stopped {
			undo.Links = previous.Links
			for _, move := range previous.Moves {
				at := slices.IndexFunc(undo.Moves, func(next config.StackMove) bool { return next.Branch == move.Branch })
				if at >= 0 {
					undo.Moves[at].Before = move.Before
				} else {
					undo.Moves = append(undo.Moves, move)
				}
			}
		}
		cfg.StackUndo[repoPath] = undo
	})
}

// UndoRestack is separate from Undo, which only moves HEAD. A refusal also ends
// the offer, as a branch that has moved on won't move back; Recover still has it.
func (s *StackService) UndoRestack(repoPath string) error {
	undo, ok := configOrDefault().StackUndo[repoPath]
	if !ok {
		return errors.New("there's no restack to undo")
	}
	moves := mapSlice(undo.Moves, func(move config.StackMove) git.StackMove { return git.StackMove(move) })
	// The layer a restack stopped on may have gained work since. Recover keeps it.
	for _, move := range moves {
		if move.After == "" {
			if tip, err := git.ResolveCommit(context.Background(), repoPath, "refs/heads/"+move.Branch); err == nil && tip != move.Before {
				snapshotTip(repoPath, "undo restack", move.Branch, tip)
			}
		}
	}
	err := git.UndoRestack(context.Background(), repoPath, moves)
	var movedOn *git.StackMovedError
	if err != nil && !errors.As(err, &movedOn) {
		return err
	}
	return errors.Join(err, updateConfig(func(cfg *config.Config) {
		if err == nil {
			cfg.Stacks[repoPath] = undo.Links
		}
		delete(cfg.StackUndo, repoPath)
	}))
}

func (s *StackService) PushAll(ctx context.Context, repoPath, root, remote string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	layers, err := git.StackLayers(ctx, repoPath, stackOf(linksOf(repoPath), root))
	if err != nil {
		return err
	}
	var branches []string
	for _, link := range layers {
		// Merged: a host may have deleted it, and this would bring it back.
		if link.Merged {
			continue
		}
		if tip, err := git.ResolveCommit(ctx, repoPath, "refs/heads/"+link.Branch); err == nil {
			branches = append(branches, link.Branch)
			// What the push overwrites on the remote, as of the last fetch.
			if theirs, err := git.ResolveCommit(ctx, repoPath, "refs/remotes/"+remote+"/"+link.Branch); err == nil && theirs != tip {
				snapshotTip(repoPath, "force-push", link.Branch, theirs)
			}
		}
	}
	if len(branches) == 0 {
		return errors.New("this stack has no branches to push")
	}
	return git.PushStack(ctx, repoPath, remote, branches)
}
