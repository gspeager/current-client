//go:build !windows

package terminal

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

var lookPath = exec.LookPath

// descendants lists every process below pid, from ps, which is on every Unix.
func descendants(ctx context.Context, pid int) []int {
	output, err := exec.CommandContext(ctx, "ps", "-A", "-o", "pid=,ppid=").Output()
	if err != nil {
		return nil
	}
	children := map[int][]int{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		child, childErr := strconv.Atoi(fields[0])
		parent, parentErr := strconv.Atoi(fields[1])
		if childErr == nil && parentErr == nil {
			children[parent] = append(children[parent], child)
		}
	}
	var found []int
	var walk func(int)
	walk = func(at int) {
		for _, child := range children[at] {
			found = append(found, child)
			walk(child)
		}
	}
	walk(pid)
	return found
}

func killTree(pid int) {
	targets := append(descendants(context.Background(), pid), pid)
	for _, target := range targets {
		_ = syscall.Kill(target, syscall.SIGHUP)
	}
	for _, target := range targets {
		_ = syscall.Kill(target, syscall.SIGKILL)
	}
}
