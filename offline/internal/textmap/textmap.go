// Package textmap hides a passphrase in an ordinary text as a list of word
// positions. Every character of the passphrase is the first letter or digit of
// one word, and the list says where that word is: its paragraph, and its place
// in that paragraph. The text alone does not say which words matter and the
// list alone does not say which text it belongs to; together they give the
// passphrase back to a person with a pencil and no tool.
//
// The list is only as good as the agreement between this package and that
// person about what a paragraph and a word are. So the rules here are the ones
// the list prints next to itself, and nothing cleverer.
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
	// Initial is the word's first letter or digit folded to a-z or 0-9, or 0
	// when it has none that folds (a word in another script, say).
	Initial byte
}

// Text is a text cut into paragraphs the way the printout shows them.
type Text struct {
	Paragraphs [][]Token
}

// Position is where one character of the passphrase is: the Word-th word of the
// Paragraph-th paragraph, both counted from 1, the way the heir counts.
type Position struct {
	Paragraph, Word int
}

// invisible removes characters that change nothing on paper but would make the
// words here differ from the words a person sees.
var invisible = strings.NewReplacer("\u00ad", "", "\u200b", "", "\u200c", "", "\u200d", "", "\u2060", "", "\ufeff", "")

// Parse cuts a text into paragraphs and words. Paragraphs are separated by an
// empty line, and the lines inside one are joined. A text with no empty line
// between its lines gets one paragraph per line, which is how text copied out
// of a word processor usually arrives. A paragraph without a single word (a row
// of asterisks) is dropped, so the printout never shows a line that may or may
// not count.
func Parse(s string) Text {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = invisible.Replace(s)
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
			return Token{Text: text, Word: true, Initial: fold(r)}
		}
	}
	return Token{Text: text}
}

// folds maps accented Latin letters to the plain letter a person writes when
// told to leave the accents out: č → c, ľ → l, ô → o, ř → r.
var folds = func() map[rune]byte {
	const pairs = "àaáaâaãaäaåaçcèeéeêeëeìiíiîiïiñnòoóoôoõoöoøoùuúuûuüuýyÿy" +
		"āaăaąaćcĉcċcčcďdđdēeĕeėeęeěeĝgğgġgģgĥhħhĩiīiĭiįiıiĵjķkĺlļlľlŀlłl" +
		"ńnņnňnōoŏoőoŕrŗrřrśsŝsşsšsţtťtŧtũuūuŭuůuűuųuŵwŷyźzżzžz"
	rs := []rune(pairs)
	m := make(map[rune]byte, len(rs)/2)
	for i := 0; i+1 < len(rs); i += 2 {
		m[rs[i]] = byte(rs[i+1])
	}
	return m
}()

func fold(r rune) byte {
	r = unicode.ToLower(r)
	if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
		return byte(r)
	}
	return folds[r]
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
// they first appear: a map only ever yields lowercase a-z and 0-9.
type InvalidError struct{ Chars []rune }

func (e *InvalidError) Error() string {
	return fmt.Sprintf("textmap: passphrase characters outside a-z and 0-9: %q", string(e.Chars))
}

// MissingError lists, sorted, the passphrase characters no word in the text
// starts with.
type MissingError struct{ Chars []byte }

func (e *MissingError) Error() string {
	return fmt.Sprintf("textmap: no word starts with %q", string(e.Chars))
}

// Hide finds a word for every character of the passphrase. pick(n) returns a
// number in [0, n) and chooses among the words that fit. A character that
// repeats gets a different word each time while the text has one to spare, so
// the list does not show which characters are the same.
func (t Text) Hide(passphrase string, pick func(n int) int) ([]Position, error) {
	if passphrase == "" || len(t.Paragraphs) == 0 {
		return nil, ErrEmpty
	}
	var invalid []rune
	for _, r := range passphrase {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') && !strings.ContainsRune(string(invalid), r) {
			invalid = append(invalid, r)
		}
	}
	if len(invalid) > 0 {
		return nil, &InvalidError{Chars: invalid}
	}

	where := map[byte][]Position{}
	for p, para := range t.Paragraphs {
		w := 0
		for _, tok := range para {
			if !tok.Word {
				continue
			}
			w++
			if tok.Initial != 0 {
				where[tok.Initial] = append(where[tok.Initial], Position{Paragraph: p + 1, Word: w})
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
func (t Text) Reveal(ps []Position) (string, error) {
	b := make([]byte, 0, len(ps))
	for _, pos := range ps {
		tok, ok := t.word(pos)
		if !ok || tok.Initial == 0 {
			return "", fmt.Errorf("textmap: no usable word at paragraph %d, word %d", pos.Paragraph, pos.Word)
		}
		b = append(b, tok.Initial)
	}
	return string(b), nil
}

func (t Text) word(pos Position) (Token, bool) {
	if pos.Paragraph < 1 || pos.Paragraph > len(t.Paragraphs) {
		return Token{}, false
	}
	n := 0
	for _, tok := range t.Paragraphs[pos.Paragraph-1] {
		if tok.Word {
			if n++; n == pos.Word {
				return tok, true
			}
		}
	}
	return Token{}, false
}
