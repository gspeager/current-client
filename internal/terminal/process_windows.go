//go:build windows

package terminal

import (
	"context"
	"os/exec"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

var lookPath = exec.LookPath

func descendants(_ context.Context, pid int) []int {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snapshot)
	children := map[int][]int{}
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		children[int(entry.ParentProcessID)] = append(children[int(entry.ParentProcessID)], int(entry.ProcessID))
	}
	var found []int
	var walk func(int)
	walk = func(at int) {
		for _, child := range children[at] {
			if child == pid {
				continue
			}
			found = append(found, child)
			walk(child)
		}
	}
	walk(pid)
	return found
}

func killTree(pid int) {
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}
