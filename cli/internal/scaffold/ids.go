package scaffold

// The names a command is given: what it files a new document as, and how it
// titles one.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/layout"
)

// Title is how fdf titles a document it scaffolds: its slug, capitalized,
// with spaces for dashes.
func Title(slug string) string {
	return strings.ToUpper(slug[:1]) + strings.ReplaceAll(slug[1:], "-", " ")
}

// NewID reads the name a command was given for a new document in register
// reg: [<group>/…]<slug>, or the document's full ID, which files it in the
// same place. It returns the full ID, or says why nothing can be filed there
// (layout's Place) and returns "".
func NewID(root, reg, name string, out io.Writer) string {
	id := reg + "/" + strings.TrimPrefix(name, reg+"/")
	if problem := layout.New(os.DirFS(root)).Place(id); problem != "" {
		fmt.Fprintln(out, "error: "+problem)
		return ""
	}
	return id
}

// WriteNew writes the new document id.md under root, and never over a file
// that is there. The file system decides, so on a disk that ignores case it
// also refuses a name that differs from an existing file's only in case,
// which Place, reading names exactly, lets through.
func WriteNew(root, id, text string) error {
	f, err := os.OpenFile(filepath.Join(root, filepath.FromSlash(id)+".md"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("a file is already at %s.md; on a disk that ignores case, its name may differ in case", id)
	}
	if err != nil {
		return err
	}
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// IsFeature reports whether id is the ID of a feature of the bundle at root:
// a document filed in features/ that exists, its name spelled exactly so: on
// a disk that ignores case, os.Stat takes a stray features/Refunds.md for
// features/refunds.
func IsFeature(root, id string) bool {
	b := layout.New(os.DirFS(root))
	pos := b.File(id + ".md")
	return pos.Kind == layout.Document && pos.Register == "features" && b.Exists(id+".md")
}

// FeatureHint is what a command adds when id names no feature: the full ID,
// when id is a feature's written the 0.7 way, without the features/ every
// 1.0 feature ID starts with, as the validator's F10 and F14 suggest it.
func FeatureHint(root, id string) string {
	if !strings.HasPrefix(id, "features/") && IsFeature(root, "features/"+id) {
		return " — did you mean features/" + id + "?"
	}
	return ""
}
