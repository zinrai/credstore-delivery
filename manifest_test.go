package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNameOutsideTheContractIsRejected(t *testing.T) {
	for _, name := range []string{"web.token", "web//token", "/web", "web/", "../web"} {
		if _, err := parseManifest(writeManifest(t, "- web/ok\n- '"+name+"'\n")); err == nil {
			t.Errorf("accepted %q", name)
		}
	}
}
