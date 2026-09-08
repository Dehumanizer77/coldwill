package dms

import (
	"errors"
	"fmt"
	"log"
	"time"

	"inh/dms/internal/i18n"
)

// Recipient is one person and the addresses they can be reached at. An empty
// address simply means "not reachable on that channel".
type Recipient struct {
	Name   string
	Email  string
	Signal string // E.164 phone number
	Lang   i18n.Lang
}

func (s *Service) owner() Recipient {
	return Recipient{
		Name: nameOrOwner(s.cfg), Email: s.cfg.UserEmail,
		Signal: s.cfg.UserSignal, Lang: i18n.Parse(s.cfg.UserLang),
	}
}

func (c Confirmer) recipient() Recipient {
	return Recipient{Name: c.Name, Email: c.Email, Signal: c.Signal, Lang: i18n.Parse(c.Lang)}
}

// durArg is a duration that is spelled out in each recipient's language at
// render time, since "7 days" and "7 dní" cannot both be baked into one string.
type durArg time.Duration

// notify sends one message to every recipient on every channel they have an
// address for, rendering it in that person's language. It returns how many
// recipient/channel deliveries succeeded and the failures. Failures are logged,
// never fatal: the caller decides what a zero delivery count means.
func (s *Service) notify(rcpts []Recipient, subjKey, bodyKey string, args ...any) (int, error) {
	return s.send(rcpts, func(l i18n.Lang) string { return i18n.S(l, subjKey) }, subjKey, bodyKey, args...)
}

// notifyWithSubject is notify for a message whose subject is supplied verbatim,
// which is how an envelope carries its own.
func (s *Service) notifyWithSubject(rcpts []Recipient, subject, bodyKey string, args ...any) (int, error) {
	return s.send(rcpts, func(i18n.Lang) string { return subject }, subject, bodyKey, args...)
}

func (s *Service) send(rcpts []Recipient, subjOf func(i18n.Lang) string, label, bodyKey string, args ...any) (int, error) {
	delivered := 0
	var errs []error

	// One transaction per address. A relay that rejects a single RCPT TO aborts
	// the whole transaction, so a batched send would silently drop the message
	// for everyone else on the list, including recipients the relay accepted.
	for _, r := range rcpts {
		subject := subjOf(r.Lang)
		body := i18n.S(r.Lang, bodyKey, localize(r.Lang, args)...)
		if r.Email != "" {
			if err := s.mail.Send([]string{r.Email}, subject, body); err != nil {
				log.Printf("dms: email %q to %s failed: %v", label, r.Email, err)
				errs = append(errs, fmt.Errorf("e-mail %s: %w", r.Email, err))
			} else {
				delivered++
			}
		}
		if r.Signal != "" && s.sig != nil {
			if err := s.sig.Send([]string{r.Signal}, subject+"\n\n"+body); err != nil {
				log.Printf("dms: signal %q to %s failed: %v", label, r.Signal, err)
				errs = append(errs, fmt.Errorf("signal %s: %w", r.Signal, err))
			} else {
				delivered++
			}
		}
	}
	return delivered, errors.Join(errs...)
}

// localize spells out any duration arguments in the given language.
func localize(l i18n.Lang, args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		if d, ok := a.(durArg); ok {
			out[i] = humanDur(l, time.Duration(d))
			continue
		}
		out[i] = a
	}
	return out
}

// notifyOwner is the common case: tell me something, in my language.
func (s *Service) notifyOwner(subjKey, bodyKey string, args ...any) {
	s.notify([]Recipient{s.owner()}, subjKey, bodyKey, args...)
}

// mailOwner reaches me on e-mail only — used to report that Signal itself is
// broken, where the broken channel obviously cannot carry its own alarm.
func (s *Service) mailOwner(subjKey, bodyKey string, args ...any) {
	l := i18n.Parse(s.cfg.UserLang)
	subject := i18n.S(l, subjKey)
	if err := s.mail.Send([]string{s.cfg.UserEmail}, subject, i18n.S(l, bodyKey, localize(l, args)...)); err != nil {
		log.Printf("dms: email %q to owner failed: %v", subjKey, err)
	}
}
