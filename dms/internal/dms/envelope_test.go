package dms

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeEnvelope(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("-----BEGIN PGP MESSAGE-----\n"+name+"\n-----END PGP MESSAGE-----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// Two envelopes with different contents and different recipients: the split
// case — one person gets the passphrase, another gets the rest.
func newTwoEnvelopeSvc(t *testing.T) (*Service, *fakeMailer, *clk) {
	t.Helper()
	cfg := testConfig(t)
	dir := t.TempDir()
	cfg.EnvelopePath, cfg.FriendEmail, cfg.FriendSignal = "", "", ""
	cfg.Envelopes = []Envelope{
		{ID: "passphrase", Path: writeEnvelope(t, dir, "pass.asc"),
			Subject: "Dedičstvo — passphrase",
			To:      []EnvelopeRecipient{{Name: "Prvá", Email: "prva@example.com"}}},
		{ID: "pristupy", Path: writeEnvelope(t, dir, "acc.asc"),
			Note: "Toto sú ostatné prístupy.",
			To:   []EnvelopeRecipient{{Name: "Druhá", Email: "druha@example.com"}}},
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatalf("config: %v", err)
	}
	c, fm := newClock(), &fakeMailer{}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	return svc, fm, c
}

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

func TestEachEnvelopeGoesToItsOwnRecipient(t *testing.T) {
	svc, fm, c := newTwoEnvelopeSvc(t)
	fireIt(t, svc, c)

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired", svc.Phase())
	}
	pass, ok := fm.sentTo("prva@example.com", "passphrase")
	if !ok {
		t.Fatalf("passphrase envelope not sent to its recipient")
	}
	if !strings.Contains(pass.body, "pass.asc") {
		t.Errorf("wrong ciphertext in the passphrase envelope")
	}
	acc, ok := fm.sentTo("druha@example.com", "obálka")
	if !ok {
		t.Fatalf("accounts envelope not sent to its recipient")
	}
	if !strings.Contains(acc.body, "acc.asc") {
		t.Errorf("wrong ciphertext in the accounts envelope")
	}
	if !strings.Contains(acc.body, "Toto sú ostatné prístupy.") {
		t.Errorf("envelope note missing from the body")
	}
	// Neither recipient may receive the other's envelope.
	if _, wrong := fm.sentTo("prva@example.com", "obálka"); wrong {
		t.Errorf("first recipient also got the second envelope")
	}
}

// The same envelope to several people: redundancy, so one unreachable person
// does not sink the whole path.
func TestOneEnvelopeManyRecipients(t *testing.T) {
	cfg := testConfig(t)
	dir := t.TempDir()
	cfg.EnvelopePath, cfg.FriendEmail = "", ""
	cfg.Envelopes = []Envelope{{ID: "passphrase", Path: writeEnvelope(t, dir, "pass.asc"),
		To: []EnvelopeRecipient{
			{Email: "prva@example.com"}, {Email: "druha@example.com"}, {Email: "tretia@example.com"},
		}}}
	cfg.applyDefaults()
	c, fm := newClock(), &fakeMailer{}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	fireIt(t, svc, c)

	for _, to := range []string{"prva@example.com", "druha@example.com", "tretia@example.com"} {
		if _, ok := fm.sentTo(to, "obálka"); !ok {
			t.Errorf("%s did not get the envelope", to)
		}
	}
}

// A missing second envelope must block the whole firing, not send half of it.
func TestMissingSecondEnvelopeBlocksFiring(t *testing.T) {
	svc, fm, c := newTwoEnvelopeSvc(t)
	_ = os.Remove(svc.cfg.Envelopes[1].Path)
	fireIt(t, svc, c)

	if svc.Phase() == PhaseFired {
		t.Fatalf("fired with an unreadable envelope")
	}
	if _, ok := fm.sentTo("prva@example.com", "passphrase"); ok {
		t.Errorf("sent an envelope while the service was unhealthy")
	}
	if fm.countSubj("PORUCHA") == 0 {
		t.Errorf("expected a fault alert naming the broken envelope")
	}
}

// Partial delivery: what got out is remembered, the rest is retried, and
// nobody receives the same envelope twice.
func TestPartialDeliveryRetriesOnlyTheRest(t *testing.T) {
	svc, fm, c := newTwoEnvelopeSvc(t)
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}

	// A missing file is a fault and blocks firing entirely, so a partial fire
	// can only come from delivery: the second recipient's mailbox rejects.
	fm.reset()
	fm.failTo = map[string]bool{"druha@example.com": true}
	c.add(7 * 24 * time.Hour)
	svc.Tick()

	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want still counting down after a partial delivery", svc.Phase())
	}
	if _, ok := fm.sentTo("prva@example.com", "passphrase"); !ok {
		t.Fatalf("the readable envelope was not delivered")
	}
	if fm.countSubj("nepodarilo") == 0 {
		t.Errorf("owner was not told about the failed envelope")
	}

	// Their mailbox recovers: the next tick delivers only what is left.
	fm.failTo = nil
	fm.reset()
	svc.Tick()

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired once everything is out", svc.Phase())
	}
	if _, ok := fm.sentTo("druha@example.com", "obálka"); !ok {
		t.Errorf("the retried envelope was not delivered")
	}
	if _, again := fm.sentTo("prva@example.com", "passphrase"); again {
		t.Errorf("the already delivered envelope was sent a second time")
	}
}

