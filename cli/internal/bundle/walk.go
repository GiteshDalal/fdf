package bundle

// The validation walk. Every position comes from the layout package, so the
// validator and the commands agree on where each document goes: the closed
// root, features/ with its flat features and nested groups, groups nested
// in every register but releases/, and the directory beside a document.

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GiteshDalal/fdf/cli/internal/layout"
)

// walk visits every Markdown file of the bundle at rootAbs.
func (c *collection) walk(rootAbs string) {
	b := layout.New(os.DirFS(rootAbs))
	strays := map[string]bool{} // a path with no position, reported once
	filepath.WalkDir(rootAbs, func(p string, e os.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case e.IsDir():
			// A hidden directory (.git, .obsidian) holds a tool's state, not FDF's.
			if p != rootAbs && strings.HasPrefix(e.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		case !strings.HasSuffix(e.Name(), ".md") || strings.HasPrefix(e.Name(), "."):
			return nil // files that are not Markdown, and hidden files, are outside FDF
		}
		rel := filepath.ToSlash(relTo(rootAbs, p))
		pos := b.File(rel)
		switch pos.Kind {
		case layout.Readme:
			return nil
		case layout.Stray:
			if !strays[pos.Where] {
				strays[pos.Where] = true
				c.fail("%s: %s (F3)", pos.Where, pos.Problem)
			}
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			c.warn("%s: could not read file (%v)", rel, rerr)
			return nil
		}
		text := c.read(rel, raw)
		if pos.Kind == layout.Index || pos.Kind == layout.Log {
			c.reserved(rel, path.Base(rel), text)
			return nil
		}
		d, ok := c.document(rel, text)
		if !ok {
			return nil
		}
		switch pos.Kind {
		case layout.Reference, layout.Context:
			c.root(rel, rel, pos.Kind == layout.Context, d, text)
		case layout.Trail:
			if pos.Register == "features" || pos.Register == "changes" {
				c.trail(rel, pos.ID, pos.Role, d, text)
			} else {
				c.registerLog(rel, pos.Register, pos.ID, d, text)
			}
		case layout.Task:
			c.task(rel, pos.ID, path.Base(rel), d)
		case layout.Document:
			c.filed(rel, pos, d)
		}
		return nil
	})
}

// filed records a register's document by its register.
func (c *collection) filed(rel string, pos layout.Position, d doc) {
	// A task whose task directory has no document beside it reads as a
	// document in a group; say what is missing instead. One at a register's
	// root has no directory to be the task directory of anything.
	if d.docType == "Task" && (pos.Register == "features" || pos.Register == "changes") && taskFileRe.MatchString(path.Base(rel)) {
		if owner := path.Dir(pos.ID); owner == pos.Register {
			c.fail("%s: `type: Task` at the root of %s/ — a task sits in the task directory beside the feature, Change or Fix it belongs to (F3)", rel, owner)
		} else {
			c.fail("%s: `type: Task`, but %s.md does not exist — a task directory sits beside the feature, Change or Fix it belongs to (F3)", rel, owner)
		}
		return
	}
	switch pos.Register {
	case "features":
		c.feature(rel, pos.ID, d)
	case "changes":
		c.change(rel, pos.ID, d)
	case "practices":
		c.practice(rel, pos.ID, d)
	case "debts", "bugs":
		c.entry(rel, pos.Register, pos.ID, d)
	case "releases":
		c.release(rel, path.Base(pos.ID), d)
	}
}
