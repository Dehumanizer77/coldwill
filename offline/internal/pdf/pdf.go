// Package pdf lays a plain text out on A4 pages and writes it as a PDF with its
// font embedded. It has one job: printing the text a passphrase map points
// into, so that the paragraphs and words on paper are exactly the ones the map
// was counted on. The layout happens once, here, so the printout does not
// depend on a browser, its fonts or its print settings.
//
// The file carries no metadata at all, no title, producer or dates, because it
// is meant to look like any other printed text. The same input always gives the
// same bytes.
package pdf

import (
	"bytes"
	"compress/zlib"
	_ "embed"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
)

// Liberation Serif covers Slovak and Czech, and embedding it means every
// printer draws the same glyphs. Its licence is in LICENSE-LiberationSerif.txt.
//
//go:embed LiberationSerif-Regular.ttf
var serifTTF []byte

var loadFont = sync.OnceValues(func() (*ttf, error) { return parseTTF(serifTTF) })

// Page geometry, in points.
const (
	pageW, pageH = 595.276, 841.890 // A4
	margin       = 72.0             // about 25 mm on every side
	titleSize    = 18.0
	titleLeading = 23.0
	bodySize     = 11.5
	bodyLeading  = 15.5
)

// UnprintableError lists the characters the embedded font has no glyph for.
type UnprintableError struct{ Chars []rune }

func (e *UnprintableError) Error() string {
	return fmt.Sprintf("pdf: the font has no glyph for %q", string(e.Chars))
}

// TooWideError is a word longer than a whole line. Printing it would mean
// breaking it, and a broken word no longer counts as one.
type TooWideError struct{ Word string }

func (e *TooWideError) Error() string { return fmt.Sprintf("pdf: %q is wider than a line", e.Word) }

// Write lays out an optional title and the paragraphs, each given as its words
// in order, and writes the PDF to w.
func Write(w io.Writer, title []string, paras [][]string) error {
	f, err := loadFont()
	if err != nil {
		return err
	}
	if err := checkGlyphs(f, title, paras); err != nil {
		return err
	}
	pages, err := layout(f, title, paras)
	if err != nil {
		return err
	}

	used := map[uint16]rune{} // every glyph drawn, and the character it stands for
	contents := make([][]byte, len(pages))
	for i, pg := range pages {
		var b bytes.Buffer
		for _, ln := range pg {
			fmt.Fprintf(&b, "BT /F1 %s Tf 1 0 0 1 %s %s Tm <", num(ln.size), num(margin), num(ln.y))
			for j, word := range ln.words {
				if j > 0 {
					glyphHex(&b, f, ' ', used)
				}
				for _, r := range word {
					glyphHex(&b, f, r, used)
				}
			}
			b.WriteString("> Tj ET\n")
		}
		contents[i] = b.Bytes()
	}
	_, err = w.Write(assemble(f, used, contents))
	return err
}

// LineEnds returns one flag per input token, true at each printed line end.
// Paragraph and token indexes are zero-based, including standalone marks.
// It uses the same layout and validation as Write, including title and page breaks.
func LineEnds(title []string, paras [][]string) ([][]bool, error) {
	f, err := loadFont()
	if err != nil {
		return nil, err
	}
	if err := checkGlyphs(f, title, paras); err != nil {
		return nil, err
	}
	pages, err := layout(f, title, paras)
	if err != nil {
		return nil, err
	}
	ends := make([][]bool, len(paras))
	offsets := make([]int, len(paras))
	for p := range paras {
		ends[p] = make([]bool, len(paras[p]))
	}
	for _, pg := range pages {
		for _, ln := range pg {
			if ln.para < 0 {
				continue
			}
			offsets[ln.para] += len(ln.words)
			ends[ln.para][offsets[ln.para]-1] = true
		}
	}
	return ends, nil
}

func checkGlyphs(f *ttf, title []string, paras [][]string) error {
	var bad []rune
	seen := map[rune]bool{}
	check := func(words []string) {
		for _, word := range words {
			for _, r := range word {
				if _, ok := f.glyph(r); !ok && !seen[r] {
					seen[r] = true
					bad = append(bad, r)
				}
			}
		}
	}
	check(title)
	for _, p := range paras {
		check(p)
	}
	if len(bad) > 0 {
		return &UnprintableError{Chars: bad}
	}
	return nil
}

// line is one printed line: its words, its size and where its baseline sits.
// para is the paragraph it belongs to, or -1 for the title.
type line struct {
	words []string
	size  float64
	y     float64
	para  int
}

type page []line

// layout breaks the title and the paragraphs into lines, and the lines into
// pages. Lines break only at spaces, never inside a word, and paragraphs are
// separated by one empty line, which is exactly what the map's rules call a
// paragraph. A paragraph that does not fit at the bottom of a page moves to the
// next one whole, because a page break looks too much like the empty line
// between paragraphs; only a paragraph longer than a page is split.
func layout(f *ttf, title []string, paras [][]string) ([]page, error) {
	width := pageW - 2*margin
	top, bottom := pageH-margin, margin
	var pages []page
	var cur page
	y := top
	newPage := func() {
		if len(cur) > 0 {
			pages = append(pages, cur)
		}
		cur, y = nil, top
	}

	if len(title) > 0 {
		lines, err := wrap(f, title, titleSize, width)
		if err != nil {
			return nil, err
		}
		for _, ws := range lines {
			y -= titleLeading
			cur = append(cur, line{ws, titleSize, y, -1})
		}
	}
	for i, p := range paras {
		lines, err := wrap(f, p, bodySize, width)
		if err != nil {
			return nil, err
		}
		if len(cur) > 0 {
			y -= bodyLeading // the empty line before a paragraph, never at the top of a page
		}
		need := float64(len(lines)) * bodyLeading
		if len(cur) > 0 && y-need < bottom && need <= top-bottom {
			newPage()
		}
		for _, ws := range lines {
			if y-bodyLeading < bottom {
				newPage()
			}
			y -= bodyLeading
			cur = append(cur, line{ws, bodySize, y, i})
		}
	}
	newPage()
	if len(pages) == 0 {
		pages = []page{nil}
	}
	return pages, nil
}

