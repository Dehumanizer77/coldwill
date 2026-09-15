package textmap

import (
	"errors"
	"math/rand/v2"
	"os"
	"regexp"
	"strconv"

	"coldwill/offline/internal/pdf"
	"reflect"
	"strings"
	"testing"
)

func paragraphTexts(t Text) []string {
	out := make([]string, len(t.Paragraphs))
	for i, p := range t.Paragraphs {
		var ws []string
		for _, tok := range p {
			ws = append(ws, tok.Text)
		}
		out[i] = strings.Join(ws, " ")
	}
	return out
}

func TestParagraphsAreSeparatedByEmptyLines(t *testing.T) {
	got := paragraphTexts(Parse("\nFirst line\nstill the first.\n\nSecond.\r\n\r\n \r\nThird\n\n"))
	want := []string{"First line still the first.", "Second.", "Third"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWithoutEmptyLinesEveryLineIsAParagraph(t *testing.T) {
	got := paragraphTexts(Parse("\n\nOne two\nThree\n\n"))
	want := []string{"One two", "Three"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParagraphWithoutWordsIsDropped(t *testing.T) {
	got := paragraphTexts(Parse("One\n\n* * *\n\n—\n\nTwo"))
	want := []string{"One", "Two"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWordsFollowTheRules(t *testing.T) {
	p := Parse("„Mlyn“ – 7. septembra o 9:30 prišlo 4 300 ľudí (1874) Chata, e-mail 20 %\u00ad").Paragraphs[0]
	var words, marks []string
	for _, tok := range p {
		if tok.Word {
			words = append(words, tok.Text)
		} else {
			marks = append(marks, tok.Text)
		}
	}
	want := []string{`"Mlyn"`, "7.", "septembra", "o", "9:30", "prišlo", "4\u00a0300", "ľudí", "(1874)", "Chata,", "e-mail", "20"}
	if !reflect.DeepEqual(words, want) {
		t.Errorf("words: got %q, want %q", words, want)
	}
	if !reflect.DeepEqual(marks, []string{"-", "%"}) {
		t.Errorf("marks: %q", marks)
	}
}

func TestOnlyRealThousandsAreJoined(t *testing.T) {
	cases := map[string][]string{
		"1 250 000 eur":         {"1\u00a0250\u00a0000", "eur"},
		"v roku 2019 150 ľudí":  {"v", "roku", "2019", "150", "ľudí"},
		"skóre 12 34":           {"skóre", "12", "34"},
		"asi 5 100, potom":      {"asi", "5\u00a0100,", "potom"},
		"1 000-krát 200 000":    {"1\u00a0000-krát", "200\u00a0000"},
		"4\u00a0300 eur":        {"4\u00a0300", "eur"},
		"3 1000 metrov":         {"3", "1000", "metrov"},
		"od 8 do 12 hodín, 9 5": {"od", "8", "do", "12", "hodín,", "9", "5"},
	}
	for in, want := range cases {
		var got []string
		for _, tok := range Parse(in).Paragraphs[0] {
			got = append(got, tok.Text)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

// These are the exact paragraphs and positions used in the two READMEs.
func TestREADMEExamples(t *testing.T) {
	cases := []struct {
		text, pass string
		positions  []Position
	}{
		{strings.Split(article, "\n\n")[0], "S7, .", []Position{{1, 18, 1}, {1, 24, 3}, {1, 11, 7}, {1, 7, 6}, {1, 17, 5}}},
		{"The council approved 86 thousand euros to repair the old mill, which dates from 1874. Work starts soon.", "W7, .", []Position{{1, 16, 1}, {1, 15, 3}, {1, 11, 5}, {1, 5, 9}, {1, 15, 5}}},
	}
	for i, tc := range cases {
		text := Parse(tc.text)
		ends := printedEnds(t, text)
		if got, err := text.Reveal(tc.positions, ends); err != nil || got != tc.pass {
			t.Errorf("README: got %q, %v; want %q", got, err, tc.pass)
		}
		file := "../../../README_sk.md"
		heading := "#### Ukrytie passphrase v texte"
		pattern := `(\d+)\. znak hesla:\s+(\d+)\. odsek,\s+(\d+)\. slovo,\s+(\d+)\. znak slova`
		if i == 1 {
			file = "../../../README.md"
			heading = "#### Hiding the passphrase in a text"
			pattern = `passphrase character (\d+):\s+paragraph (\d+), word (\d+), character (\d+) of the word`
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		section := strings.SplitN(string(data), heading, 2)[1]
		var quoted []string
		for _, line := range strings.Split(section, "\n") {
			if strings.HasPrefix(line, "> ") {
				quoted = append(quoted, line[2:])
			} else if len(quoted) > 0 {
				break
			}
		}
		if strings.Join(quoted, " ") != tc.text {
			t.Fatalf("%s paragraph differs from tested example", file)
		}
		rows := regexp.MustCompile(pattern).FindAllStringSubmatch(section, -1)
		if len(rows) != len(tc.positions) {
			t.Fatalf("%s has %d map rows", file, len(rows))
		}
		for j, row := range rows {
			var nums [4]int
			for k := range nums {
				nums[k], _ = strconv.Atoi(row[k+1])
			}
			if nums[0] != j+1 || (Position{nums[1], nums[2], nums[3]}) != tc.positions[j] {
				t.Errorf("%s row differs: %v", file, row)
			}
		}
	}

}

func sameLine(int, int) bool { return false }

const article = `Obecné zastupiteľstvo minulý týždeň schválilo 86 tisíc eur na opravu budovy, ktorú miestni odjakživa volajú starý mlyn. Stavba pri potoku pamätá ešte rok 1874 a posledné roky chátrala.

Práce sa začnú v apríli. Robotníci najskôr vymenia 240 metrov štvorcových strechy a vyčistia náhon, ktorý zarástol bazou a trnkami. Podľa stavbyvedúceho potrvajú približne 4 mesiace, ak im neprekazí leto plné búrok.

V mlynici zostalo pôvodné zariadenie aj s dvoma mlynskými kameňmi, každý váži vyše 600 kilogramov. Starý bicykel opretý o múr patril poslednému mlynárovi, ktorý tu pracoval 56 rokov. Farár ho vraj ešte pamätá, ako cez dedinu vozil vrecia múky.

Múzeum otvoria na hody 7. septembra. Sprievod vyjde ako vždy o 9:30 od zvonice na námestí a skončí pri mlyne, kde bude pre prvých 300 návštevníkov guláš zadarmo. Vstup do mlyna bude už iba za dobrovoľný príspevok.`

func TestHideReadsBack(t *testing.T) {
	text := Parse(article)
	r := rand.New(rand.NewPCG(1, 2))
	for _, pass := range []string{"c769rhb25tg8e3Fa", "aaaaaaaa", "9", "mesto1874"} {
		for i := 0; i < 50; i++ {
			ps, err := text.Hide(pass, r.IntN, sameLine)
			if err != nil {
				t.Fatalf("%q: %v", pass, err)
			}
			if got, err := text.Reveal(ps, sameLine); err != nil || got != pass {
				t.Fatalf("%q read back as %q (%v) from %v", pass, got, err, ps)
			}
		}
	}
}

func TestRepeatedCharactersGetDifferentWords(t *testing.T) {
	text := Parse("a b a c a")
	first := func(int) int { return 0 }
	ps, err := text.Hide("aaa", first, sameLine)
	if err != nil {
		t.Fatal(err)
	}
	if ps[0] == ps[1] || ps[1] == ps[2] || ps[0] == ps[2] {
		t.Errorf("three a's and three words starting with a, yet a word repeats: %v", ps)
	}
	// With more repeats than words, a word has to be used twice, and that is fine.
	if _, err := text.Hide("aaaa", first, sameLine); err != nil {
		t.Errorf("four a's in a text with three: %v", err)
	}
}

func TestHideRefusesWhatCannotBeWritten(t *testing.T) {
	text := Parse("Alfa beta 7 dní")
	pick := func(int) int { return 0 }

	var inv *InvalidError
	if _, err := text.Hide("aBa č\t€č", pick, sameLine); !errors.As(err, &inv) || string(inv.Chars) != "č\t€" {
		t.Errorf("non-ASCII and controls: got %v", err)
	}
	var miss *MissingError
	if _, err := text.Hide("qa9qx", pick, sameLine); !errors.As(err, &miss) || string(miss.Chars) != "9qx" {
		t.Errorf("characters without usable positions: got %v", err)
	}
	if _, err := text.Hide("", pick, sameLine); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty passphrase: got %v", err)
	}
	if _, err := Parse(" \n\n ").Hide("a", pick, sameLine); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty text: got %v", err)
	}
}

func TestRevealRefusesPositionsOutsideTheText(t *testing.T) {
	text := Parse("Alfa beta\n\nGama")
	for _, ps := range [][]Position{{{0, 1, 1}}, {{3, 1, 1}}, {{1, 3, 1}}, {{2, 0, 1}}, {{1, 1, 0}}, {{1, 1, 99}}} {
		if _, err := text.Reveal(ps, sameLine); err == nil {
			t.Errorf("%v should not read back", ps)
		}
	}
}

func TestEveryPrintableASCIICharacter(t *testing.T) {
	var body, pass strings.Builder
	for r := rune(32); r <= 126; r++ {
		pass.WriteRune(r)
		if r != ' ' {
			body.WriteString("a" + string(r) + "b ")
		}
	}
	text := Parse(body.String())
	ends := printedEnds(t, text)
	ps, err := text.Hide(pass.String(), func(int) int { return 0 }, ends)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := text.Reveal(ps, ends); err != nil || got != pass.String() {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestSpacesAndMarksRespectPrintedLines(t *testing.T) {
	text := Parse("a - / b 4 300 c\n\nd")
	ends := func(p, token int) bool { return p == 1 || token == 3 || token == 5 }
	// Positions count marks and spaces after a, but skip standalone marks as words.
	ps := []Position{{1, 1, 2}, {1, 1, 3}, {1, 1, 4}, {1, 1, 5}, {1, 1, 6}, {1, 3, 3}}
	if got, err := text.Reveal(ps, ends); err != nil || got != " - / 3" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, pos := range []Position{{1, 2, 2}, {1, 3, 2}, {1, 4, 2}, {2, 1, 2}} {
		if _, err := text.Reveal([]Position{pos}, ends); err == nil {
			t.Errorf("invisible space accepted: %v", pos)
		}
	}
	ps, err := text.Hide(strings.Repeat(" ", 20), func(int) int { return 0 }, ends)
	if err != nil {
		t.Fatal(err)
	}
	for _, pos := range ps {
		if pos.Word != 1 && pos.Word != 3 || pos.Paragraph != 1 {
			t.Errorf("bad space: %v", pos)
		}
	}
	// Without layout information, no inter-token character may be selected.
	for _, pass := range []string{" ", "-", "/"} {
		if _, err := text.Hide(pass, func(int) int { return 0 }, nil); err == nil {
			t.Errorf("selected %q without layout", pass)
		}
	}
	// A mark at the end of a line is visible, but a mark on the next line is not reachable.
	markText := Parse("a - b")
	if got, err := markText.Reveal([]Position{{1, 1, 3}}, func(_, i int) bool { return i == 1 }); err != nil || got != "-" {
		t.Fatalf("line-final mark: %q %v", got, err)
	}
	if _, err := markText.Reveal([]Position{{1, 1, 3}}, func(_, i int) bool { return i == 0 }); err == nil {
		t.Fatal("crossed line to a mark")
	}
}

func TestDigraphsStopCountingWithoutFoldingAccents(t *testing.T) {
	for _, word := range []string{"chX", "CHX", "cHX", "dzX", "DZX", "džX", "DŽX", "aChX", "aDzX", "aDžX", "chXe\u0301"} {
		text := Parse(word + " next")
		prefix := 1
		if strings.HasPrefix(word, "a") {
			prefix = 2
		}
		for c := 1; c <= prefix; c++ {
			if _, err := text.Reveal([]Position{{1, 1, c}}, sameLine); err != nil {
				t.Errorf("safe prefix in %q: %v", word, err)
			}
		}
		for c := prefix + 1; c <= len([]rune(word))+1; c++ {
			if _, err := text.Reveal([]Position{{1, 1, c}}, sameLine); err == nil {
				t.Errorf("counted through digraph in %q at %d", word, c)
			}
		}
		if _, err := text.Hide("X", func(int) int { return 0 }, sameLine); err == nil {
			t.Errorf("selected after digraph in %q", word)
		}
	}
	text := Parse("čA")
	if got, err := text.Reveal([]Position{{1, 1, 2}}, sameLine); err != nil || got != "A" {
		t.Fatalf("rune count: %q %v", got, err)
	}
	if _, err := text.Hide("c", func(int) int { return 0 }, sameLine); err == nil {
		t.Fatal("accent folded")
	}
}

func TestTypographyNormalizedBeforeCounting(t *testing.T) {
	got := paragraphTexts(Parse("„One“ ”two” ‚three‘ ’four’ – — end…"))
	want := []string{`"One" "two" 'three' 'four' - - end...`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPassphraseSpacesArePreserved(t *testing.T) {
	text := Parse("A B")
	for _, pass := range []string{" A ", " ", "  "} {
		ps, err := text.Hide(pass, func(int) int { return 0 }, sameLine)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := text.Reveal(ps, sameLine); err != nil || got != pass {
			t.Errorf("%q became %q: %v", pass, got, err)
		}
	}
}

func printedEnds(t *testing.T, text Text) func(int, int) bool {
	t.Helper()
	paras := make([][]string, len(text.Paragraphs))
	for p, tokens := range text.Paragraphs {
		for _, tok := range tokens {
			paras[p] = append(paras[p], tok.Text)
		}
	}
	ends, err := pdf.LineEnds(nil, paras)
	if err != nil {
		t.Fatal(err)
	}
	return func(p, i int) bool { return ends[p][i] }
}

func TestRepeatedCharactersUseDifferentPositionsInOneWord(t *testing.T) {
	text := Parse("aaaa")
	ps, err := text.Hide("aaaaa", func(int) int { return 0 }, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if ps[i].Character != i+1 {
			t.Fatalf("position repeated while others were free: %v", ps)
		}
	}
	if ps[4] != ps[0] {
		t.Fatalf("expected reuse after exhaustion: %v", ps)
	}
}

func TestDecomposedAccentsAreNotPlainASCII(t *testing.T) {
	text := Parse("cafe\u0301X")
	if got, err := text.Reveal([]Position{{1, 1, 3}}, nil); err != nil || got != "f" {
		t.Fatalf("safe prefix: %q %v", got, err)
	}
	for _, pass := range []string{"e", "X"} {
		if _, err := text.Hide(pass, func(int) int { return 0 }, nil); err == nil {
			t.Errorf("selected %q in a decomposed accented word", pass)
		}
	}
}
