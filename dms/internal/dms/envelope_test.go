package dms

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fireIt walks a service through silence, a confirmation and the countdown to
// the tick that releases the envelope.
func fireIt(t *testing.T, svc *Service, c *clk) {
	t.Helper()
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	c.add(7 * 24 * time.Hour)
	svc.Tick()
}

// The envelope goes to the primary heir and to nobody else: not the confirmers,
// who only attest, and not the owner.
func TestEnvelopeGoesOnlyToTheHeir(t *testing.T) {
	svc, fm, c := newSvc(t)
	fireIt(t, svc, c)

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired", svc.Phase())
	}
	m, ok := fm.sentTo("heir@example.com", "inheritance")
	if !ok {
		t.Fatalf("the heir did not get the envelope")
	}
	if len(m.atts) != 1 || m.atts[0].Name != "envelope.pdf" || string(m.atts[0].Data) != testPDF {
		t.Errorf("envelope not attached as the PDF from envelope_path: %+v", m.atts)
	}
	if strings.Contains(m.body, "%PDF") {
		t.Errorf("the PDF was pasted into the body instead of attached")
	}
	for _, other := range []string{"friend@example.com", "brother@example.com", "me@example"} {
		if _, ok := fm.sentTo(other, "inheritance"); ok {
			t.Errorf("%s received the envelope", other)
		}
	}
}

// A file that is not a PDF is a fault, so the wrong file is caught while the
// owner can still replace it, not on the day it goes out.
func TestEnvelopeThatIsNotAPDFIsAFault(t *testing.T) {
	svc, fm, c := newSvc(t)
	if err := os.WriteFile(svc.cfg.EnvelopePath, []byte("passphrase character 1: paragraph 1, word 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fireIt(t, svc, c)

	if svc.Phase() == PhaseFired {
		t.Fatalf("fired with an envelope that is not a PDF")
	}
	if fm.countSubj("FAULT") == 0 {
		t.Errorf("expected a fault alert about the envelope")
	}
	if _, ok := fm.sentTo("heir@example.com", "inheritance"); ok {
		t.Errorf("the wrong file was sent to the heir")
	}
}

// A check-in after a delivery that failed leaves nothing behind: if the owner
// later really dies, the envelope goes out in full.
func TestCheckInAfterFailedDeliverySendsItAgain(t *testing.T) {
	svc, fm, c := newSvc(t)
	fm.failTo = map[string]bool{"heir@example.com": true}
	fireIt(t, svc, c)
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want still counting down after a failed delivery", svc.Phase())
	}

	svc.CheckIn() // alive after all
	fm.failTo = nil
	fm.reset()

	fireIt(t, svc, c) // silence, confirmation and countdown all over again
	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired", svc.Phase())
	}
	if _, ok := fm.sentTo("heir@example.com", "inheritance"); !ok {
		t.Errorf("the envelope was not sent on the second release")
	}
}

// Configs from when envelopes could go to anyone name other recipients.
// Starting one would silently drop them, so it is refused instead.
func TestRetiredEnvelopeKeysAreRejected(t *testing.T) {
	example, err := os.ReadFile("../../config.example.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"envelopes", "friend_email", "friend_signal"} {
		t.Run(key, func(t *testing.T) {
			var raw map[string]any
			if err := json.Unmarshal(example, &raw); err != nil {
				t.Fatal(err)
			}
			raw[key] = "x"
			b, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(p, b, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err = LoadConfig(p)
			if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "heir") {
				t.Fatalf("err = %v, want a refusal naming %s and pointing at heir", err, key)
			}
		})
	}
}

func TestEnvelopeValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"no envelope", func(c *Config) { c.EnvelopePath = "" }, "envelope_path required"},
		{"no heir", func(c *Config) { c.Heir = Heir{} }, "heir needs"},
		{"heir with a name only", func(c *Config) { c.Heir = Heir{Name: "Heir"} }, "heir needs"},
		{"heir number without transport", func(c *Config) { c.Heir.Signal = "+15550000004" }, "api_url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig(t)
			tc.mutate(&cfg)
			cfg.applyDefaults()
			err := cfg.validate()
			if err == nil {
				t.Fatalf("want an error mentioning %q, got none", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}
