// Package server implements the offline tool's localhost web UI: a Setup wizard
// (generate key-file + 2-of-3 SLIP-39 shares) and a Recovery wizard (combine
// shares back into the key-file). It performs NO network access and holds no
// state between requests. Intended to run only on a loopback address on an
// air-gapped machine.
package server

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"inh/offline/internal/i18n"
	"inh/offline/internal/slip39"
)

// keyBytes is the size of the generated key-file (160-bit -> 23-word shares).
const keyBytes = 20

// ProjectURL is where the heir downloads the tool: the latest release, whose
// address stays the same as new ones are published. It is the default for the
// runbook's download field and can be overridden there, since anyone who forks
// this project serves the binaries from their own repository.
const ProjectURL = "https://github.com/Dehumanizer77/inh/releases/latest"

type Server struct {
	mux  *http.ServeMux
	tmpl *template.Template
	css  []byte
}

// TemplateFuncs are the helpers the templates need. Exported so the binary and
// the tests parse the templates exactly the same way.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		// inc turns a 0-based range index into a human 1-based label.
		"inc": func(i int) int { return i + 1 },
		// t looks a message up in the catalogue for the page's language.
		"t": i18n.T,
		// ts is the same as plain text, for script and attribute contexts where
		// html/template does its own contextual escaping.
		"ts":       i18n.S,
		"langs":    i18n.Languages,
		"langName": i18n.Name,
	}
}

// page carries what every template needs regardless of what it shows. Embedded
// in each page's data so that {{t .L "key"}} works everywhere.
type page struct {
	L i18n.Lang
}

// lang reads the language from the request. The offline tool has no cookies and
// no state, so it travels in the URL and in form fields.
func (s *Server) lang(r *http.Request) i18n.Lang {
	if v := r.FormValue("lang"); v != "" {
		return i18n.Parse(v)
	}
	return i18n.Parse(r.URL.Query().Get("lang"))
}

func New(tmpl *template.Template, css []byte) *Server {
	s := &Server{mux: http.NewServeMux(), tmpl: tmpl, css: css}
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/setup", s.handleSetup)
	s.mux.HandleFunc("/recover", s.handleRecover)
	s.mux.HandleFunc("/runbook", s.handleRunbook)
	s.mux.HandleFunc("/style.css", s.handleCSS)
	s.mux.HandleFunc("/wordlist.js", s.handleWordlistJS)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// SelfTest verifies the SLIP-39 implementation at startup: a known-answer test
// against an official vector plus a generate/combine round-trip. The tool
// refuses to start if this fails.
func SelfTest() error {
	const knownMnemonic = "duckling enlarge academic academic agency result length solution fridge kidney coal piece deal husband erode duke ajar critical decision keyboard"
	const wantHex = "bb54aac4b89dc868ba37d9cc21b2cece"
	got, err := slip39.Combine([]string{knownMnemonic}, []byte("TREZOR"))
	if err != nil {
		return fmt.Errorf("known-answer test: %w", err)
	}
	if hex.EncodeToString(got) != wantHex {
		return fmt.Errorf("known-answer test: got %x, want %s", got, wantHex)
	}
	key := make([]byte, keyBytes)
	for i := range key {
		key[i] = byte(i*7 + 1)
	}
	shares, err := slip39.Generate(key, 2, 3, nil)
	if err != nil {
		return fmt.Errorf("round-trip generate: %w", err)
	}
	rk, err := slip39.Combine([]string{shares[0], shares[2]}, nil)
	if err != nil {
		return fmt.Errorf("round-trip combine: %w", err)
	}
	if !bytes.Equal(rk, key) {
		return fmt.Errorf("round-trip mismatch")
	}
	return nil
}

func (s *Server) handleCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Write(s.css)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "index.html", page{L: s.lang(r)})
}

// wordView splits a word into the four letters that are engraved and the rest,
// so the setup page can show exactly what goes on the plate.
type wordView struct{ Head, Tail string }

type shareView struct {
	Holder string
	Words  []wordView
}

