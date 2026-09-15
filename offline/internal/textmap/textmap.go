// Package textmap maps printable ASCII passphrase characters to positions in printed text.
package textmap

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Token is one piece of a paragraph as it is printed, in order.
type Token struct {
	Text string
	// Word is false for a mark standing on its own, such as a dash, which a
	// person counting words skips.
	Word bool
}

// Text is a text cut into paragraphs the way the printout shows them.
type Text struct {
	Paragraphs [][]Token
}

// Position is where one character of the passphrase is: the Word-th word of the
// Paragraph-th paragraph, at Character within the word. All three count from 1.
type Position struct {
	Paragraph, Word, Character int
}

// invisible removes characters that change nothing on paper but would make the
// words here differ from the words a person sees.
var invisible = strings.NewReplacer("\u00ad", "", "\u200b", "", "\u200c", "", "\u200d", "", "\u2060", "", "\ufeff", "")

// Normalize makes typographic look-alikes unambiguous on paper.
func Normalize(s string) string {
	return typography.Replace(invisible.Replace(s))
}

var typography = strings.NewReplacer("„", "\"", "“", "\"", "”", "\"", "‚", "'", "‘", "'", "’", "'", "–", "-", "—", "-", "…", "...")

// Parse cuts a text into paragraphs and words. Paragraphs are separated by an
// empty line, and the lines inside one are joined. A text with no empty line
// between its lines gets one paragraph per line, which is how text copied out
// of a word processor usually arrives. A paragraph without a single word (a row
// of asterisks) is dropped, so the printout never shows a line that may or may
// not count.
func Parse(s string) Text {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = Normalize(s)
	lines := strings.Split(s, "\n")

	var blocks []string
	if hasEmptyLineBetween(lines) {
		var cur []string
		for _, ln := range append(lines, "") {
			if strings.TrimSpace(ln) != "" {
				cur = append(cur, ln)
				continue
			}
			if len(cur) > 0 {
				blocks = append(blocks, strings.Join(cur, " "))
				cur = nil
			}
		}
	} else {
		for _, ln := range lines {
			if strings.TrimSpace(ln) != "" {
				blocks = append(blocks, ln)
			}
		}
	}

	var t Text
	for _, b := range blocks {
		if p := tokenize(b); countWords(p) > 0 {
			t.Paragraphs = append(t.Paragraphs, p)
		}
	}
	return t
}

func hasEmptyLineBetween(lines []string) bool {
	seenText, gap := false, false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			gap = seenText
			continue
		}
		if gap {
			return true
		}
		seenText = true
	}
	return false
}

var (
	groupHead = regexp.MustCompile(`^[0-9]{1,3}$`)
	groupNext = regexp.MustCompile(`^[0-9]{3}(?:[^0-9]|$)`)
	groupFull = regexp.MustCompile(`^[0-9]{3}$`)
)

// tokenize splits a paragraph at white space, then joins a number written with
// spaces between its thousands ("4 300", "1 250 000") back into one word,
// because that is how it reads. The join is a no-break space, so the printout
// can never wrap such a number onto two lines.
func tokenize(p string) []Token {
	fields := strings.Fields(p)
	var out []Token
	for i := 0; i < len(fields); i++ {
		text := fields[i]
		if groupHead.MatchString(text) {
			for i+1 < len(fields) && groupNext.MatchString(fields[i+1]) {
				i++
				text += "\u00a0" + fields[i]
				if !groupFull.MatchString(fields[i]) {
					break
				}
			}
		}
		out = append(out, newToken(text))
	}
	return out
}

func newToken(text string) Token {
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return Token{Text: text, Word: true}
		}
	}
	return Token{Text: text}
}

func countWords(p []Token) int {
	n := 0
	for _, t := range p {
		if t.Word {
			n++
		}
	}
	return n
}

// ErrEmpty means there was no passphrase, or no text to hide it in.
var ErrEmpty = errors.New("textmap: empty passphrase or text")

// InvalidError lists passphrase characters no word can stand for, in the order
// they first appear: a map only yields printable ASCII (0x20–0x7E).
type InvalidError struct{ Chars []rune }

func (e *InvalidError) Error() string {
	return fmt.Sprintf("textmap: passphrase characters outside printable ASCII: %q", string(e.Chars))
}

// MissingError lists, sorted, characters without a usable printed position.
type MissingError struct{ Chars []byte }

