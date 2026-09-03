package server

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"inh/offline/internal/slip39"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	// Templates live in offline/web; tests run with CWD = this package dir.
	tmpl, err := template.ParseGlob("../../web/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	return New(tmpl, []byte("/* css */"))
}

func do(s *Server, method, path string, form url.Values) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func TestSelfTest(t *testing.T) {
	if err := SelfTest(); err != nil {
		t.Fatalf("SelfTest: %v", err)
	}
}

func TestIndex(t *testing.T) {
	s := newTestServer(t)
	rr := do(s, http.MethodGet, "/", nil)
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Nový backup") {
		t.Fatalf("index missing expected content")
	}
}

func TestSetupPost(t *testing.T) {
	s := newTestServer(t)
	rr := do(s, http.MethodPost, "/setup", url.Values{"threshold": {"2"}, "count": {"3"}})
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{"Self-test", "Prvá časť", "Druhá časť", "Tretia časť", "Stiahnuť key-file"} {
		if !strings.Contains(body, want) {
			t.Errorf("setup result missing %q", want)
		}
	}
	if !strings.Contains(body, "✓ Self-test") {
		t.Errorf("setup self-test did not pass")
	}
}

// TestRecoverRoundTrip drives a full setup→recover loop through the HTTP layer:
// generate shares, POST two of them, and confirm the recovered key hex matches.
func TestRecoverRoundTrip(t *testing.T) {
	s := newTestServer(t)
	key := []byte("RoundTripKey_16!") // 16 bytes
	shares, err := slip39.Generate(key, 2, 3, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	rr := do(s, http.MethodPost, "/recover", url.Values{
		"mnemonics": {shares[1] + "\n" + shares[2]},
	})
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Kľúč obnovený") {
		t.Fatalf("recover did not succeed: %s", firstLine(body))
	}
	wantHex := "526f756e64547269704b65795f313621" // hex of the 16-byte key
	if !strings.Contains(body, wantHex) {
		t.Fatalf("recovered hex not found in response")
	}
	if !strings.Contains(body, "1. krok z 3") {
		t.Errorf("recover result missing the chain explanation")
	}
}

func TestRecoverError(t *testing.T) {
	s := newTestServer(t)
	rr := do(s, http.MethodPost, "/recover", url.Values{"mnemonics": {"these are not valid slip39 words at all"}})
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Chyba") {
		t.Fatalf("expected error page")
	}
}

func TestRunbookForm(t *testing.T) {
	s := newTestServer(t)
	rr := do(s, http.MethodGet, "/runbook", nil)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "návod pre rodinu") {
		t.Fatalf("runbook form missing (status %d)", rr.Code)
	}
}

func TestRunbookRender(t *testing.T) {
	s := newTestServer(t)
	rr := do(s, http.MethodPost, "/runbook", url.Values{
		"person_name":    {"Alica", "Bob"},
		"person_contact": {"a@example.com", "b@example.com"},
		"person_tech":    {"tech", "nontech"},
		"wife":           {"Jana"},
		"bank":           {"Banka XY, schranka 42"},
		"threshold":      {"2"}, "count": {"3"},
	})
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"neobsahuje žiadne tajomstvá", "Technicky zdatné osoby", "Alica",
		"Ostatní držitelia častí", "Bob", "drží: Alica", "Jana",
		"Banka XY, schranka 42", "window.print()", "23 slov",
		`name="edit" value="1"`, // hidden edit form carries data back
	} {
		if !strings.Contains(body, want) {
			t.Errorf("runbook missing %q", want)
		}
	}
}

// TestRunbookEdit posts with edit=1 and checks the form comes back pre-filled.
func TestRunbookEdit(t *testing.T) {
	s := newTestServer(t)
	rr := do(s, http.MethodPost, "/runbook", url.Values{
		"edit":           {"1"},
		"person_name":    {"Alica", "Bob"},
		"person_contact": {"a@example.com", "b@example.com"},
		"person_tech":    {"tech", "nontech"},
		"wife":           {"Jana"},
		"threshold":      {"2"}, "count": {"2"},
	})
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Vygenerovať runbook", // it's the form, not the result
		`value="Alica"`, `value="Bob"`, `value="Jana"`,
		`<option value="tech" selected>`, // Alica's tech flag preserved
	} {
		if !strings.Contains(body, want) {
			t.Errorf("edit form missing %q", want)
		}
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// The key-file download is a data: URL, and html/template escapes "+" in an
// attribute (Ww8p1/&#43;Gqn3...). A browser undoes that when it parses the
// attribute, so the saved file is byte-exact — this test pins that down, because
// a scraper that skips entity decoding silently gets a different key-file and
// the .kdbx then refuses to open.
func TestKeyFileDownloadSurvivesAttributeEscaping(t *testing.T) {
	s := newTestServer(t)
	// sha256("0")[:20] — its base64 (X+zrZv/IbzjZUnhsbWlsecLbwjk=) has both
	// "+" and "/", the two characters that make this fail if it can fail.
	key, err := hex.DecodeString("5feceb66ffc86f38d952786c6d696c79c2dbc239")
	if err != nil {
		t.Fatal(err)
	}
	shares, err := slip39.Generate(key, 2, 3, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	rr := do(s, http.MethodPost, "/recover", url.Values{"mnemonics": {shares[0] + "\n" + shares[1]}})
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	body := rr.Body.String()

	m := regexp.MustCompile(`href="data:application/octet-stream;base64,([^"]+)"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no key-file download link in the recover result")
	}
	if !strings.Contains(m[1], "&#43;") {
		t.Errorf("expected the escaped attribute form; got %q", m[1])
	}
	got, err := base64.StdEncoding.DecodeString(html.UnescapeString(m[1]))
	if err != nil {
		t.Fatalf("decode as a browser would: %v", err)
	}
	if !bytes.Equal(got, key) {
		t.Errorf("downloaded key-file = %x, want %x", got, key)
	}
	if len(got) != 20 {
		t.Errorf("key-file is %d B, want 20 (160-bit)", len(got))
	}
}
