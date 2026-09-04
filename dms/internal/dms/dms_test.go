package dms

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type clk struct{ t time.Time }

func (c *clk) now() time.Time      { return c.t }
func (c *clk) add(d time.Duration) { c.t = c.t.Add(d) }

type sentMail struct {
	to            []string
	subject, body string
}

type fakeMailer struct {
	mu       sync.Mutex
	sent     []sentMail
	checkErr error
	sendErr  error
	failTo   map[string]bool // addresses whose delivery fails
}

func (m *fakeMailer) Send(to []string, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	for _, t := range to {
		if m.failTo[t] {
			return errors.New("mailbox unavailable: " + t)
		}
	}
	m.sent = append(m.sent, sentMail{to, subject, body})
	return nil
}
func (m *fakeMailer) Check() error { return m.checkErr }
func (m *fakeMailer) reset()       { m.mu.Lock(); m.sent = nil; m.mu.Unlock() }

func (m *fakeMailer) countSubj(sub string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, s := range m.sent {
		if strings.Contains(s.subject, sub) {
			n++
		}
	}
	return n
}

func (m *fakeMailer) sentTo(addr, subjContains string) (sentMail, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sent {
		if !strings.Contains(s.subject, subjContains) {
			continue
		}
		for _, t := range s.to {
			if t == addr {
				return s, true
			}
		}
	}
	return sentMail{}, false
}

// testConfig is an e-mail-only config over a fresh temp dir with a dummy
// envelope, in the single-envelope form. Callers apply defaults themselves
// after any edits, exactly as LoadConfig does.
func testConfig(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	env := filepath.Join(dir, "envelope.asc")
	if err := os.WriteFile(env, []byte("-----BEGIN PGP MESSAGE-----\nZHVtbXk=\n-----END PGP MESSAGE-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		ListenAddr: "127.0.0.1:0", PublicBaseURL: "https://dms.example",
		SMTPAddr: "127.0.0.1:25", FromEmail: "dms@example", UserEmail: "me@example",
		FriendEmail: "friend@example.com",
		Confirmers: []Confirmer{
			{ID: "friend", Name: "Friend", Email: "friend@example.com"},
			{ID: "brother", Name: "Brother", Email: "brother@example.com"},
		},
		EnvelopePath: env, StatePath: filepath.Join(dir, "state.json"),
		HMACSecret:         "0123456789abcdef-secret",
		CheckInInterval:    Duration(30 * 24 * time.Hour),
		ReminderInterval:   Duration(7 * 24 * time.Hour),
		SilenceThreshold:   Duration(60 * 24 * time.Hour),
		ReleaseDelay:       Duration(7 * 24 * time.Hour),
		HealthBeatInterval: Duration(7 * 24 * time.Hour),
		WarningInterval:    Duration(24 * time.Hour),
		AlertInterval:      Duration(24 * time.Hour),
		TickInterval:       Duration(time.Hour),
	}
	return cfg
}

func newClock() *clk { return &clk{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)} }

func newSvc(t *testing.T) (*Service, *fakeMailer, *clk) {
	t.Helper()
	c, fm := newClock(), &fakeMailer{}
	cfg := testConfig(t)
	cfg.applyDefaults() // same order as LoadConfig: defaults (incl. envelope folding) then use
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	return svc, fm, c
}

func TestStartupHealthBeat(t *testing.T) {
	svc, fm, _ := newSvc(t)
	svc.Tick()
	if fm.countSubj("v poriadku") == 0 {
		t.Errorf("expected a startup health beat")
	}
	if svc.Phase() != PhaseNormal {
		t.Errorf("phase = %s, want normal", svc.Phase())
	}
}

func TestReminderAt30d(t *testing.T) {
	svc, fm, c := newSvc(t)
	svc.Tick()
	fm.reset()
	c.add(31 * 24 * time.Hour)
	svc.Tick()
	if fm.countSubj("check-in") == 0 {
		t.Errorf("expected a check-in reminder after 31 days")
	}
	if svc.Phase() != PhaseNormal {
		t.Errorf("phase = %s, want normal", svc.Phase())
	}
}

func TestSilenceToAwaiting(t *testing.T) {
	svc, fm, c := newSvc(t)
	svc.Tick()
	fm.reset()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s, want awaiting", svc.Phase())
	}
	if _, ok := fm.sentTo("friend@example.com", "potvrdenie"); !ok {
		t.Errorf("confirmer friend not asked")
	}
	if _, ok := fm.sentTo("brother@example.com", "potvrdenie"); !ok {
		t.Errorf("confirmer brother not asked")
	}
}

func toAwaiting(t *testing.T, svc *Service, fm *fakeMailer, c *clk) {
	t.Helper()
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("setup: phase = %s, want awaiting", svc.Phase())
	}
	fm.reset()
}

