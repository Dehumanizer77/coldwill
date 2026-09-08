package dms

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type sentSignal struct {
	to  []string
	msg string
}

type fakeSignal struct {
	mu       sync.Mutex
	sent     []sentSignal
	checkErr error
	sendErr  error
}

func (f *fakeSignal) Send(numbers []string, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, sentSignal{numbers, message})
	return nil
}
func (f *fakeSignal) Check() error { return f.checkErr }
func (f *fakeSignal) reset()       { f.mu.Lock(); f.sent = nil; f.mu.Unlock() }

func (f *fakeSignal) sentTo(number, contains string) (sentSignal, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.sent {
		if !strings.Contains(s.msg, contains) {
			continue
		}
		for _, n := range s.to {
			if n == number {
				return s, true
			}
		}
	}
	return sentSignal{}, false
}

const (
	numOwner   = "+15550000001"
	numFriend  = "+15550000002"
	numBrother = "+15550000003"
)

func newSignalSvc(t *testing.T) (*Service, *fakeMailer, *fakeSignal, *clk) {
	t.Helper()
	cfg := testConfig(t)
	cfg.UserSignal = numOwner
	cfg.FriendSignal = numFriend
	cfg.Confirmers[0].Signal = numFriend
	cfg.Confirmers[1].Signal = numBrother
	cfg.Signal = SignalConfig{APIURL: "http://127.0.0.1:8080", FromNumber: "+15550000000", Timeout: Duration(5 * time.Second)}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatalf("test config invalid: %v", err)
	}
	c, fm, fs := newClock(), &fakeMailer{}, &fakeSignal{}
	svc, err := New(cfg, c.now, fm, fs)
	if err != nil {
		t.Fatal(err)
	}
	return svc, fm, fs, c
}

// Every message to me goes out on both channels.
func TestSignalFanoutToOwner(t *testing.T) {
	svc, fm, fs, c := newSignalSvc(t)
	svc.Tick()
	fm.reset()
	fs.reset()

	c.add(31 * 24 * time.Hour)
	svc.Tick()

	if fm.countSubj("check in") == 0 {
		t.Errorf("no check-in reminder by e-mail")
	}
	if _, ok := fs.sentTo(numOwner, "check in"); !ok {
		t.Errorf("no check-in reminder on Signal")
	}
}

// Confirmers with a number get the request on Signal too — and the Signal text
// must carry the same one-click link as the mail.
func TestSignalConfirmersAsked(t *testing.T) {
	svc, _, fs, c := newSignalSvc(t)
	svc.Tick()
	fs.reset()

	c.add(61 * 24 * time.Hour)
	svc.Tick()

	m, ok := fs.sentTo(numBrother, "confirmation")
	if !ok {
		t.Fatalf("confirmer brother not asked on Signal")
	}
	if !strings.Contains(m.msg, "/confirm?id=brother") {
		t.Errorf("Signal confirm request lacks the confirm link: %q", m.msg)
	}
}

// A dead Signal must not jam the switch: it alerts on e-mail and fires anyway.
func TestSignalDownStillFires(t *testing.T) {
	svc, fm, fs, c := newSignalSvc(t)
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}

	fs.checkErr = errors.New("container down")
	fs.sendErr = errors.New("container down")
	fm.reset()
	c.add(7 * 24 * time.Hour)
	svc.Tick()

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired despite a broken Signal", svc.Phase())
	}
	if _, ok := fm.sentTo("friend@example.com", "inheritance"); !ok {
		t.Errorf("envelope not delivered by e-mail")
	}
	if fm.countSubj("Signal is not working") == 0 {
		t.Errorf("expected an e-mail alert about the broken Signal channel")
	}
	if fm.countSubj("FAULT") != 0 {
		t.Errorf("broken Signal must not count as a service fault")
	}
}

// The mirror case: mail is down at fire time but Signal gets the envelope out.
func TestSignalDeliversEnvelopeWhenMailSendFails(t *testing.T) {
	svc, fm, fs, c := newSignalSvc(t)
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}

	fm.sendErr = errors.New("relay refused") // Check() still passes: self-test stays healthy
	fs.reset()
	c.add(7 * 24 * time.Hour)
	svc.Tick()

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired via Signal", svc.Phase())
	}
	m, ok := fs.sentTo(numFriend, "BEGIN PGP MESSAGE")
	if !ok {
		t.Fatalf("envelope not delivered on Signal")
	}
	if !strings.Contains(m.msg, "inheritance") {
		t.Errorf("Signal envelope missing its subject line: %q", m.msg)
	}
}

