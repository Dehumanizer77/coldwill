package slip39

import (
	"errors"
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

// WordError says which word in a line could not be resolved and why, so the
// caller can phrase it in the reader's language. Suggestion is set when the
// first PrefixLen letters match a word but the rest does not, which is the
// signature of a typo rather than an unknown word.
type WordError struct {
	Part       int    // 1-based, 0 when a single line was normalized
	Index      int    // 1-based word position in the line
	Word       string // what the person wrote
	TooShort   bool   // shorter than PrefixLen, therefore ambiguous
	Suggestion string // the word those first letters actually belong to
}

func (e *WordError) Error() string {
	switch {
	case e.TooShort:
		return fmt.Sprintf("word %d %q is shorter than %d letters", e.Index, e.Word, PrefixLen)
	case e.Suggestion != "":
		return fmt.Sprintf("word %d %q is not in the SLIP-39 wordlist (did you mean %q?)", e.Index, e.Word, e.Suggestion)
	default:
		return fmt.Sprintf("word %d %q is not in the SLIP-39 wordlist", e.Index, e.Word)
	}
}

// ErrNoWords means the line held nothing that could be a word.
var ErrNoWords = errors.New("no words in the line")

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
		return "", ErrNoWords
	}
	words := make([]string, len(tokens))
	for i, t := range tokens {
		w, err := expandWord(t)
		if err != nil {
			var we *WordError
			if errors.As(err, &we) {
				we.Index = i + 1
			}
			return "", err
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
		return "", &WordError{Word: t, TooShort: true}
	}
	full, ok := prefixIndex[t[:PrefixLen]]
	if !ok {
		return "", &WordError{Word: t}
	}
	if !strings.HasPrefix(full, t) {
		return "", &WordError{Word: t, Suggestion: full}
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
			var we *WordError
			if errors.As(err, &we) {
				we.Part = i + 1
			}
			return nil, err
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
