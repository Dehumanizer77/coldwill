package dms

import (
	"testing"
	"time"
)

// A relay that rejects one address must not cost the others their copy: an
// envelope with two recipients has to reach the second one even when the first
// is refused.
func TestRejectedRecipientDoesNotBlockTheRest(t *testing.T) {
	cfg := testConfig(t)
	dir := t.TempDir()
	cfg.EnvelopePath, cfg.FriendEmail = "", ""
	cfg.Envelopes = []Envelope{{ID: "passphrase", Path: writeEnvelope(t, dir, "pass.asc"),
		To: []EnvelopeRecipient{
			{Email: "odmietnuta@example.com"},
			{Email: "prijata@example.com"},
			{Email: "tretia@example.com"},
		}}}
	cfg.applyDefaults()
	c, fm := newClock(), &fakeMailer{failTo: map[string]bool{"odmietnuta@example.com": true}}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	fireIt(t, svc, c)

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired — one bad address must not stop the firing", svc.Phase())
	}
	for _, to := range []string{"prijata@example.com", "tretia@example.com"} {
		if _, ok := fm.sentTo(to, "obálka"); !ok {
			t.Errorf("%s did not get the envelope", to)
		}
	}
	if _, ok := fm.sentTo("odmietnuta@example.com", "obálka"); ok {
		t.Errorf("the rejected address should not be recorded as delivered")
	}
}

// The same for the owner's own notifications: a broken owner mailbox must not
// stop the confirmers from being asked.
func TestRejectedOwnerDoesNotBlockConfirmers(t *testing.T) {
	cfg := testConfig(t)
	cfg.applyDefaults()
	c, fm := newClock(), &fakeMailer{failTo: map[string]bool{"me@example": true}}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()

	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s, want awaiting", svc.Phase())
	}
	for _, to := range []string{"friend@example.com", "brother@example.com"} {
		if _, ok := fm.sentTo(to, "potvrdenie"); !ok {
			t.Errorf("confirmer %s was not asked", to)
		}
	}
}

// Every address is its own SMTP transaction, so one message to three people is
// three sends — that is what makes a single rejection survivable.
func TestOneTransactionPerRecipient(t *testing.T) {
	cfg := testConfig(t)
	cfg.applyDefaults()
	c, fm := newClock(), &fakeMailer{}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	fm.reset()
	svc.Tick() // asks both confirmers

	fm.mu.Lock()
	defer fm.mu.Unlock()
	for _, s := range fm.sent {
		if len(s.to) != 1 {
			t.Errorf("send %q went to %d addresses at once: %v", s.subject, len(s.to), s.to)
		}
	}
	if len(fm.sent) < 2 {
		t.Errorf("expected at least one send per confirmer, got %d", len(fm.sent))
	}
}
