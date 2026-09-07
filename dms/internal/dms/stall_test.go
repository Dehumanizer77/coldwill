package dms

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// deafSMTP accepts connections, greets, and then never answers again — the
// failure the audit describes. Without a deadline a send against it blocks
// forever; with one it fails in bounded time.
type deafSMTP struct {
	ln     net.Listener
	greet  bool
	closed chan struct{}
}

func newDeafSMTP(t *testing.T, greet bool) *deafSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d := &deafSMTP{ln: ln, greet: greet, closed: make(chan struct{})}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			if greet {
				fmt.Fprint(c, "220 deaf ESMTP\r\n")
			}
			go func() { <-d.closed; c.Close() }() // hold the connection open, say nothing
		}
	}()
	t.Cleanup(func() { close(d.closed); ln.Close() })
	return d
}

func TestSendToStalledServerTimesOut(t *testing.T) {
	d := newDeafSMTP(t, true)
	m := &SMTPMailer{Addr: d.ln.Addr().String(), From: "dms@example.com", Timeout: 300 * time.Millisecond}

	start := time.Now()
	err := m.Send([]string{"me@example.com"}, "test", "telo")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("send against a stalled server returned no error")
	}
	if elapsed > 5*time.Second {
		t.Fatalf("send took %s — the deadline did not apply", elapsed)
	}
}

func TestSendToSilentServerTimesOut(t *testing.T) {
	d := newDeafSMTP(t, false) // not even a greeting
	m := &SMTPMailer{Addr: d.ln.Addr().String(), From: "dms@example.com", Timeout: 300 * time.Millisecond}

	start := time.Now()
	err := m.Send([]string{"me@example.com"}, "test", "telo")
	if err == nil {
		t.Fatal("send against a silent server returned no error")
	}
	if e := time.Since(start); e > 5*time.Second {
		t.Fatalf("send took %s — the deadline did not apply", e)
	}
	if !strings.Contains(err.Error(), "smtp") {
		t.Errorf("unhelpful error: %v", err)
	}
}

// The point of the finding: while a tick is stuck talking to a dead relay, the
// owner's check-in must still get through.
func TestCheckInIsNotBlockedByAStalledTick(t *testing.T) {
	cfg := testConfig(t)
	cfg.applyDefaults()
	c := newClock()
	blocked := make(chan struct{})
	release := make(chan struct{})
	svc, err := New(cfg, c.now, &blockingMailer{blocked: blocked, release: release}, nil)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() { svc.Tick(); close(done) }() // will hang inside the mailer

	select {
	case <-blocked:
	case <-time.After(5 * time.Second):
		t.Fatal("the tick never reached the mailer")
	}

	// The tick is stuck mid-send. The veto must not wait for it.
	vetoed := make(chan struct{})
	go func() { svc.CheckIn(); close(vetoed) }()
	select {
	case <-vetoed:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("CheckIn blocked behind a stalled send — the owner cannot veto")
	}

	// Phase must be readable too, which the HTTP status page needs.
	if p := svc.Phase(); p != PhaseNormal {
		t.Errorf("phase = %s, want normal", p)
	}
	close(release)
	<-done
}

// blockingMailer stops inside Send until it is released.
type blockingMailer struct {
	blocked chan struct{}
	release chan struct{}
	once    bool
}

func (m *blockingMailer) Check() error { return nil }
func (m *blockingMailer) Send(to []string, subject, body string) error {
	if !m.once {
		m.once = true
		close(m.blocked)
		<-m.release
	}
	return nil
}

// A veto that lands while the envelopes are going out stops the rest of them.
func TestVetoDuringReleaseStopsRemainingEnvelopes(t *testing.T) {
	cfg := testConfig(t)
	dir := t.TempDir()
	cfg.EnvelopePath, cfg.FriendEmail = "", ""
	cfg.Envelopes = []Envelope{
		{ID: "prva", Path: writeEnvelope(t, dir, "a.asc"), To: []EnvelopeRecipient{{Email: "a@example.com"}}},
		{ID: "druha", Path: writeEnvelope(t, dir, "b.asc"), To: []EnvelopeRecipient{{Email: "b@example.com"}}},
	}
	cfg.applyDefaults()
	c := newClock()
	gate := &gateMailer{reached: make(chan struct{}), release: make(chan struct{})}
	svc, err := New(cfg, c.now, gate, nil)
	if err != nil {
		t.Fatal(err)
	}

	svc.Tick()
	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	c.add(7 * 24 * time.Hour)

	gate.arm("a@example.com")
	done := make(chan struct{})
	go func() { svc.Tick(); close(done) }()

	select {
	case <-gate.reached:
	case <-time.After(5 * time.Second):
		t.Fatal("release never reached the first envelope")
	}
	svc.CheckIn() // the owner turns up alive mid-delivery
	close(gate.release)
	<-done

	if svc.Phase() != PhaseNormal {
		t.Fatalf("phase = %s, want normal after the veto", svc.Phase())
	}
	if !gate.sawRecipient("a@example.com") {
		t.Errorf("the first envelope should have gone out before the veto")
	}
	if gate.sawRecipient("b@example.com") {
		t.Errorf("the second envelope was sent after the owner vetoed")
	}
}

// gateMailer blocks the first send to a chosen address until released.
type gateMailer struct {
	fakeMailer
	on      string
	reached chan struct{}
	release chan struct{}
	fired   bool
}

func (m *gateMailer) arm(addr string) { m.on = addr }
func (m *gateMailer) Send(to []string, subject, body string) error {
	if !m.fired && len(to) == 1 && to[0] == m.on {
		m.fired = true
		close(m.reached)
		<-m.release
	}
	return m.fakeMailer.Send(to, subject, body)
}
func (m *gateMailer) sawRecipient(addr string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sent {
		for _, t := range s.to {
			if t == addr {
				return true
			}
		}
	}
	return false
}
