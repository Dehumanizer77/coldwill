package dms

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Mailer sends plain-text UTF-8 email and can check that its transport is
// reachable (used by the self-test).
type Mailer interface {
	Send(to []string, subject, body string) error
	Check() error
}

// SMTPMailer talks to a local MTA (e.g. postfix on 127.0.0.1:25), no auth.
type SMTPMailer struct {
	Addr string
	From string
}

func (m *SMTPMailer) Check() error {
	c, err := net.DialTimeout("tcp", m.Addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", m.Addr, err)
	}
	return c.Close()
}

// Send delivers one message. It deliberately does not use smtp.SendMail: that
// helper upgrades to STARTTLS whenever the server offers it and then verifies
// the certificate against the dialed name, which a local postfix can never
// satisfy ("cannot validate certificate for 127.0.0.1 because it doesn't
// contain any IP SANs") — and every notification, the envelope included, would
// fail. On loopback TLS buys nothing: the bytes never leave the machine. Any
// other relay must offer STARTTLS and present a certificate that verifies.
func (m *SMTPMailer) Send(to []string, subject, body string) error {
	host, _, err := net.SplitHostPort(m.Addr)
	if err != nil {
		return fmt.Errorf("smtp addr %q: %w", m.Addr, err)
	}
	c, err := smtp.Dial(m.Addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", m.Addr, err)
	}
	defer c.Close()

	if !isLoopbackHost(host) {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return fmt.Errorf("smtp %s: remote relay does not offer STARTTLS", m.Addr)
		}
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("smtp %s: starttls: %w", m.Addr, err)
		}
	}

	if err := c.Mail(m.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp RCPT TO %s: %w", addr, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write([]byte(m.message(to, subject, body))); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close: %w", err)
	}
	return c.Quit()
}

func (m *SMTPMailer) message(to []string, subject, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", m.From)
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(to, ", "))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