type setupData struct {
	page
	Threshold     int
	Count         int
	Shares        []shareView
	WordsPerShare int
	KeyHex        string
	KeyURL        template.URL
	Verified      bool
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	l := s.lang(r)
	if r.Method != http.MethodPost {
		s.render(w, "setup.html", page{L: l})
		return
	}
	threshold := atoiDefault(r.FormValue("threshold"), 2)
	count := atoiDefault(r.FormValue("count"), 3)

	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		s.renderErr(w, l, "err.rand", err)
		return
	}
	mnems, err := slip39.Generate(key, threshold, count, nil)
	if err != nil {
		s.renderErr(w, l, "err.params", err)
		return
	}

	// Self-check: the first `threshold` shares must rebuild the key.
	verified := false
	if threshold >= 1 && threshold <= len(mnems) {
		rk, e := slip39.Combine(mnems[:threshold], nil)
		verified = e == nil && bytes.Equal(rk, key)
	}

	holders := holderLabels(l, count)
	shares := make([]shareView, len(mnems))
	wps := 0
	for i, m := range mnems {
		words := strings.Fields(m)
		wv := make([]wordView, len(words))
		for j, w := range words {
			// Metal plates only fit four letters; show which four they are.
			wv[j] = wordView{Head: w[:slip39.PrefixLen], Tail: w[slip39.PrefixLen:]}
		}
		shares[i] = shareView{Holder: holders[i], Words: wv}
		wps = len(words)
	}

	s.render(w, "setup_result.html", setupData{
		page:          page{L: l},
		Threshold:     threshold,
		Count:         count,
		Shares:        shares,
		WordsPerShare: wps,
		KeyHex:        hex.EncodeToString(key),
		KeyURL:        keyDataURL(key),
		Verified:      verified,
	})
}

// recoverForm drives the word grid. The fields are rendered server-side so the
// page works with JavaScript disabled; the script only adds completion, focus
// jumps and extra parts on top of them.
type recoverForm struct {
	page
	Parts        []int
	Words        []int
	WordsPerPart int
}

func newRecoverForm(l i18n.Lang, parts, words int) recoverForm {
	f := recoverForm{page: page{L: l}, Parts: make([]int, parts), Words: make([]int, words), WordsPerPart: words}
	for i := range f.Parts {
		f.Parts[i] = i
	}
	for i := range f.Words {
		f.Words[i] = i
	}
	return f
}

type recoverData struct {
	page
	NumShares int
	KeyHex    string
	KeyURL    template.URL
}

func (s *Server) handleRecover(w http.ResponseWriter, r *http.Request) {
	l := s.lang(r)
	if r.Method != http.MethodPost {
		// Two parts of 23 words is what this tool produces by default; the page
		// can add parts and switch the length.
		s.render(w, "recover.html", newRecoverForm(l, 2, 23))
		return
	}
	lines := collectParts(r)
	if len(lines) == 0 {
		s.renderErr(w, l, "err.noshares")
		return
	}
	// Plates carry four-letter abbreviations, so that is what people type in.
	// Expanding here means the person recovering never has to know that the
	// engraved words are shortened at all.
	lines, err := slip39.NormalizeMnemonics(lines)
	if err != nil {
		s.renderErr(w, l, "err.words", wordProblem(l, err))
		return
	}
	key, err := slip39.Combine(lines, nil)
	if err != nil {
		s.renderErr(w, l, "err.combine", err)
		return
	}
	s.render(w, "recover_result.html", recoverData{
		page:      page{L: l},
		NumShares: len(lines),
		KeyHex:    hex.EncodeToString(key),
		KeyURL:    keyDataURL(key),
	})
}

// collectParts reads the parts either from the per-word inputs (p0w0, p0w1, …)
// or from the paste-everything textarea, whichever the person used.
func collectParts(r *http.Request) []string {
	var lines []string
	for p := 0; p < maxParts; p++ {
		var words []string
		for i := 0; i < maxWordsPerPart; i++ {
			if v := strings.TrimSpace(r.FormValue(fmt.Sprintf("p%dw%d", p, i))); v != "" {
				words = append(words, v)
			}
		}
		if len(words) > 0 {
			lines = append(lines, strings.Join(words, " "))
		}
	}
	for _, ln := range strings.Split(r.FormValue("mnemonics"), "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			lines = append(lines, t)
		}
	}
	return lines
}

