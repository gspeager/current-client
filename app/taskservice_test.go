package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestARunNeedsApprovalAndAgainAfterAChange(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	marker := filepath.Join(repo, "ran")
	makefile := filepath.Join(repo, "Makefile")
	if err := os.WriteFile(makefile, []byte("go:\n\ttouch "+marker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var s TaskService

	list, err := s.List(repo)
	if err != nil || len(list.Tasks) != 1 || list.Tasks[0].Approved {
		t.Fatalf("first list = %+v, %v", list, err)
	}
	if err := s.Run(repo, "make:go", "r1"); !errors.Is(err, errTaskNotApproved) {
		t.Fatalf("an unapproved run: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("an unapproved command ran")
	}

	if err := s.Approve(repo, "make:go"); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); !list.Tasks[0].Approved {
		t.Fatalf("after approving = %+v", list)
	}
	if err := s.Run(repo, "make:go", "r2"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the approved command didn't run")
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := os.WriteFile(makefile, []byte("go:\n\techo changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); list.Tasks[0].Approved {
		t.Fatal("a changed recipe should need approving again")
	}
	if err := s.Run(repo, "make:go", "r1"); !errors.Is(err, errTaskNotApproved) {
		t.Fatalf("after a change: %v", err)
	}
}

func TestChecklistAndCustomCommands(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	var s TaskService
	if err := s.AddCustom(repo, "deploy", "echo deploy"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustom(repo, "deploy", "echo again"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetChecklist(repo, "commit", []string{"custom:deploy"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetChecklist(repo, "push", []string{"make:x"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(repo)
	if err != nil || len(list.Tasks) != 1 || list.Tasks[0].Command != "echo again" || !list.Tasks[0].Custom {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if len(list.BeforeCommit) != 1 || list.BeforeCommit[0] != "custom:deploy" || list.BeforePush[0] != "make:x" {
		t.Fatalf("checklist = %+v", list)
	}
	if err := s.RemoveCustom(repo, "deploy"); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); len(list.Tasks) != 0 {
		t.Fatalf("after removal = %+v", list)
	}
}
