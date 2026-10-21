// Package fdf is the root of the module that builds the fdf command, the
// command-line tool for FDF (Feature Document Format). It embeds the agent
// skills and the format specs the command installs and vendors, as [Assets].
//
// Most people want the [fdf command] itself, which its own page documents in
// full. On macOS or Linux, install the latest release with:
//
//	curl -fsSL https://giteshdalal.com/fdf.sh | bash
//
// or, with Go 1.26 or later:
//
//	go install github.com/GiteshDalal/fdf/cli/cmd/fdf@latest
//
// FDF keeps each software feature as a Markdown and Gherkin document in the
// repository, with its design spec, implementation plan and acceptance tests
// beside it, and fdf validates the whole bundle so that the documents cannot
// silently drift from the code. The [project page] explains why that matters
// when AI agents write much of the code; the README covers the format, every
// command, and upgrading a bundle from an earlier version.
//
// [fdf command]: https://pkg.go.dev/github.com/GiteshDalal/fdf/cli/cmd/fdf
// [project page]: https://giteshdalal.com/projects/fdf/
package fdf
