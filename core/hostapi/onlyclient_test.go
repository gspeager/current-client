package hostapi

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every host API call goes through this package: no other non-test file in
// core/ or app/ may import net/http, except app/csp.go, which only sets the
// page's Content-Security-Policy on the asset server's responses.
func TestNoOtherHTTPClientInCoreOrApp(t *testing.T) {
	root := moduleRoot(t)
	allowed := map[string]bool{filepath.Join("app", "csp.go"): true}
	var offenders []string
	for _, dir := range []string{"core", "app"} {
		filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			if strings.HasPrefix(path, filepath.Join(root, "core", "hostapi")+string(filepath.Separator)) {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parsing %s: %v", path, err)
			}
			for _, imp := range file.Imports {
				relative := strings.TrimPrefix(path, root+string(filepath.Separator))
				if strings.Trim(imp.Path.Value, `"`) == "net/http" && !allowed[relative] {
					offenders = append(offenders, relative)
				}
			}
			return nil
		})
	}
	if len(offenders) > 0 {
		t.Fatalf("net/http imported outside core/hostapi: %v", offenders)
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test")
		}
		dir = parent
	}
}