// Generous bounds for the recovery form; SLIP-39 mnemonics are 20-33 words.
const (
	maxParts        = 16
	maxWordsPerPart = 40
)

func (s *Server) handleWordlistJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Write([]byte("window.SLIP39_WORDS=["))
	for i, word := range slip39.Wordlist() {
		if i > 0 {
			w.Write([]byte(","))
		}
		fmt.Fprintf(w, "%q", word)
	}
	fmt.Fprintf(w, "];window.SLIP39_PREFIX=%d;", slip39.PrefixLen)
}

// wordProblem phrases a slip39 word error for the reader. The crypto package
// reports what went wrong; which language to say it in is this layer's business.
func wordProblem(l i18n.Lang, err error) string {
	var we *slip39.WordError
	if !errors.As(err, &we) {
		if errors.Is(err, slip39.ErrNoWords) {
			return i18n.S(l, "word.empty")
		}
		return err.Error()
	}
	where := i18n.S(l, "word.at", we.Index)
	if we.Part > 0 {
		where = i18n.S(l, "word.at.part", we.Part, we.Index)
	}
	switch {
	case we.TooShort:
		return where + " " + i18n.S(l, "word.short", we.Word, slip39.PrefixLen)
	case we.Suggestion != "":
		return where + " " + i18n.S(l, "word.typo", we.Word, slip39.PrefixLen, we.Suggestion)
	default:
		return where + " " + i18n.S(l, "word.unknown", we.Word)
	}
}

// Person is one trusted party in the runbook. Tech marks whether they are
// technically skilled (whom the family calls / who receives the envelope) vs a
// non-technical holder.
type Person struct {
	Name    string
	Contact string
	Tech    bool
}

type partLoc struct {
	Num    int
	Holder string
}

// runbookForm pre-fills the runbook form (empty defaults on GET, or the
// previously-entered values when the user clicks "Upraviť" on a result).
type runbookForm struct {
	page
	Author, Date, Wife       string
	Persons                  []Person // every row, in order
	Threshold, Count         int
	Bank, KdbxCopies         string
	ToolWhere, ToolURL       string // where inh-offline is kept, and where to download it
	WalletNotes, FamilyNotes string
}

type runbookData struct {
	page
	Author, Date, Wife       string
	AllPersons               []Person // every row, for the hidden "Upraviť" form
	TechPersons              []Person
	OtherPersons             []Person
	TechNames                string // všetky technicky zdatné osoby, na vypísanie v texte
	Parts                    []partLoc
	Threshold, Count         int
	Bank, KdbxCopies         string
	ToolWhere, ToolURL       string
	WalletNotes, FamilyNotes string

	// Parenthetical asides that only appear when there is something to say.
	// Built here rather than in the template so the message stays one sentence
	// in the catalogue instead of being assembled from fragments.
	TechSuffix, ToolSuffix string
}

// joinNames renders a list of people the way a sentence needs it: "A",
// "A alebo B", "A, B alebo C". The heir should see everyone she can call, not
// just whoever happened to be entered first.
func joinNames(l i18n.Lang, names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + i18n.S(l, "names.or") + names[len(names)-1]
}

// parsePersonRows reads the variable-length person rows. Every row submits a
// name/contact/tech triple (text inputs and selects always submit), so the
// three slices align by index; part i is held by row i.
func parsePersonRows(r *http.Request) []Person {
	names := r.Form["person_name"]
	contacts := r.Form["person_contact"]
	techs := r.Form["person_tech"]
	n := len(names)
	if len(contacts) > n {
		n = len(contacts)
	}
	if len(techs) > n {
		n = len(techs)
	}
	rows := make([]Person, 0, n)
	for i := 0; i < n; i++ {
		var p Person
		if i < len(names) {
			p.Name = strings.TrimSpace(names[i])
		}
		if i < len(contacts) {
			p.Contact = strings.TrimSpace(contacts[i])
		}
		p.Tech = i < len(techs) && techs[i] == "tech"
		rows = append(rows, p)
	}
	return rows
}

