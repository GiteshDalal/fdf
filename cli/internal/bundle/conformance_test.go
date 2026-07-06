package bundle

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConformanceFixtures(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("..", "..", "..", "testdata", "*"))
	if err != nil || len(fixtures) == 0 {
		t.Fatalf("no fixtures found: %v", err)
	}
	for _, dir := range fixtures {
		dir := dir
		t.Run(filepath.Base(dir), func(t *testing.T) {
			bundleRoot := filepath.Join(dir, "bundle")
			repoRoot := ""
			if fi, err := os.Stat(filepath.Join(dir, "repo")); err == nil && fi.IsDir() {
				repoRoot = filepath.Join(dir, "repo")
				bundleRoot = filepath.Join(repoRoot, "docs", "features")
			}
			raw, err := os.ReadFile(filepath.Join(dir, "expect.txt"))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			exit := Validate(bundleRoot, Options{RepoRoot: repoRoot, Out: &out})
			for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
				switch {
				case strings.HasPrefix(line, "exit: "):
					want := strings.TrimSpace(strings.TrimPrefix(line, "exit: "))
					got := "0"
					if exit != 0 {
						got = "1"
					}
					if got != want {
						t.Fatalf("exit %s, want %s\n%s", got, want, out.String())
					}
				case strings.HasPrefix(line, "contains: "):
					sub := strings.TrimPrefix(line, "contains: ")
					if !strings.Contains(out.String(), sub) {
						t.Fatalf("output missing %q\n%s", sub, out.String())
					}
				}
			}
		})
	}
}
