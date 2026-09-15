package pdf

import (
	"bytes"
	"compress/zlib"
	"reflect"

	"coldwill/offline/internal/textmap"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestFontCoversSlovakAndCzech(t *testing.T) {
	f, err := loadFont()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range "aáäčďéěíĺľňóôŕřšťúůýžAÁÄČĎÉĚÍĹĽŇÓÔŔŘŠŤÚŮÝŽ0123456789 .,:;?!„“‚‘\"'–—-()%€§/" {
		g, ok := f.glyph(r)
		if !ok {
			t.Errorf("no glyph for %q", r)
			continue
		}
		if f.advance(g) == 0 {
			t.Errorf("%q has no width", r)
		}
	}
}

func TestWrapNeverBreaksAWord(t *testing.T) {
	f, err := loadFont()
	if err != nil {
		t.Fatal(err)
	}
	words := strings.Fields(strings.Repeat("Obecné zastupiteľstvo schválilo 86 tisíc eur na opravu starého mlyna, ", 20))
	width := pageW - 2*margin
	lines, err := wrap(f, words, bodySize, width)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 5 {
		t.Fatalf("expected many lines, got %d", len(lines))
	}
	var flat []string
	for _, ln := range lines {
		if w := f.width(strings.Join(ln, " "), bodySize); w > width+0.01 {
			t.Errorf("line is %.1f pt wide, more than %.1f: %q", w, width, ln)
		}
		flat = append(flat, ln...)
	}
	if strings.Join(flat, " ") != strings.Join(words, " ") {
		t.Errorf("wrapping changed the words")
	}
}

func TestParagraphsStayWholeAcrossPages(t *testing.T) {
	f, err := loadFont()
	if err != nil {
		t.Fatal(err)
	}
	var paras [][]string
	for i := 0; i < 30; i++ {
		paras = append(paras, strings.Fields(strings.Repeat("slovo ", 40+i)))
	}
	pages, err := layout(f, []string{"Nadpis"}, paras)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) < 3 {
		t.Fatalf("expected several pages, got %d", len(pages))
	}
	pageOf := map[int]int{}
	for pi, pg := range pages {
		for _, ln := range pg {
			if ln.y < margin-0.01 {
				t.Errorf("page %d: a line sits below the bottom margin", pi+1)
			}
			if ln.para < 0 {
				continue
			}
			if first, ok := pageOf[ln.para]; !ok {
				pageOf[ln.para] = pi
			} else if first != pi {
				t.Errorf("paragraph %d starts on page %d and continues on page %d", ln.para+1, first+1, pi+1)
			}
		}
	}
	if len(pageOf) != len(paras) {
		t.Errorf("%d paragraphs laid out, want %d", len(pageOf), len(paras))
	}
}

func TestParagraphLongerThanAPageIsSplit(t *testing.T) {
	f, err := loadFont()
	if err != nil {
		t.Fatal(err)
	}
	words := strings.Fields(strings.Repeat("dlhý odsek ", 1500))
	pages, err := layout(f, nil, [][]string{{"Krátky."}, words})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, pg := range pages {
		for _, ln := range pg {
			if ln.para == 1 {
				got = append(got, ln.words...)
			}
		}
	}
	if len(pages) < 2 || strings.Join(got, " ") != strings.Join(words, " ") {
		t.Errorf("a long paragraph should run across %d pages with every word kept", len(pages))
	}
}