// handleRunbook renders the personalized family instruction sheet. It contains
// NO secrets (no shares, key, passphrase or seed) — only the map of where
// things are and the recovery procedure. The user prints it to PDF.
//
// GET → empty form. POST with edit=1 → form pre-filled with the submitted
// values (the result page's "Upraviť" button). POST otherwise → the result.
func (s *Server) handleRunbook(w http.ResponseWriter, r *http.Request) {
	l := s.lang(r)
	if r.Method != http.MethodPost {
		s.render(w, "runbook_form.html", runbookForm{page: page{L: l}, Threshold: 2, Count: 3, ToolURL: ProjectURL})
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderErr(w, l, "err.form", err)
		return
	}
	f := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	rows := parsePersonRows(r)
	threshold := atoiDefault(f("threshold"), 2)
	count := atoiDefault(f("count"), 3)

	if r.FormValue("edit") == "1" {
		s.render(w, "runbook_form.html", runbookForm{
			page:   page{L: l},
			Author: f("author"), Date: f("date"), Wife: f("wife"),
			Persons: rows, Threshold: threshold, Count: count,
			Bank: f("bank"), KdbxCopies: f("kdbx_copies"),
			ToolWhere: f("tool_where"), ToolURL: toolURL(f("tool_url")),
			WalletNotes: f("wallet_notes"), FamilyNotes: f("family_notes"),
		})
		return
	}

	var tech, other []Person
	var parts []partLoc
	for i, p := range rows {
		parts = append(parts, partLoc{Num: i + 1, Holder: p.Name}) // part i is held by row i
		if p.Name == "" {
			continue
		}
		if p.Tech {
			tech = append(tech, p)
		} else {
			other = append(other, p)
		}
	}
	names := make([]string, 0, len(tech))
	for _, p := range tech {
		names = append(names, p.Name)
	}

	s.render(w, "runbook.html", runbookData{
		page:   page{L: l},
		Author: f("author"), Date: f("date"), Wife: f("wife"),
		AllPersons: rows, TechPersons: tech, OtherPersons: other, TechNames: joinNames(l, names),
		Parts:     parts,
		Threshold: threshold, Count: count,
		Bank: f("bank"), KdbxCopies: f("kdbx_copies"),
		ToolWhere: f("tool_where"), ToolURL: toolURL(f("tool_url")),
		WalletNotes: f("wallet_notes"), FamilyNotes: f("family_notes"),
		TechSuffix: aside(l, joinNames(l, names)),
		ToolSuffix: aside(l, f("tool_where")),
	})
}

// aside wraps a value in the language's parenthetical form, or returns nothing
// when there is no value, so the sentence around it reads correctly either way.
func aside(l i18n.Lang, v string) string {
	if v == "" {
		return ""
	}
	return i18n.S(l, "rb.aside", v)
}

// toolURL falls back to this project's own repository when the field is empty,
// so a runbook never goes to print without saying where to get the tool.
func toolURL(v string) string {
	if v == "" {
		return ProjectURL
	}
	return v
}

// ---- helpers ----

func keyDataURL(key []byte) template.URL {
	return template.URL("data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(key))
}

func holderLabels(l i18n.Lang, count int) []string {
	if count == 3 {
		return []string{i18n.S(l, "holder.1"), i18n.S(l, "holder.2"), i18n.S(l, "holder.3")}
	}
	out := make([]string, count)
	for i := range out {
		out[i] = i18n.S(l, "holder.n", i+1)
	}
	return out
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderErr(w http.ResponseWriter, l i18n.Lang, key string, args ...any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data := struct {
		page
		Message template.HTML
	}{page{L: l}, i18n.T(l, key, args...)}
	if err := s.tmpl.ExecuteTemplate(w, "error.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
