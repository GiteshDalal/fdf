package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/GiteshDalal/fdf/cli/internal/install"
)

// packageDoc is the package comment, which pkg.go.dev publishes as the
// command's documentation.
func packageDoc(t *testing.T) string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "doc.go", nil, parser.ParseComments|parser.PackageClauseOnly)
	if err != nil {
		t.Fatal(err)
	}
	if f.Doc == nil {
		t.Fatal("doc.go has no package comment")
	}
	return f.Doc.Text()
}

// The package comment lists the commands as the overview does, line for
// line, so a command added, renamed or reworded in helpTopics cannot leave
// the published documentation behind.
func TestThePackageDocListsEveryCommandAsTheOverviewDoes(t *testing.T) {
	var want strings.Builder
	for _, line := range strings.SplitAfter(commandIndex(), "\n") {
		if strings.TrimSpace(line) != "" {
			want.WriteString("\t")
		}
		want.WriteString(line)
	}
	if !strings.Contains(packageDoc(t), want.String()) {
		t.Errorf("doc.go's command list is not the overview's; paste this code block into it:\n%s", want.String())
	}
}

// Every skill fdf install places is named in the package comment.
func TestThePackageDocNamesEverySkill(t *testing.T) {
	doc := packageDoc(t)
	for _, name := range install.SkillNames() {
		if !strings.Contains(doc, name) {
			t.Errorf("doc.go does not name the %s skill", name)
		}
	}
}