func write(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	err := Write(&b, []string{"Starý", "mlyn"}, [][]string{
		{"Ťava", "prišla", "4\u00a0300", "krát."},
		{"Druhý", "odsek", "–", "koniec."},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestWriteIsWellFormed(t *testing.T) {
	doc := write(t)
	if !bytes.HasPrefix(doc, []byte("%PDF-1.7\n")) {
		t.Fatalf("no PDF header")
	}
	m := regexp.MustCompile(`startxref\n(\d+)\n%%EOF\n$`).FindSubmatch(doc)
	if m == nil {
		t.Fatalf("no startxref at the end")
	}
	xref, _ := strconv.Atoi(string(m[1]))
	lines := strings.Split(string(doc[xref:]), "\n")
	if lines[0] != "xref" {
		t.Fatalf("startxref points at %q", lines[0])
	}
	var count int
	if _, err := fmt.Sscanf(lines[1], "0 %d", &count); err != nil || count < 8 {
		t.Fatalf("xref header %q", lines[1])
	}
	for i := 1; i < count; i++ {
		entry := lines[2+i]
		if len(entry)+1 != 20 {
			t.Errorf("xref entry %d is %d bytes, want 20", i, len(entry)+1)
		}
		off, _ := strconv.Atoi(entry[:10])
		if want := fmt.Sprintf("%d 0 obj\n", i); !bytes.HasPrefix(doc[off:], []byte(want)) {
			t.Errorf("xref entry %d points at %q", i, doc[off:off+12])
		}
	}
}

// The printout is supposed to look like any other printed text, so the file
// must not say what made it.
func TestWriteLeavesNoMetadata(t *testing.T) {
	doc := write(t)
	for _, bad := range []string{"/Info", "/Producer", "/Creator", "/CreationDate", "/Title", "/Metadata", "coldwill"} {
		if bytes.Contains(doc, []byte(bad)) {
			t.Errorf("PDF contains %q", bad)
		}
	}
}

func TestWriteIsDeterministic(t *testing.T) {
	if !bytes.Equal(write(t), write(t)) {
		t.Errorf("the same input gave different files")
	}
}

func TestWriteRefusesWhatItCannotPrint(t *testing.T) {
	var ue *UnprintableError
	err := Write(io.Discard, nil, [][]string{{"ahoj", "🙂", "svet"}})
	if !errors.As(err, &ue) || string(ue.Chars) != "🙂" {
		t.Errorf("emoji: got %v", err)
	}
	var tw *TooWideError
	err = Write(io.Discard, nil, [][]string{{"krátke", strings.Repeat("x", 400)}})
	if !errors.As(err, &tw) {
		t.Errorf("a word wider than the line: got %v", err)
	}
}

// Decode the actual content streams written by Write. This checks exported
// boundaries against the PDF bytes, including page breaks and standalone marks.
func TestLineEndsMatchPrintedPDF(t *testing.T) {
	title := []string{"A", "title"}
	paras := [][]string{{"a", "-", "b", "4\u00a0300", "c"}, strings.Fields(strings.Repeat("alpha - beta gamma delta ", 700)), {"Last", "paragraph."}}
	ends, err := LineEnds(title, paras)
	if err != nil {
		t.Fatal(err)
	}
	var doc bytes.Buffer
	if err := Write(&doc, title, paras); err != nil {
		t.Fatal(err)
	}
	f, err := loadFont()
	if err != nil {
		t.Fatal(err)
	}
	decode := map[uint16]rune{}
	for _, words := range append([][]string{title}, paras...) {
		for _, r := range strings.Join(words, " ") {
			g, _ := f.glyph(r)
			decode[g] = r
		}
	}
	// NBSP may share a glyph with the ordinary space in this font.
	space, _ := f.glyph(' ')
	decode[space] = ' '
	var printed []string
	streams := regexp.MustCompile(`(?s)stream\n(.*?)\nendstream`).FindAllSubmatch(doc.Bytes(), -1)
	bodyLine := regexp.MustCompile(`BT /F1 11\.50 Tf[^\n]* <([0-9A-F]+)> Tj ET`)
	for _, stream := range streams {
		zr, err := zlib.NewReader(bytes.NewReader(stream[1]))
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(zr)
		zr.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range bodyLine.FindAllSubmatch(data, -1) {
			var b strings.Builder
			for i := 0; i < len(line[1]); i += 4 {
				g, err := strconv.ParseUint(string(line[1][i:i+4]), 16, 16)
				if err != nil {
					t.Fatal(err)
				}
				b.WriteRune(decode[uint16(g)])
			}
			printed = append(printed, b.String())
		}
	}
	var expected []string
	for p, tokens := range paras {
		start := 0
		for i, end := range ends[p] {
			if end {
				expected = append(expected, strings.ReplaceAll(strings.Join(tokens[start:i+1], " "), "\u00a0", " "))
				start = i + 1
			}
		}
		if start != len(tokens) {
			t.Fatalf("paragraph %d has no final boundary", p)
		}
	}
	if len(printed) < 100 {
		t.Fatalf("expected a multipage document, got %d lines", len(printed))
	}
	if !reflect.DeepEqual(printed, expected) {
		t.Fatal("exported line boundaries differ from printed PDF")
	}

	// Exercise textmap with the exported token indexes, which include the dashes.
	var body []string
	for _, tokens := range paras {
		body = append(body, strings.Join(tokens, " "))
	}
	text := textmap.Parse(strings.Join(body, "\n\n"))
	lineEnd := func(p, token int) bool { return ends[p][token] }
	positions, err := text.Hide(strings.Repeat(" ", 4000), func(int) int { return 0 }, lineEnd)
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range positions {
		words := 0
		token := -1
		for i, tok := range text.Paragraphs[pos.Paragraph-1] {
			if tok.Word {
				words++
				if words == pos.Word {
					token = i
					break
				}
			}
		}
		if token < 0 {
			t.Fatal("word missing")
		}
		tokens := paras[pos.Paragraph-1]
		last := token
		for !ends[pos.Paragraph-1][last] {
			last++
		}
		lineTail := []rune(strings.Join(tokens[token:last+1], " "))
		if pos.Character > len(lineTail) || lineTail[pos.Character-1] != ' ' {
			t.Fatalf("position points outside printed line or to NBSP: %+v", pos)
		}
	}
}

func TestLineEndsRejectsUnprintableLayout(t *testing.T) {
	for _, paras := range [][][]string{{{"🙂"}}, {{strings.Repeat("x", 400)}}} {
		if _, err := LineEnds(nil, paras); err == nil {
			t.Fatal("invalid layout accepted")
		}
	}
}
