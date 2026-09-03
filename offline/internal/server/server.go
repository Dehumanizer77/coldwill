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
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"inh/offline/internal/slip39"
)

// keyBytes is the size of the generated key-file (160-bit -> 23-word shares).
const keyBytes = 20

type Server struct {
	mux  *http.ServeMux
	tmpl *template.Template
	css  []byte
}

func New(tmpl *template.Template, css []byte) *Server {
	s := &Server{mux: http.NewServeMux(), tmpl: tmpl, css: css}
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/setup", s.handleSetup)
	s.mux.HandleFunc("/recover", s.handleRecover)
	s.mux.HandleFunc("/runbook", s.handleRunbook)
	s.mux.HandleFunc("/style.css", s.handleCSS)
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
	s.render(w, "index.html", nil)
}

type shareView struct {
	Holder string
	Words  []string
}

type setupData struct {
	Threshold     int
	Count         int
	Shares        []shareView
	WordsPerShare int
	KeyHex        string
	KeyURL        template.URL
	Verified      bool
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.render(w, "setup.html", nil)
		return
	}
	threshold := atoiDefault(r.FormValue("threshold"), 2)
	count := atoiDefault(r.FormValue("count"), 3)

	key := make([]byte, keyBytes)
	if _, err := rand.Read(key); err != nil {
		s.renderErr(w, "Zlyhal generátor náhody: "+err.Error())
		return
	}
	mnems, err := slip39.Generate(key, threshold, count, nil)
	if err != nil {
		s.renderErr(w, "Neplatné parametre: "+err.Error())
		return
	}

	// Self-check: the first `threshold` shares must rebuild the key.
	verified := false
	if threshold >= 1 && threshold <= len(mnems) {
		rk, e := slip39.Combine(mnems[:threshold], nil)
		verified = e == nil && bytes.Equal(rk, key)
	}

	holders := holderLabels(count)
	shares := make([]shareView, len(mnems))
	wps := 0
	for i, m := range mnems {
		words := strings.Fields(m)
		shares[i] = shareView{Holder: holders[i], Words: words}
		wps = len(words)
	}

	s.render(w, "setup_result.html", setupData{
		Threshold:     threshold,
		Count:         count,
		Shares:        shares,
		WordsPerShare: wps,
		KeyHex:        hex.EncodeToString(key),
		KeyURL:        keyDataURL(key),
		Verified:      verified,
	})
}

type recoverData struct {
	NumShares int
	KeyHex    string
	KeyURL    template.URL
}

func (s *Server) handleRecover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.render(w, "recover.html", nil)
		return
	}
	var lines []string
	for _, ln := range strings.Split(r.FormValue("mnemonics"), "\n") {
		if t := strings.TrimSpace(ln); t != "" {
			lines = append(lines, t)
		}
	}
	if len(lines) == 0 {
		s.renderErr(w, "Nezadal si žiadny podiel.")
		return
	}
	key, err := slip39.Combine(lines, nil)
	if err != nil {
		s.renderErr(w, "Obnova zlyhala: "+err.Error()+" — skontroluj, či sú slová a počet podielov správne.")
		return
	}
	s.render(w, "recover_result.html", recoverData{
		NumShares: len(lines),
		KeyHex:    hex.EncodeToString(key),
		KeyURL:    keyDataURL(key),
	})
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
	Author, Date, Wife       string
	Persons                  []Person // every row, in order
	Threshold, Count         int
	Bank, KdbxCopies         string
	WalletNotes, FamilyNotes string
}

type runbookData struct {
	Author, Date, Wife       string
	AllPersons               []Person // every row, for the hidden "Upraviť" form
	TechPersons              []Person
	OtherPersons             []Person
	PrimaryHelper            string
	Parts                    []partLoc
	Threshold, Count         int
	Bank, KdbxCopies         string
	WalletNotes, FamilyNotes string
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
	if r.Method != http.MethodPost {
		s.render(w, "runbook_form.html", runbookForm{Threshold: 2, Count: 3})
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderErr(w, "Neplatný formulár: "+err.Error())
		return
	}
	f := func(k string) string { return strings.TrimSpace(r.FormValue(k)) }
	rows := parsePersonRows(r)
	threshold := atoiDefault(f("threshold"), 2)
	count := atoiDefault(f("count"), 3)

	if r.FormValue("edit") == "1" {
		s.render(w, "runbook_form.html", runbookForm{
			Author: f("author"), Date: f("date"), Wife: f("wife"),
			Persons: rows, Threshold: threshold, Count: count,
			Bank: f("bank"), KdbxCopies: f("kdbx_copies"),
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
	primary := ""
	if len(tech) > 0 {
		primary = tech[0].Name
	}

	s.render(w, "runbook.html", runbookData{
		Author: f("author"), Date: f("date"), Wife: f("wife"),
		AllPersons: rows, TechPersons: tech, OtherPersons: other, PrimaryHelper: primary,
		Parts:     parts,
		Threshold: threshold, Count: count,
		Bank: f("bank"), KdbxCopies: f("kdbx_copies"),
		WalletNotes: f("wallet_notes"), FamilyNotes: f("family_notes"),
	})
}

// ---- helpers ----

func keyDataURL(key []byte) template.URL {
	return template.URL("data:application/octet-stream;base64," + base64.StdEncoding.EncodeToString(key))
}

func holderLabels(count int) []string {
	if count == 3 {
		return []string{"Prvá časť", "Druhá časť", "Tretia časť"}
	}
	out := make([]string, count)
	for i := range out {
		out[i] = "Časť #" + strconv.Itoa(i+1)
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

func (s *Server) renderErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "error.html", struct{ Message string }{msg}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
