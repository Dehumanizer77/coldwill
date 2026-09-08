package i18n

import (
	"regexp"
	"strings"
	"testing"
)

// English is the source of truth, and a translation that silently loses a key
// renders that part of the page in another language. The test names what is
// missing so the gap is visible rather than discovered by a reader.
func TestEveryTranslationCoversEveryKey(t *testing.T) {
	for _, l := range Languages() {
		if l == Default {
			continue
		}
		t.Run(string(l), func(t *testing.T) {
			var missing, extra []string
			for _, k := range Keys(Default) {
				if _, ok := catalogs[l][k]; !ok {
					missing = append(missing, k)
				}
			}
			for _, k := range Keys(l) {
				if _, ok := catalogs[Default][k]; !ok {
					extra = append(extra, k)
				}
			}
			if len(missing) > 0 {
				t.Errorf("missing %d key(s): %s", len(missing), strings.Join(missing, ", "))
			}
			if len(extra) > 0 {
				t.Errorf("has %d key(s) English does not: %s", len(extra), strings.Join(extra, ", "))
			}
		})
	}
}

// A translation with a different number of placeholders than the original
// either drops a value or renders %!s(MISSING) at someone.
func TestPlaceholdersMatchEnglish(t *testing.T) {
	verb := regexp.MustCompile(`%[a-zA-Z]`)
	for _, l := range Languages() {
		for _, k := range Keys(l) {
			want := verb.FindAllString(catalogs[Default][k], -1)
			got := verb.FindAllString(catalogs[l][k], -1)
			if len(want) != len(got) {
				t.Errorf("%s/%s: %d placeholders, English has %d", l, k, len(got), len(want))
			}
		}
	}
}

// Every catalogue is complete today, but the fallback is what keeps a
// half-finished translation usable, so it is tested rather than assumed.
func TestFallsBackToEnglish(t *testing.T) {
	const k = "index.h1"
	saved := catalogs[CS][k]
	delete(catalogs[CS], k)
	defer func() { catalogs[CS][k] = saved }()

	if got := string(T(CS, k)); got != enMessages[k] {
		t.Errorf("a missing key did not fall back to English: %q", got)
	}
	if got := string(T(EN, "no.such.key")); got != "[[no.such.key]]" {
		t.Errorf("a key missing everywhere should be visible, got %q", got)
	}
}

// Arguments carry user input (names, bank details), so they must be escaped
// even though the message itself is trusted HTML.
func TestArgumentsAreEscaped(t *testing.T) {
	got := string(T(EN, "rb.map.held", `<script>alert(1)</script>`))
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