// A veto after a partial fire must forget what was delivered: if the owner
// later really dies, every envelope has to go out again.
func TestCheckInClearsPartialDelivery(t *testing.T) {
	svc, fm, c := newTwoEnvelopeSvc(t)
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	fm.failTo = map[string]bool{"druha@example.com": true}
	c.add(7 * 24 * time.Hour)
	svc.Tick() // partial: first envelope out, second not

	svc.CheckIn() // alive after all
	fm.failTo = nil
	fm.reset()

	fireIt(t, svc, c) // silence, confirm and countdown all over again
	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired", svc.Phase())
	}
	if _, ok := fm.sentTo("prva@example.com", "passphrase"); !ok {
		t.Errorf("after a veto the first envelope must be sent again")
	}
	if _, ok := fm.sentTo("druha@example.com", "obálka"); !ok {
		t.Errorf("second envelope not sent")
	}
}

// The old single-envelope config keeps working unchanged.
func TestLegacySingleEnvelopeConfig(t *testing.T) {
	cfg := testConfig(t) // envelope_path + friend_email, no envelopes[]
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatalf("legacy config rejected: %v", err)
	}
	if len(cfg.Envelopes) != 1 || cfg.Envelopes[0].ID != "default" {
		t.Fatalf("legacy config folded into %#v", cfg.Envelopes)
	}
	if to := cfg.Envelopes[0].To; len(to) != 1 || to[0].Email != "friend@example.com" {
		t.Fatalf("legacy recipient = %#v", to)
	}
}

func TestEnvelopeValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"no envelope at all", func(c *Config) { c.EnvelopePath, c.FriendEmail = "", "" }, "at least one envelope"},
		{"duplicate id", func(c *Config) {
			c.Envelopes = []Envelope{
				{ID: "a", Path: "/x", To: []EnvelopeRecipient{{Email: "a@b"}}},
				{ID: "a", Path: "/y", To: []EnvelopeRecipient{{Email: "c@d"}}},
			}
		}, "duplicate envelope id"},
		{"no recipient", func(c *Config) {
			c.Envelopes = []Envelope{{ID: "a", Path: "/x"}}
		}, "at least one recipient"},
		{"recipient with no address", func(c *Config) {
			c.Envelopes = []Envelope{{ID: "a", Path: "/x", To: []EnvelopeRecipient{{Name: "X"}}}}
		}, "neither email nor signal"},
		{"no path", func(c *Config) {
			c.Envelopes = []Envelope{{ID: "a", To: []EnvelopeRecipient{{Email: "a@b"}}}}
		}, "path required"},
		{"signal number without transport", func(c *Config) {
			c.Envelopes = []Envelope{{ID: "a", Path: "/x", To: []EnvelopeRecipient{{Signal: "+15550000001"}}}}
		}, "api_url"},
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
