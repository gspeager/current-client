package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestFileChurn(t *testing.T) {
	dir := gittest.InitRepo(t)

	gittest.WriteFile(t, dir, "hot.go", "v1")
	gittest.WriteFile(t, dir, "cold.go", "v1")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-m", "initial")

	for i := 0; i < 3; i++ {
		gittest.CommitFile(t, dir, "hot.go", "v"+string(rune('2'+i)), "touch hot")
	}

	churn, err := ComputeFileChurn(context.Background(), dir, 365, 10)
	if err != nil {
		t.Fatalf("ComputeFileChurn: %v", err)
	}
	if len(churn) != 2 {
		t.Fatalf("got %d entries, want 2", len(churn))
	}
	if churn[0].Path != "hot.go" || churn[0].Changes != 4 {
		t.Errorf("top entry = %+v, want {hot.go 4}", churn[0])
	}
	if churn[1].Path != "cold.go" || churn[1].Changes != 1 {
		t.Errorf("second entry = %+v, want {cold.go 1}", churn[1])
	}
}

func TestFileChurnRespectsLimit(t *testing.T) {
	dir := gittest.InitRepo(t)

	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		gittest.WriteFile(t, dir, name, "v1")
	}
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-m", "initial")

	churn, err := ComputeFileChurn(context.Background(), dir, 365, 2)
	if err != nil {
		t.Fatalf("ComputeFileChurn: %v", err)
	}
	if len(churn) != 2 {
		t.Fatalf("got %d entries, want 2 (limit)", len(churn))
	}
}
