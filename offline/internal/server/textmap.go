package server

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"strconv"
	"strings"

	"coldwill/offline/internal/i18n"
	"coldwill/offline/internal/pdf"
	"coldwill/offline/internal/textmap"
)

type textmapForm struct {
	page
	Headline, Text string
	Problem        template.HTML
}

type textmapData struct {
	page
	Chars      int
	Map        string
	MapRows    int
	Headline   string
	Paragraphs []string
	PDFURL     template.URL
}

// handleTextmap hides the wallet passphrase in a text the person pastes. It
// picks a printed position for every character, checks the map reads back to
// the passphrase, and hands out the map together with a PDF of the text laid out so
// that its paragraphs and words are the ones the map was counted on. The map
// and the PDF come from the same request, so they cannot drift apart. The
// passphrase is never written into a page, not even back into the form after a
// mistake.
func (s *Server) handleTextmap(w http.ResponseWriter, r *http.Request) {
	l := s.lang(r)
	if r.Method != http.MethodPost {
		s.render(w, "textmap_form.html", textmapForm{page: page{L: l}})
		return
	}
	headline := textmap.Normalize(strings.TrimSpace(r.FormValue("headline")))
	body := r.FormValue("text")
	passphrase := r.FormValue("passphrase")
	again := func(key string, args ...any) {
		s.render(w, "textmap_form.html", textmapForm{
			page: page{L: l}, Headline: headline, Text: body, Problem: i18n.T(l, key, args...),
		})
	}

	text := textmap.Parse(body)
	paras := make([][]string, len(text.Paragraphs))
	preview := make([]string, len(text.Paragraphs))
	for i, p := range text.Paragraphs {
		for _, tok := range p {
			paras[i] = append(paras[i], tok.Text)
		}
		preview[i] = strings.Join(paras[i], " ")
	}
	ends, err := pdf.LineEnds(strings.Fields(headline), paras)
	var unprintable *pdf.UnprintableError
	var tooWide *pdf.TooWideError
	switch {
	case errors.As(err, &unprintable):
		again("tm.err.print", quoteChars(unprintable.Chars))
		return
	case errors.As(err, &tooWide):
		again("tm.err.wide", tooWide.Word)
		return
	case err != nil:
		again("tm.err.pdf", err)
		return
	}

	lineEnd := func(p, token int) bool { return ends[p][token] }
	positions, err := text.Hide(passphrase, randIndex, lineEnd)
	var invalid *textmap.InvalidError
	var missing *textmap.MissingError
	switch {
	case errors.As(err, &invalid):
		again("tm.err.chars", quoteChars(invalid.Chars))
		return
	case errors.As(err, &missing):
		again("tm.err.missing", quoteChars([]rune(string(missing.Chars))))
		return
	case err != nil:
		again("tm.err.empty")
		return
	}
	if back, err := text.Reveal(positions, lineEnd); err != nil || back != passphrase {
		again("tm.err.readback")
		return
	}

	var doc bytes.Buffer
	if err := pdf.Write(&doc, strings.Fields(headline), paras); err != nil {
		again("tm.err.pdf", err)
		return
	}

	m := passphraseMap(l, positions, headline != "")
	s.render(w, "textmap.html", textmapData{
		page:       page{L: l},
		Chars:      len(positions),
		Map:        m,
		MapRows:    strings.Count(m, "\n") + 1,
		Headline:   headline,
		Paragraphs: preview,
		PDFURL:     template.URL("data:application/pdf;base64," + base64.StdEncoding.EncodeToString(doc.Bytes())),
	})
}

// passphraseMap is the text that goes into the database: one line per
// character, spelled out so that nobody reads "3.17" as a number, followed by
// the rules for counting, because the list is no use to an heir who counts
// differently from the tool.
func passphraseMap(l i18n.Lang, ps []textmap.Position, headline bool) string {
	var b strings.Builder
	b.WriteString(i18n.S(l, "tm.map.head") + "\n\n")
	for i, p := range ps {
		b.WriteString(i18n.S(l, "tm.map.line", pad2(i+1), pad2(p.Paragraph), pad2(p.Word), pad2(p.Character)) + "\n")
	}
	rules := []string{"tm.rule.para", "tm.rule.first", "tm.rule.words", "tm.rule.marks", "tm.rule.chars", "tm.rule.space", "tm.rule.exact"}
	if headline {
		rules[1] = "tm.rule.headline"
	}
	b.WriteString("\n" + i18n.S(l, "tm.map.rules") + "\n")
	for _, k := range rules {
		b.WriteString("- " + i18n.S(l, k) + "\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func pad2(n int) string { return fmt.Sprintf("%2d", n) }

// randIndex picks a number in [0, n) from the system's CSPRNG. The choice among
// the positions with the same character has to be random: a rule such as
// "the first one" puts common letters early in their paragraphs, and the list
// alone would start to hint at the characters.
func randIndex(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err) // only when the operating system cannot supply randomness at all
	}
	return int(v.Int64())
}

// quoteChars puts each character of an error message in quotes, so that a
// space or a look-alike letter is visible.
func quoteChars(rs []rune) string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = strconv.Quote(string(r))
	}
	return strings.Join(out, ", ")
}