// Nothing delivered anywhere = no fire, and it is retried on the next tick.
func TestNoChannelDeliversDoesNotFire(t *testing.T) {
	svc, fm, fs, c := newSignalSvc(t)
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}

	fm.sendErr = errors.New("relay refused")
	fs.sendErr = errors.New("container down")
	c.add(7 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want still counting down", svc.Phase())
	}

	fm.sendErr, fs.sendErr = nil, nil
	fm.reset()
	svc.Tick()
	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired on retry", svc.Phase())
	}
	if _, ok := fm.sentTo("friend@example.com", "inheritance"); !ok {
		t.Errorf("envelope not delivered on retry")
	}
}

// ---- SignalAPI (the real HTTP client) ----

func TestSignalAPISend(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/send" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"timestamp":"1"}`))
	}))
	defer srv.Close()

	api := NewSignalAPI(srv.URL+"/", "+15550000000", 5*time.Second)
	if err := api.Send([]string{numOwner}, "ahoj"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if got["message"] != "ahoj" || got["number"] != "+15550000000" {
		t.Errorf("bad payload: %v", got)
	}
	if r, _ := got["recipients"].([]any); len(r) != 1 || r[0] != numOwner {
		t.Errorf("bad recipients: %v", got["recipients"])
	}
}

func TestSignalAPISendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unregistered user"}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	err := NewSignalAPI(srv.URL, "+15550000000", 5*time.Second).Send([]string{numOwner}, "ahoj")
	if err == nil || !strings.Contains(err.Error(), "unregistered user") {
		t.Fatalf("err = %v, want the API message", err)
	}
}

func TestSignalAPICheck(t *testing.T) {
	accounts := `["+15550000000"]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/accounts" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(accounts))
	}))
	defer srv.Close()

	api := NewSignalAPI(srv.URL, "+15550000000", 5*time.Second)
	if err := api.Check(); err != nil {
		t.Fatalf("check: %v", err)
	}

	// An unlinked device is the realistic silent failure: the API is up, our
	// number is simply gone.
	accounts = `["+15550000009"]`
	if err := api.Check(); err == nil || !strings.Contains(err.Error(), "not linked") {
		t.Fatalf("err = %v, want a not-linked error", err)
	}
}

func TestSignalAPICheckUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // nothing listening
	if err := NewSignalAPI(srv.URL, "+15550000000", time.Second).Check(); err == nil {
		t.Fatal("want an error from an unreachable API")
	}
}

// ---- config validation ----

func TestSignalConfigValidation(t *testing.T) {
	full := SignalConfig{APIURL: "http://127.0.0.1:8080", FromNumber: "+15550000000"}
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"e-mail only", func(c *Config) {}, ""},
		{"complete", func(c *Config) { c.Signal, c.UserSignal = full, numOwner }, ""},
		{"number without transport", func(c *Config) { c.UserSignal = numOwner }, "api_url"},
		{"confirmer number without transport", func(c *Config) { c.Confirmers[0].Signal = numFriend }, "api_url"},
		{"transport without numbers", func(c *Config) { c.Signal = full }, "no recipient"},
		{"half transport", func(c *Config) { c.Signal = SignalConfig{APIURL: full.APIURL} }, "both api_url and from_number"},
		{"not E.164", func(c *Config) { c.Signal, c.UserSignal = full, "15550000001" }, "E.164"},
		{"sender not E.164", func(c *Config) {
			c.Signal, c.UserSignal = SignalConfig{APIURL: full.APIURL, FromNumber: "15550000000"}, numOwner
		}, "from_number"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig(t)
			tc.mutate(&cfg)
			cfg.applyDefaults()
			err := cfg.validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("want an error containing %q, got none", tc.wantErr)
			case tc.wantErr != "" && !strings.Contains(err.Error(), tc.wantErr):
				t.Fatalf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}
