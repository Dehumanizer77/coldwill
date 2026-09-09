package dms

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
)

// fakeSMTP speaks just enough SMTP to accept one message, and it advertises
// STARTTLS the way a default postfix on :25 does — which is exactly what used
// to make the real mailer fail certificate verification for "127.0.0.1".
type fakeSMTP struct {
	ln   net.Listener
	mu   sync.Mutex
	log  []string
	body string
	done chan struct{}
}

func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, done: make(chan struct{})}
	go f.serve()
	t.Cleanup(func() { ln.Close() })
	return f
}

func (f *fakeSMTP) addr() string { return f.ln.Addr().String() }

func (f *fakeSMTP) record(s string) { f.mu.Lock(); f.log = append(f.log, s); f.mu.Unlock() }

func (f *fakeSMTP) saw(cmd string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range f.log {
		if strings.Contains(strings.ToUpper(l), strings.ToUpper(cmd)) {
			return true
		}
	}
	return false
}

func (f *fakeSMTP) serve() {
	conn, err := f.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	defer close(f.done)
	r := bufio.NewScanner(conn)
	fmt.Fprint(conn, "220 fake ESMTP\r\n")
	for r.Scan() {
		line := r.Text()
		f.record(line)
		up := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(up, "EHLO"):
			fmt.Fprint(conn, "250-fake\r\n250-STARTTLS\r\n250 8BITMIME\r\n")
		case strings.HasPrefix(up, "HELO"):
			fmt.Fprint(conn, "250 fake\r\n")
		case strings.HasPrefix(up, "STARTTLS"):
			fmt.Fprint(conn, "454 TLS not available\r\n")
		case strings.HasPrefix(up, "MAIL FROM"), strings.HasPrefix(up, "RCPT TO"):
			fmt.Fprint(conn, "250 ok\r\n")
		case strings.HasPrefix(up, "DATA"):
			fmt.Fprint(conn, "354 send it\r\n")
			var b strings.Builder
			for r.Scan() {
				if r.Text() == "." {
					break
				}
				b.WriteString(r.Text() + "\n")
			}
			f.mu.Lock()
			f.body = b.String()
			f.mu.Unlock()
			fmt.Fprint(conn, "250 queued\r\n")
		case strings.HasPrefix(up, "QUIT"):
			fmt.Fprint(conn, "221 bye\r\n")
			return
		default:
			fmt.Fprint(conn, "250 ok\r\n")
		}
	}
}

// A local relay that offers STARTTLS must not be upgraded: postfix's default
// certificate cannot possibly name "127.0.0.1", the connection never leaves the
// machine, and the failure mode is every notification — the envelope included —
// silently not being sent.
func TestSendToLoopbackSkipsSTARTTLS(t *testing.T) {
	f := newFakeSMTP(t)
	m := &SMTPMailer{Addr: f.addr(), From: "dms@example.com"}

	if err := m.Send([]string{"me@example.com", "friend@example.com"}, "[DMS] všetko v poriadku", "telo správy"); err != nil {
		t.Fatalf("send: %v", err)
	}
	<-f.done

	if f.saw("STARTTLS") {
		t.Errorf("client tried STARTTLS against a loopback relay")
	}
	for _, want := range []string{"MAIL FROM:<dms@example.com>", "RCPT TO:<me@example.com>", "RCPT TO:<friend@example.com>"} {
		if !f.saw(want) {
			t.Errorf("missing SMTP command %q; got %v", want, f.log)
		}
	}
	f.mu.Lock()
	body := f.body
	f.mu.Unlock()
	for _, want := range []string{
		"From: dms@example.com",
		"To: me@example.com, friend@example.com",
		"Content-Type: text/plain; charset=\"UTF-8\"",
		"telo správy",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("message missing %q; got:\n%s", want, body)
		}
	}
	// The fixture is deliberately accented, because the switch really does send
	// Slovak and Czech mail: a subject with non-ASCII in it has to go out
	// MIME-encoded rather than raw, or the header is invalid.
	if !strings.Contains(body, "Subject: =?UTF-8?") {
		t.Errorf("subject was not MIME-encoded:\n%s", body)
	}
	if strings.Contains(body, "Subject: [DMS] všetko") {
		t.Errorf("raw non-ASCII subject reached the wire:\n%s", body)
	}
}

func TestIsLoopbackHost(t *testing.T) {
	for host, want := range map[string]bool{
		"127.0.0.1": true, "127.0.1.1": true, "::1": true, "localhost": true,
		"203.0.113.5": false, "mail.example.com": false, "": false,
	} {
		if got := isLoopbackHost(host); got != want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}