func TestConfirmCountdownFire(t *testing.T) {
	svc, fm, c := newSvc(t)
	toAwaiting(t, svc, fm, c)

	if err := svc.Confirm("friend"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want countdown", svc.Phase())
	}
	fm.reset()
	svc.Tick() // first countdown warning, not yet fired
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("fired too early")
	}
	if fm.countSubj("čoskoro pošle") == 0 {
		t.Errorf("expected a countdown warning")
	}
	fm.reset()
	c.add(7 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired", svc.Phase())
	}
	m, ok := fm.sentTo("friend@example.com", "dedičstvo")
	if !ok {
		t.Fatalf("envelope not sent to friend")
	}
	if !strings.Contains(m.body, "BEGIN PGP MESSAGE") {
		t.Errorf("envelope body missing PGP ciphertext")
	}
}

func TestCheckinCancels(t *testing.T) {
	svc, fm, c := newSvc(t)
	toAwaiting(t, svc, fm, c)
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	svc.CheckIn() // user is alive after all
	if svc.Phase() != PhaseNormal {
		t.Fatalf("phase = %s, want normal after check-in", svc.Phase())
	}
	fm.reset()
	c.add(8 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() == PhaseFired {
		t.Fatalf("must not fire after check-in")
	}
	if _, ok := fm.sentTo("friend@example.com", "dedičstvo"); ok {
		t.Fatalf("envelope sent despite check-in")
	}
}

func TestFailSafeUnhealthyDoesNotFire(t *testing.T) {
	svc, fm, c := newSvc(t)
	toAwaiting(t, svc, fm, c)
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	fm.checkErr = errors.New("smtp down") // self-test will fail
	fm.reset()
	c.add(7 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() == PhaseFired {
		t.Fatalf("fired while unhealthy")
	}
	if fm.countSubj("PORUCHA") == 0 {
		t.Errorf("expected a fault alert")
	}
}

func TestEnvelopeMissingDoesNotFire(t *testing.T) {
	svc, fm, c := newSvc(t)
	toAwaiting(t, svc, fm, c)
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(svc.cfg.EnvelopePath)
	fm.reset()
	c.add(7 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() == PhaseFired {
		t.Fatalf("fired without a valid envelope")
	}
}

// ---- HTTP ----

func httpDo(h http.Handler, method, target string, form url.Values) *httptest.ResponseRecorder {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestCheckinHTTP(t *testing.T) {
	svc, _, _ := newSvc(t)
	h := svc.Handler()
	tok := svc.token("checkin")

	if rr := httpDo(h, "GET", "/checkin?token="+tok, nil); rr.Code != 200 || !strings.Contains(rr.Body.String(), "Som živý") {
		t.Fatalf("GET checkin: code %d", rr.Code)
	}
	if rr := httpDo(h, "GET", "/checkin?token=bad", nil); rr.Code != 403 {
		t.Fatalf("bad token should be 403, got %d", rr.Code)
	}
	if rr := httpDo(h, "POST", "/checkin", url.Values{"token": {tok}}); rr.Code != 200 || !strings.Contains(rr.Body.String(), "Ďakujem") {
		t.Fatalf("POST checkin: code %d", rr.Code)
	}
}

func TestConfirmHTTP(t *testing.T) {
	svc, fm, c := newSvc(t)
	toAwaiting(t, svc, fm, c)
	h := svc.Handler()
	tok := svc.token("confirm:friend")

	if rr := httpDo(h, "GET", "/confirm?id=friend&token="+tok, nil); rr.Code != 200 || !strings.Contains(rr.Body.String(), "Potvrdzujem") {
		t.Fatalf("GET confirm: code %d", rr.Code)
	}
	if rr := httpDo(h, "GET", "/confirm?id=friend&token=bad", nil); rr.Code != 403 {
		t.Fatalf("bad token should be 403, got %d", rr.Code)
	}
	if rr := httpDo(h, "POST", "/confirm", url.Values{"id": {"friend"}, "token": {tok}}); rr.Code != 200 {
		t.Fatalf("POST confirm: code %d", rr.Code)
	}
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want countdown after confirm", svc.Phase())
	}
}

// The shipped example is what a deployment starts from — it must load and pass
// validation as-is (only the secrets/addresses get replaced).
func TestExampleConfigLoads(t *testing.T) {
	cfg, err := LoadConfig("../../config.example.json")
	if err != nil {
		t.Fatalf("config.example.json: %v", err)
	}
	if !cfg.Signal.Enabled() {
		t.Errorf("example should demonstrate the Signal block")
	}
	if cfg.CheckInInterval.D() != 30*24*time.Hour {
		t.Errorf("check_in_interval = %s, want 30d", cfg.CheckInInterval.D())
	}
}
