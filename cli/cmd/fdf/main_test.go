package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMinimalBundle creates the valid-minimal bundle files under dir.
func writeMinimalBundle(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"INDEX.md": "---\nfdf_version: \"0.2\"\n---\n\n# B\n\n* [spec](https://github.com/GiteshDalal/fdf/blob/main/SPEC.md) - pin.\n",
		"LOG.md":   "# Bundle Update Log\n\n## 2026-07-06\n* **Initialization**: created.\n",
	}
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateHonorsEnvAndFlagRoots(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeMinimalBundle(t, filepath.Join(tmp, "documents", "features"))
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(tmp)

	t.Setenv("FDF_ROOT_DIR", "documents/features")
	var out bytes.Buffer
	if exit := runValidate(nil, &out); exit != 0 {
		t.Fatalf("env root: exit %d\n%s", exit, out.String())
	}
	if !strings.Contains(out.String(), "conformant") {
		t.Fatalf("env root output:\n%s", out.String())
	}

	t.Setenv("FDF_ROOT_DIR", "does/not/exist")
	out.Reset()
	if exit := runValidate([]string{"--root", "documents/features"}, &out); exit != 0 {
		t.Fatalf("--root must beat env: exit %d\n%s", exit, out.String())
	}
}
