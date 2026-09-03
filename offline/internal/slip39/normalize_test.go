package slip39

import (
	"strings"
	"testing"
)

// The whole point: metal plates hold four letters per word, so recovery has to
// accept exactly that. This also pins the property the abbreviation relies on.
func TestWordlistPrefixesAreUnique(t *testing.T) {
	seen := make(map[string]string, len(wordlist))
	for _, w := range wordlist {
		if len(w) < PrefixLen {
			t.Fatalf("word %q is shorter than %d letters", w, PrefixLen)
		}
		p := w[:PrefixLen]
		if other, dup := seen[p]; dup {
			t.Fatalf("prefix %q is shared by %q and %q", p, other, w)
		}
		seen[p] = w
	}
	if len(seen) != 1024 {
		t.Fatalf("got %d prefixes, want 1024", len(seen))
	}
}

func TestNormalizeMnemonic(t *testing.T) {
	cases := []struct{ in, want, errContains string }{
		{in: "academic acid acrobat", want: "academic acid acrobat"},
		{in: "acad acid acro", want: "academic acid acrobat"},             // as engraved
		{in: "ACAD Acid acro", want: "academic acid acrobat"},             // case
		{in: "1. acad  2. acid   3. acro", want: "academic acid acrobat"}, // numbering
		{in: "acad, acid; acro.", want: "academic acid acrobat"},          // punctuation
		{in: "academ acid acro", want: "academic acid acrobat"},           // partial, still unambiguous
		{in: "aca acid acro", errContains: "príliš krátke"},               // 3 letters are ambiguous
		{in: "zzzz acid acro", errContains: "nie je slovo"},               // unknown
		{in: "acadxxxx acid", errContains: "malo byť"},                    // typo past the prefix
		{in: "   ", errContains: "žiadne slová"},
	}
	for _, c := range cases {
		got, err := NormalizeMnemonic(c.in)
		switch {
		case c.errContains == "" && err != nil:
			t.Errorf("NormalizeMnemonic(%q): unexpected error %v", c.in, err)
		case c.errContains == "" && got != c.want:
			t.Errorf("NormalizeMnemonic(%q) = %q, want %q", c.in, got, c.want)
		case c.errContains != "" && err == nil:
			t.Errorf("NormalizeMnemonic(%q) = %q, want an error mentioning %q", c.in, got, c.errContains)
		case c.errContains != "" && !strings.Contains(err.Error(), c.errContains):
			t.Errorf("NormalizeMnemonic(%q) error = %v, want it to mention %q", c.in, err, c.errContains)
		}
	}
}

// End to end: split a key, abbreviate every word to what fits on metal, and
// recover from that.
func TestRecoverFromEngravedAbbreviations(t *testing.T) {
	key := []byte("twenty-byte key!! ok")
	if len(key) != 20 {
		t.Fatalf("test key is %d bytes, want 20", len(key))
	}
	shares, err := Generate(key, 2, 3, nil)
	if err != nil {
		t.Fatal(err)
	}

	engrave := func(m string) string {
		words := strings.Fields(m)
		for i, w := range words {
			words[i] = w[:PrefixLen]
		}
		return strings.Join(words, " ")
	}
	lines := []string{engrave(shares[0]), engrave(shares[2])}
	for _, l := range lines {
		for _, w := range strings.Fields(l) {
			if len(w) != PrefixLen {
				t.Fatalf("engraved word %q is not %d letters", w, PrefixLen)
			}
		}
	}

	norm, err := NormalizeMnemonics(lines)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	got, err := Combine(norm, nil)
	if err != nil {
		t.Fatalf("combine: %v", err)
	}
	if string(got) != string(key) {
		t.Errorf("recovered %q, want %q", got, key)
	}
}
