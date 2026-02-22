package services

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAppLayerHasNoDirectSQLCDependency(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve caller path")
	}
	appDir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))

	err := filepath.WalkDir(appDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "github.com/fr0stylo/ddash/internal/db/queries") {
			return fmt.Errorf("app layer must not import sqlc package, found in %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan app layer: %v", err)
	}
}
