package graph

import "testing"

func TestLayoutLinearHistory(t *testing.T) {
	commits := []Commit{
		{SHA: "c3", ParentSHAs: []string{"c2"}},
		{SHA: "c2", ParentSHAs: []string{"c1"}},
		{SHA: "c1", ParentSHAs: nil},
	}
	nodes := Layout(commits)
	for i, n := range nodes {
		if n.Lane != 0 {
			t.Fatalf("node %d (%s): lane = %d, want 0", i, n.SHA, n.Lane)
		}
		if len(n.PassThrough) != 0 || len(n.ConvergingLanes) != 0 {
			t.Fatalf("node %d (%s): expected no pass-through or converging lanes, got %+v", i, n.SHA, n)
		}
	}
	if got := nodes[0].ParentLanes; len(got) != 1 || got[0] != 0 {
		t.Fatalf("c3 parent lanes = %v, want [0]", got)
	}
	if got := nodes[2].ParentLanes; len(got) != 0 {
		t.Fatalf("c1 (root) parent lanes = %v, want []", got)
	}
}

func TestLayoutForkAndMerge(t *testing.T) {
	//	*   merge
	//	|\
	//	| * feat1
	//	* | main2
	//	|/
	//	* base
	//	* root
	commits := []Commit{
		{SHA: "merge", ParentSHAs: []string{"main2", "feat1"}},
		{SHA: "feat1", ParentSHAs: []string{"base"}},
		{SHA: "main2", ParentSHAs: []string{"base"}},
		{SHA: "base", ParentSHAs: []string{"root"}},
		{SHA: "root", ParentSHAs: nil},
	}
	nodes := Layout(commits)

	byName := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		byName[n.SHA] = n
	}

	if lane := byName["merge"].Lane; lane != 0 {
		t.Fatalf("merge lane = %d, want 0", lane)
	}
	if got := byName["merge"].ParentLanes; len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("merge parent lanes = %v, want [0 1]", got)
	}
	if got := byName["merge"].PassThrough; len(got) != 0 {
		t.Fatalf("merge pass-through = %v, want [] (nothing existed before the first row)", got)
	}
	if lane := byName["main2"].Lane; lane != 0 {
		t.Fatalf("main2 lane = %d, want 0 (continues merge's first parent)", lane)
	}
	if got := byName["main2"].PassThrough; len(got) != 1 || got[0] != 1 {
		t.Fatalf("main2 pass-through = %v, want [1] (feat1's lane not yet reached)", got)
	}
	if lane := byName["feat1"].Lane; lane != 1 {
		t.Fatalf("feat1 lane = %d, want 1 (merge's second parent lane)", lane)
	}
	if got := byName["feat1"].PassThrough; len(got) != 1 || got[0] != 0 {
		t.Fatalf("feat1 pass-through = %v, want [0] (main2's lane not yet reached)", got)
	}
	if lane := byName["base"].Lane; lane != 0 {
		t.Fatalf("base lane = %d, want 0", lane)
	}
	if got := byName["base"].ParentLanes; len(got) != 1 || got[0] != 0 {
		t.Fatalf("base parent lanes = %v, want [0]", got)
	}
	if got := byName["base"].ConvergingLanes; len(got) != 1 || got[0] != 1 {
		t.Fatalf("base converging lanes = %v, want [1] (feat1's branch joins here)", got)
	}
	if lane := byName["root"].Lane; lane != 0 {
		t.Fatalf("root lane = %d, want 0", lane)
	}
}
