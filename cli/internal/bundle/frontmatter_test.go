package bundle

import (
	"reflect"
	"testing"
)

func TestSplitFrontmatter(t *testing.T) {
	block, ok, body := splitFrontmatter("---\ntype: Feature\n---\n\nBody\n")
	if !ok || block != "type: Feature\n" || body != "\nBody\n" {
		t.Fatalf("got %q %v %q", block, ok, body)
	}
	if _, ok, _ := splitFrontmatter("no frontmatter"); ok {
		t.Fatal("expected no block")
	}
	if _, ok, _ := splitFrontmatter("---\nunterminated: yes\n"); ok {
		t.Fatal("unterminated must not be ok")
	}
}

func TestParseFrontmatter(t *testing.T) {
	data, err := parseFrontmatter("type: Task\nstatus: done\ndepends-on: [01-a, 02-b]\nresource:\n  - path/one\n  - \"path/two\"\ntags: [x]\n")
	if err != nil {
		t.Fatal(err)
	}
	if data["type"] != "Task" || data["status"] != "done" {
		t.Fatalf("scalars: %v", data)
	}
	if got := asList(data["depends-on"]); !reflect.DeepEqual(got, []string{"01-a", "02-b"}) {
		t.Fatalf("inline list: %v", got)
	}
	if got := asList(data["resource"]); !reflect.DeepEqual(got, []string{"path/one", "path/two"}) {
		t.Fatalf("block list: %v", got)
	}
	if _, err := parseFrontmatter("just some prose"); err == nil {
		t.Fatal("expected parse error")
	}
}
