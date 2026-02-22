package routes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoutesDoNotDependOnLegacyAppServiceConstructors(t *testing.T) {
	forbidden := []string{
		"NewOrganizationManagementService(",
		"NewOrganizationConfigService(",
		"NewServiceReadServiceFromStore(",
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read routes directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(".", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(data)
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				t.Fatalf("%s contains forbidden dependency token %q", path, token)
			}
		}
	}
}
