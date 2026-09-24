package bundle

import (
	"reflect"
	"testing"
)

// An entry's indented continuation lines join it, CRLF line ends included; an
// unindented line, a nested item, a blank line or a heading ends it.
func TestLogicalLinesJoinsIndentedContinuations(t *testing.T) {
	body := "## venues/opening-hours\r\n" +
		"- Venue owner sets\r\n" +
		"  opening hours — `go test`\r\n" +
		"flush left\r\n" +
		"- next\r\n" +
		"  - nested\r\n" +
		"\r\n" +
		"  after a blank\r\n"
	want := []string{
		"## venues/opening-hours",
		"- Venue owner sets opening hours — `go test`",
		"flush left",
		"- next",
		"  - nested",
		"",
		"  after a blank",
		"",
	}
	if got := LogicalLines(body); !reflect.DeepEqual(got, want) {
		t.Errorf("LogicalLines:\n got %q\nwant %q", got, want)
	}
}