// wrap fills lines greedily with whole words.
func wrap(f *ttf, words []string, size, width float64) ([][]string, error) {
	space := f.width(" ", size)
	var lines [][]string
	var cur []string
	w := 0.0
	for _, word := range words {
		ww := f.width(word, size)
		if ww > width {
			return nil, &TooWideError{Word: word}
		}
		if len(cur) > 0 && w+space+ww > width {
			lines = append(lines, cur)
			cur, w = nil, 0
		}
		if len(cur) > 0 {
			w += space
		}
		cur = append(cur, word)
		w += ww
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	return lines, nil
}

func glyphHex(b *bytes.Buffer, f *ttf, r rune, used map[uint16]rune) {
	g, _ := f.glyph(r)
	if _, ok := used[g]; !ok {
		if r == '\u00a0' {
			r = ' '
		}
		used[g] = r
	}
	fmt.Fprintf(b, "%04X", g)
}

// assemble writes the objects, the cross-reference table and the trailer.
// Objects 1-7 are the catalogue, the page tree and the font; each page then
// takes two numbers, the page and its content stream.
func assemble(f *ttf, used map[uint16]rune, contents [][]byte) []byte {
	total := 7 + 2*len(contents)
	offsets := make([]int, total)
	var buf bytes.Buffer
	object := func(n int, body string) {
		offsets[n-1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", n, body)
	}
	stream := func(n int, dict string, data []byte) {
		offsets[n-1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n<< %s /Length %d >>\nstream\n", n, dict, len(data))
		buf.Write(data)
		buf.WriteString("\nendstream\nendobj\n")
	}

	buf.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	object(1, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, len(contents))
	for i := range contents {
		kids[i] = fmt.Sprintf("%d 0 R", 8+2*i)
	}
	object(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(contents)))
	object(3, "<< /Type /Font /Subtype /Type0 /BaseFont /LiberationSerif /Encoding /Identity-H /DescendantFonts [4 0 R] /ToUnicode 7 0 R >>")
	object(4, "<< /Type /Font /Subtype /CIDFontType2 /BaseFont /LiberationSerif"+
		" /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >>"+
		" /FontDescriptor 5 0 R /CIDToGIDMap /Identity /DW 1000 /W ["+widths(f, used)+"] >>")
	object(5, fmt.Sprintf("<< /Type /FontDescriptor /FontName /LiberationSerif /Flags 34"+
		" /FontBBox [%d %d %d %d] /ItalicAngle 0 /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 6 0 R >>",
		f.toPDF(f.bbox[0]), f.toPDF(f.bbox[1]), f.toPDF(f.bbox[2]), f.toPDF(f.bbox[3]),
		f.toPDF(f.ascent), f.toPDF(f.descent), f.toPDF(f.capHeight)))
	stream(6, fmt.Sprintf("/Filter /FlateDecode /Length1 %d", len(f.data)), deflate(f.data))
	stream(7, "/Filter /FlateDecode", deflate(toUnicode(used)))
	for i, c := range contents {
		object(8+2*i, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %s %s] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>",
			num(pageW), num(pageH), 9+2*i))
		stream(9+2*i, "/Filter /FlateDecode", deflate(c))
	}

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", total+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", total+1, xref)
	return buf.Bytes()
}

func sortedGlyphs(used map[uint16]rune) []uint16 {
	gs := make([]uint16, 0, len(used))
	for g := range used {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i] < gs[j] })
	return gs
}

func widths(f *ttf, used map[uint16]rune) string {
	var parts []string
	for _, g := range sortedGlyphs(used) {
		parts = append(parts, fmt.Sprintf("%d [%d]", g, f.toPDF(f.advance(g))))
	}
	return strings.Join(parts, " ")
}

// toUnicode maps glyphs back to characters, so text copied out of the PDF is
// the text that went in.
func toUnicode(used map[uint16]rune) []byte {
	var b bytes.Buffer
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n" +
		"/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n" +
		"/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n" +
		"1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	gs := sortedGlyphs(used)
	for start := 0; start < len(gs); start += 100 {
		end := min(start+100, len(gs))
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for _, g := range gs[start:end] {
			fmt.Fprintf(&b, "<%04X> <", g)
			for _, u := range utf16.Encode([]rune{used[g]}) {
				fmt.Fprintf(&b, "%04X", u)
			}
			b.WriteString(">\n")
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return b.Bytes()
}

func deflate(data []byte) []byte {
	var b bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&b, zlib.BestCompression) // the level is valid, so there is no error
	zw.Write(data)
	zw.Close()
	return b.Bytes()
}

func num(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
