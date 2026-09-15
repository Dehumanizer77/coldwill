package server

import (
	"bytes"
	"encoding/base64"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

const textmapArticle = `Obecné zastupiteľstvo minulý týždeň schválilo 86 tisíc eur na opravu budovy, ktorú miestni odjakživa volajú starý mlyn. Stavba pri potoku pamätá ešte rok 1874 a posledné roky chátrala.

Práce sa začnú v apríli. Robotníci najskôr vymenia 240 metrov štvorcových strechy a vyčistia náhon, ktorý zarástol bazou a trnkami. Podľa stavbyvedúceho potrvajú približne 4 mesiace, ak im neprekazí leto plné búrok.

V mlynici zostalo pôvodné zariadenie aj s dvoma mlynskými kameňmi, každý váži vyše 600 kilogramov. Starý bicykel opretý o múr patril poslednému mlynárovi, ktorý tu pracoval 56 rokov. Farár ho vraj ešte pamätá, ako cez dedinu vozil vrecia múky.

Múzeum otvoria na hody 7. septembra. Sprievod vyjde ako vždy o 9:30 od zvonice na námestí a skončí pri mlyne, kde bude pre prvých 300 návštevníkov guláš zadarmo. Vstup do mlyna bude už iba za dobrovoľný príspevok.`

func TestTextmapFormIsLinkedAndRenders(t *testing.T) {
	s := newTestServer(t)
	if body := do(s, http.MethodGet, "/", nil).Body.String(); !strings.Contains(body, `href="/textmap?lang=en"`) {
		t.Errorf("the index does not link the passphrase page")
	}
	rr := do(s, http.MethodGet, "/textmap?lang=sk", nil)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "Passphrase v texte") {
		t.Fatalf("form missing (status %d)", rr.Code)
	}
}

// The page's whole promise: the map it hands out, read by hand against the
// text, gives the passphrase back, and the passphrase itself is nowhere on the
// page.
func TestTextmapMapReadsBackByHand(t *testing.T) {
	s := newTestServer(t)
	const passphrase = " S7, .@_+/- "
	rr := do(s, http.MethodPost, "/textmap", url.Values{
		"lang": {"sk"}, "headline": {"Starý mlyn sa dočká opravy"},
		"text": {textmapArticle + "\n\n„Kontakt:“ info_box@example.test +123 / — suma 4 300 eur…"}, "passphrase": {passphrase},
	})
	body := rr.Body.String()
	if rr.Code != 200 || !strings.Contains(body, "✓ Kontrola") {
		t.Fatalf("no map (status %d): %s", rr.Code, firstLine(body))
	}

	lines := regexp.MustCompile(`(\d+)\. znak hesla:\s+(\d+)\. odsek,\s+(\d+)\. slovo,\s+(\d+)\. znak slova`).FindAllStringSubmatch(body, -1)
	if len(lines) != len(passphrase) {
		t.Fatalf("%d map lines, want %d", len(lines), len(passphrase))
	}
	// Read the normalized preview returned by the page, without using textmap.
	preview := regexp.MustCompile(`<ol class="paras">([\s\S]*?)</ol>`).FindStringSubmatch(body)
	if preview == nil {
		t.Fatal("no paragraph preview")
	}
	paras := regexp.MustCompile(`<li>(.*?)</li>`).FindAllStringSubmatch(preview[1], -1)
	var got strings.Builder
	for i, m := range lines {
		n, _ := strconv.Atoi(m[1])
		p, _ := strconv.Atoi(m[2])
		w, _ := strconv.Atoi(m[3])
		if n != i+1 {
			t.Errorf("line %d is numbered %d", i+1, n)
		}
		c, _ := strconv.Atoi(m[4])
		tokens := strings.Split(html.UnescapeString(paras[p-1][1]), " ")
		word := 0
		for j, tok := range tokens {
			if strings.IndexFunc(tok, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) < 0 {
				continue
			}
			word++
			if word != w {
				continue
			}
			// Count from the word through any following standalone marks and spaces.
			chars := []rune(strings.Join(tokens[j:], " "))
			if c < 1 || c > len(chars) {
				t.Fatalf("invalid character %d", c)
			}
			got.WriteRune(chars[c-1])
			break
		}
	}
	if got.String() != passphrase {
		t.Errorf("read by hand: %q, want %q", got.String(), passphrase)
	}
	for _, want := range []string{"nadpis sa nepočíta", "„4 300“", "&#34;Kontakt:&#34;", "suma 4\u00a0300 eur..."} {
		if !strings.Contains(body, want) {
			t.Errorf("map is missing %q", want)
		}
	}

	link := regexp.MustCompile(`href="data:application/pdf;base64,([^"]+)"`).FindStringSubmatch(body)
	if link == nil {
		t.Fatalf("no PDF download")
	}
	doc, err := base64.StdEncoding.DecodeString(html.UnescapeString(link[1]))
	if err != nil || !bytes.HasPrefix(doc, []byte("%PDF-")) {
		t.Errorf("the download is not a PDF (%v)", err)
	}
	if strings.Contains(strings.Replace(body, link[0], "", 1), passphrase) {
		t.Errorf("the passphrase appears on the page")
	}
}

func TestTextmapKeepsTheTextButNotThePassphraseAfterAMistake(t *testing.T) {
	s := newTestServer(t)
	body := do(s, http.MethodPost, "/textmap", url.Values{
		"lang": {"sk"}, "headline": {"Nadpis"}, "text": {"Ahoj svet, 7 dní."}, "passphrase": {"ahoj9"},
	}).Body.String()
	for _, want := range []string{
		"V texte nie je použiteľná vytlačená pozícia", "&#34;9&#34;, &#34;a&#34;",
		`value="Nadpis"`, "Ahoj svet, 7 dní.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("error page is missing %q", want)
		}
	}
	if strings.Contains(body, "ahoj9") {
		t.Errorf("the passphrase was written back into the form")
	}
}

func TestTextmapRefusesWhatCannotBeWrittenOrPrinted(t *testing.T) {
	s := newTestServer(t)
	body := do(s, http.MethodPost, "/textmap", url.Values{
		"text": {textmapArticle}, "passphrase": {"Mlyn č€"},
	}).Body.String()
	if !strings.Contains(body, "printable ASCII") || !strings.Contains(body, "&#34;č&#34;, &#34;€&#34;") {
		t.Errorf("non-ASCII should be refused and named: %s", body)
	}

	body = do(s, http.MethodPost, "/textmap", url.Values{
		"text": {"Ahoj 🙂 svet"}, "passphrase": {"as"},
	}).Body.String()
	if !strings.Contains(body, "cannot be printed") {
		t.Errorf("an emoji should be refused before a PDF is made")
	}
}

func TestTextmapNormalizesHeadlineAndPreservesSpaceOnlyPassphrase(t *testing.T) {
	s := newTestServer(t)
	body := do(s, http.MethodPost, "/textmap", url.Values{
		"headline": {"„Title“ — end…"}, "text": {"A B"}, "passphrase": {"  "},
	}).Body.String()
	for _, want := range []string{"✓ Check:", "(2 characters)", "&#34;Title&#34; - end...", "character  2 of the word"} {
		if !strings.Contains(body, want) {
			t.Errorf("result is missing %q", want)
		}
	}
}
