package dms

import (
	"errors"
	"fmt"
	"log"
)

// Recipient is one person and the addresses they can be reached at. An empty
// address simply means "not reachable on that channel".
type Recipient struct {
	Name   string
	Email  string
	Signal string // E.164 phone number
}

func (s *Service) owner() Recipient {
	return Recipient{Name: nameOrOwner(s.cfg), Email: s.cfg.UserEmail, Signal: s.cfg.UserSignal}
}

func (s *Service) friend() Recipient {
	return Recipient{Email: s.cfg.FriendEmail, Signal: s.cfg.FriendSignal}
}

func (c Confirmer) recipient() Recipient {
	return Recipient{Name: c.Name, Email: c.Email, Signal: c.Signal}
}

// notify delivers one message to every recipient on every channel they have an
// address for. It returns how many recipient/channel deliveries succeeded and
// the failures. Failures are logged, never fatal: the caller decides what a
// zero delivery count means.
func (s *Service) notify(rcpts []Recipient, subject, body string) (int, error) {
	delivered := 0
	var errs []error

	// One transaction per address. A relay that rejects a single RCPT TO aborts
	// the whole transaction, so a batched send would silently drop the message
	// for everyone else on the list — including recipients the relay accepted.
	for _, r := range rcpts {
		if r.Email != "" {
			if err := s.mail.Send([]string{r.Email}, subject, body); err != nil {
				log.Printf("dms: email %q to %s failed: %v", subject, r.Email, err)
				errs = append(errs, fmt.Errorf("e-mail %s: %w", r.Email, err))
			} else {
				delivered++
			}
		}
		if r.Signal != "" && s.sig != nil {
			if err := s.sig.Send([]string{r.Signal}, subject+"\n\n"+body); err != nil {
				log.Printf("dms: signal %q to %s failed: %v", subject, r.Signal, err)
				errs = append(errs, fmt.Errorf("signal %s: %w", r.Signal, err))
			} else {
				delivered++
			}
		}
	}
	return delivered, errors.Join(errs...)
}

// notifyOwner is the common case: tell me something, on every channel I have.
func (s *Service) notifyOwner(subject, body string) {
	s.notify([]Recipient{s.owner()}, subject, body)
}

// mailOwner reaches me on e-mail only — used to report that Signal itself is
// broken, where the broken channel obviously cannot carry its own alarm.
func (s *Service) mailOwner(subject, body string) {
	if err := s.mail.Send([]string{s.cfg.UserEmail}, subject, body); err != nil {
		log.Printf("dms: email %q to owner failed: %v", subject, err)
	}
}
