package dms

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// The finding: a confirmation link kept from an earlier waiting cycle could be
// replayed in a later one without the confirmer agreeing again.
func TestOldConfirmLinkIsRefusedInTheNextCycle(t *testing.T) {
	svc, _, c := newSvc(t)
	h := svc.Handler()

	// First cycle: silence, confirmers asked, link captured.
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s, want awaiting", svc.Phase())
	}
	oldCycle := svc.currentCycle()
	if oldCycle == "" {
		t.Fatal("no cycle id was generated for the waiting cycle")
	}
	oldToken := svc.token(svc.confirmAction("friend", oldCycle))

	// The owner turns up alive; the cycle ends.
	svc.CheckIn()
	if svc.currentCycle() != "" {
		t.Errorf("check-in did not clear the cycle id")
	}
	if rr := httpDo(h, "GET", "/confirm?id=friend&token="+oldToken, nil); rr.Code != 403 {
		t.Errorf("old confirm link still works right after a check-in: %d", rr.Code)
	}

	// Second cycle, months later: the old link must still be refused.
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("phase = %s, want awaiting again", svc.Phase())
	}
	if svc.currentCycle() == oldCycle {
		t.Fatal("the second cycle reused the first cycle's id")
	}
	if rr := httpDo(h, "GET", "/confirm?id=friend&token="+oldToken, nil); rr.Code != 403 {
		t.Fatalf("GET with a replayed confirm token: %d, want 403", rr.Code)
	}
	if rr := httpDo(h, "POST", "/confirm", url.Values{"id": {"friend"}, "token": {oldToken}}); rr.Code != 403 {
		t.Fatalf("POST with a replayed confirm token: %d, want 403", rr.Code)
	}
	if svc.Phase() != PhaseAwaiting {
		t.Fatalf("a replayed token moved the phase to %s", svc.Phase())
	}

	// The link issued for this cycle does work.
	fresh := svc.token(svc.confirmAction("friend", svc.currentCycle()))
	if rr := httpDo(h, "POST", "/confirm", url.Values{"id": {"friend"}, "token": {fresh}}); rr.Code != 200 {
		t.Fatalf("current confirm link rejected: %d", rr.Code)
	}
	if svc.Phase() != PhaseCountdown {
		t.Fatalf("phase = %s, want countdown", svc.Phase())
	}
}

// The link that goes out in the e-mail is the one that works.
func TestConfirmLinkFromTheEmailWorks(t *testing.T) {
	svc, fm, c := newSvc(t)
	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()

	m, ok := fm.sentTo("friend@example.com", "confirmation")
	if !ok {
		t.Fatal("confirmer was not asked")
	}
	i := strings.Index(m.body, "/confirm?")
	if i < 0 {
		t.Fatalf("no confirm link in the message:\n%s", m.body)
	}
	link := strings.Fields(m.body[i:])[0]
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	rr := httpDo(svc.Handler(), "GET", u.RequestURI(), nil)
	if rr.Code != 200 {
		t.Fatalf("the emailed confirm link returned %d", rr.Code)
	}
}

// Bumping checkin_key_version revokes a leaked check-in URL.
func TestCheckinKeyVersionRevokesOldLinks(t *testing.T) {
	cfg := testConfig(t)
	cfg.applyDefaults()
	c, fm := newClock(), &fakeMailer{}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	old := svc.token(svc.checkinAction())
	if rr := httpDo(svc.Handler(), "GET", "/checkin?token="+old, nil); rr.Code != 200 {
		t.Fatalf("check-in link should work before rotation: %d", rr.Code)
	}

	// The owner suspects the link leaked: bump the version and restart.
	cfg.CheckinKeyVersion++
	if err := cfg.validate(); err != nil {
		t.Fatalf("bumped config invalid: %v", err)
	}
	svc2, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rr := httpDo(svc2.Handler(), "GET", "/checkin?token="+old, nil); rr.Code != 403 {
		t.Errorf("leaked check-in link still works after rotation: %d", rr.Code)
	}
	fresh := svc2.token(svc2.checkinAction())
	if fresh == old {
		t.Fatal("rotation did not change the token")
	}
	if rr := httpDo(svc2.Handler(), "POST", "/checkin", url.Values{"token": {fresh}}); rr.Code != 200 {
		t.Errorf("new check-in link does not work: %d", rr.Code)
	}
}

func TestNegativeCheckinKeyVersionRejected(t *testing.T) {
	cfg := testConfig(t)
	cfg.CheckinKeyVersion = -1
	cfg.applyDefaults()
	if err := cfg.validate(); err == nil || !strings.Contains(err.Error(), "checkin_key_version") {
		t.Fatalf("err = %v, want it to reject a negative checkin_key_version", err)
	}
}
