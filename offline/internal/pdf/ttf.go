package pdf

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ttf is what the writer needs from a TrueType font: which glyph draws a
// character, how wide it is, and the few metrics a PDF font descriptor asks
// for. The font is embedded whole, so nothing here has to understand outlines.
type ttf struct {
	data       []byte
	unitsPerEm int
	bbox       [4]int // xMin, yMin, xMax, yMax
	ascent     int
	descent    int
	capHeight  int
	advances   []uint16
	glyphs     map[rune]uint16
}

var errFont = errors.New("pdf: malformed font")

func parseTTF(data []byte) (*ttf, error) {
	tables, err := tableDirectory(data)
	if err != nil {
		return nil, err
	}
	head, hhea, hmtx, cmap := tables["head"], tables["hhea"], tables["hmtx"], tables["cmap"]
	if len(head) < 54 || len(hhea) < 36 || hmtx == nil || cmap == nil {
		return nil, errFont
	}
	f := &ttf{
		data:       data,
		unitsPerEm: int(u16(head, 18)),
		bbox:       [4]int{int(i16(head, 36)), int(i16(head, 38)), int(i16(head, 40)), int(i16(head, 42))},
		ascent:     int(i16(hhea, 4)),
		descent:    int(i16(hhea, 6)),
	}
	f.capHeight = f.ascent
	if os2 := tables["OS/2"]; len(os2) >= 90 && u16(os2, 0) >= 2 {
		f.capHeight = int(i16(os2, 88))
	}
	n := int(u16(hhea, 34))
	if f.unitsPerEm == 0 || n == 0 || len(hmtx) < 4*n {
		return nil, errFont
	}
	f.advances = make([]uint16, n)
	for i := range f.advances {
		f.advances[i] = u16(hmtx, 4*i)
	}
	if f.glyphs, err = parseCmap(cmap); err != nil {
		return nil, err
	}
	return f, nil
}

func tableDirectory(data []byte) (map[string][]byte, error) {
	if len(data) < 12 || binary.BigEndian.Uint32(data) != 0x00010000 {
		return nil, errFont
	}
	num := int(u16(data, 4))
	if len(data) < 12+16*num {
		return nil, errFont
	}
	out := make(map[string][]byte, num)
	for i := 0; i < num; i++ {
		rec := data[12+16*i:]
		off, size := uint64(binary.BigEndian.Uint32(rec[8:])), uint64(binary.BigEndian.Uint32(rec[12:]))
		if off+size > uint64(len(data)) {
			return nil, errFont
		}
		out[string(rec[:4])] = data[off : off+size]
	}
	return out, nil
}

// parseCmap reads the Unicode BMP subtable (format 4), which covers every
// character a Latin text needs.
func parseCmap(t []byte) (map[rune]uint16, error) {
	if len(t) < 4 {
		return nil, errFont
	}
	num := int(u16(t, 2))
	if len(t) < 4+8*num {
		return nil, errFont
	}
	for i := 0; i < num; i++ {
		platform, encoding := u16(t, 4+8*i), u16(t, 6+8*i)
		off := int(binary.BigEndian.Uint32(t[8+8*i:]))
		bmp := platform == 3 && encoding == 1 || platform == 0 && encoding <= 3
		if !bmp || off+14 > len(t) || u16(t, off) != 4 {
			continue
		}
		return parseFormat4(t[off:])
	}
	return nil, fmt.Errorf("%w: no Unicode BMP cmap", errFont)
}

func parseFormat4(t []byte) (map[rune]uint16, error) {
	length := int(u16(t, 2))
	if length < 14 || length > len(t) {
		return nil, errFont
	}
	t = t[:length]
	segs := int(u16(t, 6)) / 2
	ends, starts, deltas, ranges := 14, 16+2*segs, 16+4*segs, 16+6*segs
	if len(t) < 16+8*segs {
		return nil, errFont
	}
	m := map[rune]uint16{}
	for s := 0; s < segs; s++ {
		end, start := int(u16(t, ends+2*s)), int(u16(t, starts+2*s))
		delta, ro := u16(t, deltas+2*s), int(u16(t, ranges+2*s))
		for c := start; c <= end && c != 0xFFFF; c++ {
			var g uint16
			if ro == 0 {
				g = uint16(c) + delta
			} else {
				at := ranges + 2*s + ro + 2*(c-start)
				if at+2 > len(t) {
					return nil, errFont
				}
				if g = u16(t, at); g != 0 {
					g += delta
				}
			}
			if g != 0 {
				m[rune(c)] = g
			}
		}
	}
	return m, nil
}

// glyph finds the glyph for a character. A no-break space prints as a space:
// the text uses it only to keep a number like "4 300" on one line.
func (f *ttf) glyph(r rune) (uint16, bool) {
	if r == '\u00a0' {
		r = ' '
	}
	g, ok := f.glyphs[r]
	return g, ok
}

func (f *ttf) advance(g uint16) int {
	if int(g) < len(f.advances) {
		return int(f.advances[g])
	}
	return int(f.advances[len(f.advances)-1])
}

// width is how wide s is at the given size, in points.
func (f *ttf) width(s string, size float64) float64 {
	u := 0
	for _, r := range s {
		if g, ok := f.glyph(r); ok {
			u += f.advance(g)
		}
	}
	return float64(u) * size / float64(f.unitsPerEm)
}

// toPDF scales font units to the thousandths of a point PDF widths use.
func (f *ttf) toPDF(v int) int {
	if v < 0 {
		return -((-v*1000 + f.unitsPerEm/2) / f.unitsPerEm)
	}
	return (v*1000 + f.unitsPerEm/2) / f.unitsPerEm
}

func u16(b []byte, off int) uint16 { return binary.BigEndian.Uint16(b[off:]) }
func i16(b []byte, off int) int16  { return int16(binary.BigEndian.Uint16(b[off:])) }
