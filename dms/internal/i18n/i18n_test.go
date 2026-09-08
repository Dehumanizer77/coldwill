package i18n

import (
	"regexp"
	"strings"
	"testing"
)

// English is the source of truth, and a translation that silently loses a key
// renders that part of the page in another language. The test names what is
// missing so the gap is visible rather than discovered by a reader.
func TestSlovakCoversEveryKey(t *testing.T) {
	var missing, extra []string
	for _, k := range Keys(EN) {
		if _, ok := catalogs[SK][k]; !ok {
			missing = append(missing, k)
		}
	}
	for _, k := range Keys(SK) {
		if _, ok := catalogs[EN][k]; !ok {
			extra = append(extra, k)
		}
	}
	if len(missing) > 0 {
		t.Errorf("Slovak is missing %d key(s): %s", len(missing), strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		t.Errorf("Slovak has %d key(s) English does not: %s", len(extra), strings.Join(extra, ", "))
	}
}

// Czech is allowed to be incomplete for now, but it must not invent keys that
// no longer exist, which is how a translation quietly stops being applied.
func TestCzechHasNoUnknownKeys(t *testing.T) {
	for _, k := range Keys(CS) {
		if _, ok := catalogs[EN][k]; !ok {
			t.Errorf("Czech has key %q that English does not", k)
		}
	}
}

// A translation with a different number of placeholders than the original
// either drops a value or panics at render time with %!s(MISSING).
func TestPlaceholdersMatchEnglish(t *testing.T) {
	verb := regexp.MustCompile(`%[a-zA-Z]`)
	for _, l := range []Lang{SK, CS} {
		for _, k := range Keys(l) {
			want := verb.FindAllString(catalogs[EN][k], -1)
			got := verb.FindAllString(catalogs[l][k], -1)
			if len(want) != len(got) {
				t.Errorf("%s/%s: %d placeholders, English has %d", l, k, len(got), len(want))
			}
		}
	}
}

func TestFallsBackToEnglish(t *testing.T) {
	// "subj.healthy" is deliberately not in the Czech catalogue yet.
	if got := string(T(CS, "subj.healthy")); got != enMessages["subj.healthy"] {
		t.Errorf("Czech did not fall back to English: %q", got)
	}
	if got := string(T(EN, "no.such.key")); got != "[[no.such.key]]" {
		t.Errorf("a missing key should be visible, got %q", got)
	}
}

// Arguments carry user input (names, bank details), so they must be escaped
// even though the message itself is trusted HTML.
func TestArgumentsAreEscaped(t *testing.T) {
	got := string(T(EN, "page.confirm.partial", `<script>alert(1)</script>`, 2))
	if strings.Contains(got, "<script>") {
		t.Fatalf("argument was not escaped: %q", got)
	}
	if !strings.Contains(got, "&lt;script&gt;") {
		t.Fatalf("argument not escaped as expected: %q", got)
	}
}

func TestParse(t *testing.T) {
	for in, want := range map[string]Lang{"en": EN, "sk": SK, "cs": CS, "": Default, "de": Default, "../x": Default} {
		if got := Parse(in); got != want {
			t.Errorf("Parse(%q) = %q, want %q", in, got, want)
		}
	}
}
