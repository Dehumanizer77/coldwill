package dms

import (
	"strings"
	"testing"
	"time"
)

func awaiting(t *testing.T, svc *Service, c *clk) {
	t.Helper()
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s, want awaiting", svc.Phase())
	}
}

func newQuorumSvc(t *testing.T, quorum int) (*Service, *fakeMailer, *clk) {
	t.Helper()
	cfg := testConfig(t)
	cfg.ConfirmQuorum = quorum
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

// Default behaviour is unchanged: one confirmer starts the countdown.
func TestQuorumOneIsTheDefault(t *testing.T) {
	cfg := testConfig(t)
	cfg.applyDefaults()
	if cfg.ConfirmQuorum != 1 {
		t.Fatalf("default quorum = %d, want 1", cfg.ConfirmQuorum)
	}
	svc, _, c := newQuorumSvc(t, 1)
	awaiting(t, svc, c)
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want countdown from a single confirmation", svc.Phase())
	}
}

// With a quorum of two, one attestation is not enough — and the same person
// confirming twice is still one person.
func TestQuorumTwoNeedsTwoDifferentPeople(t *testing.T) {
	svc, fm, c := newQuorumSvc(t, 2)
	awaiting(t, svc, c)
	fm.reset()

	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s — one confirmation must not start the countdown", svc.Phase())
	}
	if got, need := svc.Confirmations(); got != 1 || need != 2 {
		t.Fatalf("confirmations = %d/%d, want 1/2", got, need)
	}
	if fm.countSubj("waiting for more") == 0 {
		t.Errorf("owner was not told about the partial confirmation")
	}

	if err := svc.Confirm("friend"); err != nil { // same person again
		t.Fatal(err)
	}
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s — the same confirmer twice must not count twice", svc.Phase())
	}

	if err := svc.Confirm("brother"); err != nil {
		t.Fatal(err)
	}
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want countdown once the quorum is met", svc.Phase())
	}
}

// A veto clears the collected attestations: the next cycle starts from zero.
func TestCheckInClearsConfirmations(t *testing.T) {
	svc, _, c := newQuorumSvc(t, 2)
	awaiting(t, svc, c)
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	svc.CheckIn()
	if got, _ := svc.Confirmations(); got != 0 {
		t.Fatalf("confirmations after a check-in = %d, want 0", got)
	}

	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s — the earlier confirmation must not carry over", svc.Phase())
	}
}

// A quorum nobody could ever reach would strand the inheritance.
func TestQuorumAboveConfirmerCountRejected(t *testing.T) {
	cfg := testConfig(t) // two confirmers
	cfg.ConfirmQuorum = 3
	cfg.applyDefaults()
	err := cfg.validate()
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err = %v, want a refusal of an unreachable quorum", err)
	}
	cfg.ConfirmQuorum = -1
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "at least 1") {
		t.Fatalf("err = %v, want a refusal of a non-positive quorum", err)
	}
}
