package dms

import (
	"testing"
	"time"
)

// The heir's mailbox refusing the envelope is not a release: the switch stays
// in the countdown, tells the owner, and tries again on the next tick.
func TestRejectedHeirMailboxIsRetried(t *testing.T) {
	svc, fm, c := newSvc(t)
	fm.failTo = map[string]bool{"heir@example.com": true}
	fireIt(t, svc, c)

	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want still counting down", svc.Phase())
	}
	if fm.countSubj("could not be sent") == 0 {
		t.Errorf("the owner was not told the envelope did not go out")
	}

	fm.failTo = nil
	fm.reset()
	svc.Tick()
	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired once the mailbox accepts it", svc.Phase())
	}
	if _, ok := fm.sentTo("heir@example.com", "inheritance"); !ok {
		t.Errorf("the retried envelope did not reach the heir")
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
		if _, ok := fm.sentTo(to, "confirmation"); !ok {
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
