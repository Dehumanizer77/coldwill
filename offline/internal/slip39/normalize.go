package slip39

import (
	"fmt"
	"strings"
)

// PrefixLen is how many leading letters identify a SLIP-39 word. The official
// wordlist is built so that four letters are unique across all 1024 words —
// which is exactly why metal backup plates only have room to stamp four, and
// why a part engraved as "acad rive kern…" is not a damaged part but a
// complete one. Three letters would not do: 226 of them are ambiguous.
const PrefixLen = 4

// prefixIndex maps each 4-letter prefix to its full word.
var prefixIndex map[string]string

func initPrefixIndex() {
	prefixIndex = make(map[string]string, len(wordlist))
	for _, w := range wordlist {
		if len(w) < PrefixLen {
			panic("slip39: wordlist contains a word shorter than the prefix length: " + w)
		}
		p := w[:PrefixLen]
		if other, dup := prefixIndex[p]; dup {
			panic("slip39: wordlist prefix collision: " + other + " / " + w)
		}
		prefixIndex[p] = w
	}
}

// NormalizeMnemonic turns what a person reads off a metal plate into a
// spec-compliant mnemonic: it lowercases, ignores numbering and punctuation,
// and expands four-letter (or longer) prefixes to whole words. It is strict
// where guessing would be dangerous — an unknown or ambiguous token is an
// error naming the offending word, never a silent substitution.
func NormalizeMnemonic(line string) (string, error) {
	tokens := strings.FieldsFunc(strings.ToLower(line), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	if len(tokens) == 0 {
		return "", fmt.Errorf("riadok neobsahuje žiadne slová")
	}
	words := make([]string, len(tokens))
	for i, t := range tokens {
		w, err := expandWord(t)
		if err != nil {
			return "", fmt.Errorf("%d. slovo: %w", i+1, err)
		}
		words[i] = w
	}
	return strings.Join(words, " "), nil
}

func expandWord(t string) (string, error) {
	if _, ok := wordIndex[t]; ok {
		return t, nil
	}
	if len(t) < PrefixLen {
		return "", fmt.Errorf("%q je príliš krátke — na kove sú vždy aspoň %d písmená", t, PrefixLen)
	}
	full, ok := prefixIndex[t[:PrefixLen]]
	if !ok {
		return "", fmt.Errorf("%q nie je slovo zo SLIP-39 zoznamu", t)
	}
	if !strings.HasPrefix(full, t) {
		return "", fmt.Errorf("%q nie je slovo zo SLIP-39 zoznamu (podľa prvých %d písmen by to malo byť %q)",
			t, PrefixLen, full)
	}
	return full, nil
}

// NormalizeMnemonics normalizes several parts at once, one per line, skipping
// blank lines.
func NormalizeMnemonics(lines []string) ([]string, error) {
	out := make([]string, 0, len(lines))
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		m, err := NormalizeMnemonic(ln)
		if err != nil {
			return nil, fmt.Errorf("%d. časť: %w", i+1, err)
		}
		out = append(out, m)
	}
	return out, nil
}

// Wordlist returns a copy of the official 1024-word SLIP-39 list, in index
// order. The recovery page ships it to the browser so it can complete words
// offline, exactly as this package does server-side.
func Wordlist() []string {
	out := make([]string, len(wordlist))
	copy(out, wordlist)
	return out
}
