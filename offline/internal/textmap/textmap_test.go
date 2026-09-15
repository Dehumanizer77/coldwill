package textmap

import (
	"errors"
	"math/rand/v2"
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
	type w struct {
		Text    string
		Initial string
	}
	var words []w
	var marks []string
	for _, tok := range p {
		if tok.Word {
			words = append(words, w{tok.Text, string(tok.Initial)})
		} else {
			marks = append(marks, tok.Text)
		}
	}
	want := []w{
		{"„Mlyn“", "m"}, {"7.", "7"}, {"septembra", "s"}, {"o", "o"}, {"9:30", "9"},
		{"prišlo", "p"}, {"4\u00a0300", "4"}, {"ľudí", "l"}, {"(1874)", "1"},
		{"Chata,", "c"}, {"e-mail", "e"}, {"20", "2"},
	}
	if !reflect.DeepEqual(words, want) {
		t.Errorf("words\n got %q\nwant %q", words, want)
	}
	if !reflect.DeepEqual(marks, []string{"–", "%"}) {
		t.Errorf("marks that do not count: got %q", marks)
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

func TestAccentsFoldToPlainLetters(t *testing.T) {
	for in, want := range map[rune]byte{'č': 'c', 'Ľ': 'l', 'ô': 'o', 'ř': 'r', 'ů': 'u', 'Ž': 'z', 'ä': 'a', 'ě': 'e', 'Ť': 't', 'A': 'a', '7': '7'} {
		if got := fold(in); got != want {
			t.Errorf("fold(%q) = %q, want %q", in, got, want)
		}
	}
	if got := fold('ж'); got != 0 {
		t.Errorf("a letter with no plain form should give 0, got %q", got)
	}
	for r, b := range folds {
		if b < 'a' || b > 'z' {
			t.Errorf("%q folds to %q, not a plain letter", r, b)
		}
	}
}

// Both READMEs walk through these paragraphs by hand. If the code counted them
// any other way, the documentation would teach the heir to count wrong.
func TestREADMEExamples(t *testing.T) {
	sk := "Prvý odsek.\n\nDruhý odsek.\n\n" +
		"V mlynici zostalo pôvodné zariadenie aj s dvoma mlynskými kameňmi, každý váži\n" +
		"vyše 600 kilogramov. Starý bicykel opretý o múr patril poslednému mlynárovi,\n" +
		"ktorý tu pracoval 56 rokov."
	if got, err := Parse(sk).Reveal([]Position{{3, 14}, {3, 17}, {3, 25}, {3, 27}}); err != nil || got != "6bt5" {
		t.Errorf("Slovak README example: got %q, %v; want 6bt5", got, err)
	}
	en := "One.\n\nTwo.\n\n" +
		"In the old mill the original machinery survived, including two millstones\n" +
		"weighing over 600 kilograms each. A bicycle leaning against the wall belonged\n" +
		"to the last miller, who worked there for 56 years."
	if got, err := Parse(en).Reveal([]Position{{3, 14}, {3, 18}, {3, 32}}); err != nil || got != "6b5" {
		t.Errorf("English README example: got %q, %v; want 6b5", got, err)
	}
}

const article = `Obecné zastupiteľstvo minulý týždeň schválilo 86 tisíc eur na opravu budovy, ktorú miestni odjakživa volajú starý mlyn. Stavba pri potoku pamätá ešte rok 1874 a posledné roky chátrala.

Práce sa začnú v apríli. Robotníci najskôr vymenia 240 metrov štvorcových strechy a vyčistia náhon, ktorý zarástol bazou a trnkami. Podľa stavbyvedúceho potrvajú približne 4 mesiace, ak im neprekazí leto plné búrok.

V mlynici zostalo pôvodné zariadenie aj s dvoma mlynskými kameňmi, každý váži vyše 600 kilogramov. Starý bicykel opretý o múr patril poslednému mlynárovi, ktorý tu pracoval 56 rokov. Farár ho vraj ešte pamätá, ako cez dedinu vozil vrecia múky.

Múzeum otvoria na hody 7. septembra. Sprievod vyjde ako vždy o 9:30 od zvonice na námestí a skončí pri mlyne, kde bude pre prvých 300 návštevníkov guláš zadarmo. Vstup do mlyna bude už iba za dobrovoľný príspevok.`

func TestHideReadsBack(t *testing.T) {
	text := Parse(article)
	r := rand.New(rand.NewPCG(1, 2))
	for _, pass := range []string{"c769rhb25tg8e3fa", "aaaaaaaa", "9", "mesto1874"} {
		for i := 0; i < 50; i++ {
			ps, err := text.Hide(pass, r.IntN)
			if err != nil {
				t.Fatalf("%q: %v", pass, err)
			}
			if got, err := text.Reveal(ps); err != nil || got != pass {
				t.Fatalf("%q read back as %q (%v) from %v", pass, got, err, ps)
			}
		}
	}
}

func TestRepeatedCharactersGetDifferentWords(t *testing.T) {
	text := Parse("a b a c a")
	first := func(int) int { return 0 }
	ps, err := text.Hide("aaa", first)
	if err != nil {
		t.Fatal(err)
	}
	if ps[0] == ps[1] || ps[1] == ps[2] || ps[0] == ps[2] {
		t.Errorf("three a's and three words starting with a, yet a word repeats: %v", ps)
	}
	// With more repeats than words, a word has to be used twice, and that is fine.
	if _, err := text.Hide("aaaa", first); err != nil {
		t.Errorf("four a's in a text with three: %v", err)
	}
}

func TestHideRefusesWhatCannotBeWritten(t *testing.T) {
	text := Parse("Alfa beta 7 dní")
	pick := func(int) int { return 0 }

	var inv *InvalidError
	if _, err := text.Hide("aBa č", pick); !errors.As(err, &inv) || string(inv.Chars) != "B č" {
		t.Errorf("uppercase, space and accent: got %v", err)
	}
	var miss *MissingError
	if _, err := text.Hide("qa9qx", pick); !errors.As(err, &miss) || string(miss.Chars) != "9qx" {
		t.Errorf("characters no word starts with: got %v", err)
	}
	if _, err := text.Hide("", pick); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty passphrase: got %v", err)
	}
	if _, err := Parse(" \n\n ").Hide("a", pick); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty text: got %v", err)
	}
}

func TestRevealRefusesPositionsOutsideTheText(t *testing.T) {
	text := Parse("Alfa beta\n\nGama")
	for _, ps := range [][]Position{{{0, 1}}, {{3, 1}}, {{1, 3}}, {{2, 0}}} {
		if _, err := text.Reveal(ps); err == nil {
			t.Errorf("%v should not read back", ps)
		}
	}
}
