package specver

import "testing"

func TestParseAcceptsMajorDotMinor(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Version
	}{
		{"0.2", Version{0, 2}},
		{"0.7", Version{0, 7}},
		{"1.0", Version{1, 0}},
		{"1.12", Version{1, 12}},
		{"10.3", Version{10, 3}},
	} {
		got, ok := Parse(tc.in)
		if !ok || got != tc.want {
			t.Errorf("Parse(%q) = %v, %v; want %v, true", tc.in, got, ok, tc.want)
		}
		if got.String() != tc.in {
			t.Errorf("Parse(%q).String() = %q; want it to round-trip", tc.in, got.String())
		}
	}
}

func TestParseRejectsAnythingElse(t *testing.T) {
	for _, in := range []string{"", "1", "1.", ".1", "1.0.0", "v1.0", " 1.0", "1.0 ", "1.x", "-1.0", "+1.0", "01.0", "1.00", "1.01", "0x1.0"} {
		if v, ok := Parse(in); ok {
			t.Errorf("Parse(%q) = %v, true; want it rejected", in, v)
		}
	}
}

func TestOrderIsNumericNotLexical(t *testing.T) {
	v := func(s string) Version {
		t.Helper()
		out, ok := Parse(s)
		if !ok {
			t.Fatalf("Parse(%q) failed", s)
		}
		return out
	}
	if !v("0.7").Less(v("1.0")) || !v("0.2").Less(v("0.10")) || !v("1.2").Less(v("1.10")) {
		t.Fatal("versions compare by number: 0.7 < 1.0, 0.2 < 0.10, 1.2 < 1.10")
	}
	if v("1.0").Less(v("1.0")) || !v("1.0").AtLeast(v("1.0")) || !v("1.0").AtLeast(v("0.7")) || v("0.7").AtLeast(v("1.0")) {
		t.Fatal("AtLeast is Less's complement: 1.0 >= 1.0, 1.0 >= 0.7, 0.7 < 1.0")
	}
	list := []string{"1.10", "0.10", "1.0", "0.2", "1.2", "0.7"}
	Sort(list)
	want := []string{"0.2", "0.7", "0.10", "1.0", "1.2", "1.10"}
	for i := range want {
		if list[i] != want[i] {
			t.Fatalf("Sort = %v; want %v", list, want)
		}
	}
}
