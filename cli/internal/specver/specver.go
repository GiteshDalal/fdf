// Package specver reads FDF specification versions. A version is
// MAJOR.MINOR, as a bundle pins it in fdf_version and as spec/<version>.md
// names it, and versions compare by number: 0.7 comes before 1.0, and 1.2
// before 1.10.
package specver

import (
	"sort"
	"strconv"
	"strings"
)

// Version is a specification version.
type Version struct{ Major, Minor int }

// Parse reads "MAJOR.MINOR". Anything else is not a version: an empty
// string, one part or three, a prefix such as "v", spaces, a sign, or a
// redundant leading zero ("01.0", "1.00").
func Parse(s string) (Version, bool) {
	major, minor, ok := strings.Cut(s, ".")
	if !ok {
		return Version{}, false
	}
	a, okA := number(major)
	b, okB := number(minor)
	if !okA || !okB {
		return Version{}, false
	}
	return Version{a, b}, true
}

// number reads a non-negative decimal written without a sign or a redundant
// leading zero.
func number(s string) (int, bool) {
	if s == "" || len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func (v Version) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor)
}

// Less reports whether v comes before o.
func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	return v.Minor < o.Minor
}

// AtLeast reports whether v is o or a later version.
func (v Version) AtLeast(o Version) bool { return !v.Less(o) }

// Known0x reports whether s is one of the versions FDF had before 1.0, 0.1 to
// 0.7: the pins fdf migrate upgrades from. Any other 0.x pin is a mistake, to
// be corrected, not a bundle to migrate.
func Known0x(s string) bool {
	v, ok := Parse(s)
	return ok && v.Major == 0 && v.Minor >= 1 && v.Minor <= 7
}

// Sort orders version strings oldest first. A string that is not a version
// sorts after every version, and such strings keep their lexical order.
func Sort(vs []string) {
	sort.SliceStable(vs, func(i, j int) bool {
		a, okA := Parse(vs[i])
		b, okB := Parse(vs[j])
		switch {
		case okA && okB:
			return a.Less(b)
		case okA != okB:
			return okA
		default:
			return vs[i] < vs[j]
		}
	})
}