func (e *MissingError) Error() string {
	return fmt.Sprintf("textmap: no usable position for %q", string(e.Chars))
}

// Hide finds a printed position for each character. pick(n) chooses in [0, n).
// Repeated characters use different positions while any remain unused.
// ends reports whether a zero-based paragraph/token ends a printed line.
// With nil ends, only characters inside words are eligible.
func (t Text) Hide(passphrase string, pick func(n int) int, ends func(paragraph, token int) bool) ([]Position, error) {
	if passphrase == "" || len(t.Paragraphs) == 0 {
		return nil, ErrEmpty
	}
	var invalid []rune
	for _, r := range passphrase {
		if (r < ' ' || r > '~') && !strings.ContainsRune(string(invalid), r) {
			invalid = append(invalid, r)
		}
	}
	if len(invalid) > 0 {
		return nil, &InvalidError{Chars: invalid}
	}

	where := map[byte][]Position{}
	for p, para := range t.Paragraphs {
		w := 0
		for i, tok := range para {
			if !tok.Word {
				continue
			}
			w++
			for c, r := range readable(para, p, i, ends) {
				if r >= ' ' && r <= '~' {
					where[byte(r)] = append(where[byte(r)], Position{p + 1, w, c + 1})
				}
			}
		}
	}
	seen := map[byte]bool{}
	var missing []byte
	for i := 0; i < len(passphrase); i++ {
		if c := passphrase[i]; len(where[c]) == 0 && !seen[c] {
			seen[c] = true
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
		return nil, &MissingError{Chars: missing}
	}

	used := map[Position]bool{}
	out := make([]Position, len(passphrase))
	for i := 0; i < len(passphrase); i++ {
		all := where[passphrase[i]]
		var free []Position
		for _, pos := range all {
			if !used[pos] {
				free = append(free, pos)
			}
		}
		if len(free) == 0 {
			free = all
		}
		pos := free[pick(len(free))]
		used[pos] = true
		out[i] = pos
	}
	return out, nil
}

// Reveal reads the passphrase back from the text the way the heir does. The
// server runs it on every map before handing the map out.
func (t Text) Reveal(ps []Position, ends func(paragraph, token int) bool) (string, error) {
	var b strings.Builder
	for _, pos := range ps {
		if pos.Paragraph < 1 || pos.Paragraph > len(t.Paragraphs) || pos.Word < 1 || pos.Character < 1 {
			return "", fmt.Errorf("textmap: invalid position: %+v", pos)
		}
		para := t.Paragraphs[pos.Paragraph-1]
		w := 0
		var chars []rune
		for i, tok := range para {
			if tok.Word {
				w++
				if w == pos.Word {
					chars = readable(para, pos.Paragraph-1, i, ends)
					break
				}
			}
		}
		if pos.Character > len(chars) || chars[pos.Character-1] < ' ' || chars[pos.Character-1] > '~' {
			return "", fmt.Errorf("textmap: no usable character at %+v", pos)
		}
		b.WriteRune(chars[pos.Character-1])
	}
	return b.String(), nil
}

// readable preserves rune offsets, including accents and internal no-break
// spaces, but stops before counting through a potentially ambiguous digraph.
// Standalone marks belong to the preceding word only on the same printed line.
func readable(para []Token, p, i int, ends func(int, int) bool) []rune {
	chars := []rune(para[i].Text)
	// A selectable space must have another counted word on the same line.
	nextWord := false
	if ends != nil {
		for j := i; j+1 < len(para) && !ends(p, j); j++ {
			if para[j+1].Word {
				nextWord = true
				break
			}
		}
		for j := i; j+1 < len(para) && !ends(p, j); j++ {
			if nextWord {
				chars = append(chars, ' ')
			} else {
				// Keep the offset for counting marks, but exclude this space.
				chars = append(chars, 0)
			}
			if para[j+1].Word {
				break
			}
			chars = append(chars, []rune(para[j+1].Text)...)
		}
	}
	for j, r := range chars {
		if unicode.IsMark(r) {
			// A decomposed accent belongs to the preceding printed character.
			// Neither its ASCII base nor later rune offsets are safe to select.
			chars = chars[:max(0, j-1)]
			break
		}
	}
	for j := 1; j < len(chars); j++ {
		a, b := unicode.ToLower(chars[j-1]), unicode.ToLower(chars[j])
		if a == 'c' && b == 'h' || a == 'd' && (b == 'z' || b == 'ž') {
			return chars[:j]
		}
	}
	return chars
}
