package graph

import "sort"

type Commit struct {
	SHA        string
	ParentSHAs []string
}

type Node struct {
	SHA  string
	Lane int

	// ParentLanes[i] is the lane parent i continues in; [0] is always Lane.
	ParentLanes []int
	// ConvergingLanes end at this row, joining this node.
	ConvergingLanes []int
	// PassThrough lanes run straight past this row.
	PassThrough []int
}

// Layout expects git log's newest-first order. Each lane waits for a SHA; a
// commit takes the lane waiting for it, or the lowest free lane.
func Layout(commits []Commit) []Node {
	nodes := make([]Node, len(commits))
	lanes := &laneTracker{}

	for i, c := range commits {
		before := lanes.active()

		lane, converging := lanes.resolve(c.SHA)
		parentLanes := lanes.assignParents(lane, c.ParentSHAs)

		after := lanes.active()
		sort.Ints(converging)

		nodes[i] = Node{
			SHA:             c.SHA,
			Lane:            lane,
			ParentLanes:     parentLanes,
			ConvergingLanes: converging,
			PassThrough:     passThroughLanes(before, after, lane, parentLanes),
		}
	}

	return nodes
}

// waitingFor[lane] is the SHA that lane expects next; "" means free.
type laneTracker struct {
	waitingFor []string
}

func (t *laneTracker) active() map[int]bool {
	active := make(map[int]bool, len(t.waitingFor))
	for i, sha := range t.waitingFor {
		if sha != "" {
			active[i] = true
		}
	}
	return active
}

func (t *laneTracker) laneWaitingFor(sha string) int {
	for i, waiting := range t.waitingFor {
		if waiting == sha {
			return i
		}
	}
	return -1
}

func (t *laneTracker) freeLane() int {
	for i, sha := range t.waitingFor {
		if sha == "" {
			return i
		}
	}
	t.waitingFor = append(t.waitingFor, "")
	return len(t.waitingFor) - 1
}

// resolve keeps the first lane waiting for sha and frees the others, which
// converge here.
func (t *laneTracker) resolve(sha string) (lane int, converging []int) {
	lane = -1
	for i, waiting := range t.waitingFor {
		if waiting != sha {
			continue
		}
		if lane == -1 {
			lane = i
			continue
		}
		converging = append(converging, i)
		t.waitingFor[i] = ""
	}
	if lane == -1 {
		lane = t.freeLane()
	}
	return lane, converging
}

// assignParents continues lane with the first parent and gives each merge
// parent its own lane.
func (t *laneTracker) assignParents(lane int, parentSHAs []string) []int {
	parentLanes := make([]int, len(parentSHAs))
	if len(parentSHAs) == 0 {
		t.waitingFor[lane] = ""
		return parentLanes
	}

	t.waitingFor[lane] = parentSHAs[0]
	parentLanes[0] = lane

	for i := 1; i < len(parentSHAs); i++ {
		parentSHA := parentSHAs[i]
		parentLane := t.laneWaitingFor(parentSHA)
		if parentLane == -1 {
			parentLane = t.freeLane()
			t.waitingFor[parentLane] = parentSHA
		}
		parentLanes[i] = parentLane
	}
	return parentLanes
}

func passThroughLanes(before, after map[int]bool, ownLane int, parentLanes []int) []int {
	var passThrough []int
	for lane := range before {
		if lane == ownLane || !after[lane] || isParentLane(lane, parentLanes) {
			continue
		}
		passThrough = append(passThrough, lane)
	}
	sort.Ints(passThrough)
	return passThrough
}

func isParentLane(lane int, parentLanes []int) bool {
	for _, pl := range parentLanes {
		if pl == lane {
			return true
		}
	}
	return false
}
