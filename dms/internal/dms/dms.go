package dms

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"inh/dms/internal/i18n"
)

// Service is the dead-man's switch state machine. Time and email are injected
// (clock, Mailer) so the whole thing is deterministically testable.
type Service struct {
	cfg   Config
	clock func() time.Time
	mail  Mailer
	sig   SignalSender // nil = e-mail only

	mu sync.Mutex
	st State
}

// New builds the service. signal may be nil, in which case everything runs on
// e-mail alone.
func New(cfg Config, clock func() time.Time, mailer Mailer, signal SignalSender) (*Service, error) {
	if err := ensureDir(cfg.StatePath); err != nil {
		return nil, err
	}
	st, err := loadState(cfg.StatePath, clock())
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, clock: clock, mail: mailer, sig: signal, st: st}, nil
}

func (s *Service) now() time.Time { return s.clock() }

// outgoing is a message the tick decided to send. Bodies are built while the
// state lock is held (they read state); the sending itself happens after it is
// released, so a stalled SMTP server cannot block the owner's veto.
type outgoing struct {
	to       []Recipient
	subjKey  string
	bodyKey  string
	args     []any
	mailOnly bool // used for the alert about Signal itself being down
}

// Tick performs one evaluation: probes, phase transitions, and (when due and
// healthy) releasing the envelopes. Safe to call on any cadence; it is
// idempotent between meaningful time boundaries.
//
// Network work deliberately happens outside the state lock. Everything the DMS
// talks to can stall — postfix, the Signal container, a relay that accepts a
// connection and then goes quiet — and a tick that held the lock while waiting
// would also block CheckIn, which is the owner's veto.
func (s *Service) Tick() {
	now := s.now()

	// Probes first, with no lock held.
	healthy, reason := s.selfTest()
	sigErr := s.probeSignal()

	s.mu.Lock()
	s.st.Healthy = healthy
	var outbox []outgoing
	add := func(subjKey, bodyKey string, to []Recipient, args ...any) {
		outbox = append(outbox, outgoing{to: to, subjKey: subjKey, bodyKey: bodyKey, args: args})
	}
	me := []Recipient{s.owner()}

	if s.sig != nil {
		s.st.SignalOK = sigErr == nil
		if sigErr != nil && now.Sub(s.st.LastSignalAlertAt) >= s.cfg.AlertInterval.D() {
			// The broken channel cannot carry its own alarm, so this goes by e-mail.
			outbox = append(outbox, outgoing{
				to: me, subjKey: "subj.signaldown", bodyKey: "body.signaldown",
				args: []any{sigErr}, mailOnly: true,
			})
			s.st.LastSignalAlertAt = now
		}
	}

	if !healthy {
		if now.Sub(s.st.LastAlertAt) >= s.cfg.AlertInterval.D() {
			add("subj.fault", "body.fault", me, reason)
			s.st.LastAlertAt = now
		}
	} else if now.Sub(s.st.LastHealthBeatAt) >= s.cfg.HealthBeatInterval.D() {
		add("subj.healthy", "body.healthy", me, s.healthSignalNote(), s.checkinURL())
		s.st.LastHealthBeatAt = now
	}

	fireNow := false
	switch s.st.Phase {
	case PhaseNormal:
		silence := now.Sub(s.st.LastCheckIn)
		switch {
		case silence >= s.cfg.SilenceThreshold.D():
			cycle := newCycleID()
			if cycle == "" {
				add("subj.fault", "body.cycleid", me)
				break
			}
			s.st.Phase = PhaseAwaiting
			s.st.CycleID = cycle
			s.st.Confirmations = nil
			s.st.LastConfirmReqAt = now
			s.st.LastReminderAt = now
			s.addConfirmRequests(&outbox)
			add("subj.awaiting", "body.awaiting", me, durArg(silence), s.checkinURL())
		case silence >= s.cfg.CheckInInterval.D():
			if now.Sub(s.st.LastReminderAt) >= s.cfg.ReminderInterval.D() {
				add("subj.checkin", "body.checkin", me, s.checkinURL())
				s.st.LastReminderAt = now
			}
		}

	case PhaseAwaiting:
		if now.Sub(s.st.LastReminderAt) >= s.cfg.ReminderInterval.D() {
			add("subj.stillwait", "body.checkin.urgent", me, s.checkinURL())
			s.st.LastReminderAt = now
		}
		if now.Sub(s.st.LastConfirmReqAt) >= s.cfg.ReminderInterval.D() {
			s.addConfirmRequests(&outbox)
			s.st.LastConfirmReqAt = now
		}

	case PhaseCountdown:
		if now.Sub(s.st.LastWarningAt) >= s.cfg.WarningInterval.D() {
			left := s.cfg.ReleaseDelay.D() - now.Sub(s.st.ConfirmedAt)
			add("subj.countdown", "body.countdown", me, durArg(max(left, 0)), s.checkinURL())
			s.st.LastWarningAt = now
		}
		fireNow = healthy && now.Sub(s.st.ConfirmedAt) >= s.cfg.ReleaseDelay.D()

	case PhaseFired:
		// terminal — nothing to do
	}
	s.persist()
	s.mu.Unlock()

	s.deliver(outbox)
	if fireNow {
		s.releaseEnvelopes(now)
	}
}

// deliver sends what the tick decided to send, with no lock held.
func (s *Service) deliver(outbox []outgoing) {
	for _, m := range outbox {
		if m.mailOnly {
			s.mailOwner(m.subjKey, m.bodyKey, m.args...)
			continue
		}
		s.notify(m.to, m.subjKey, m.bodyKey, m.args...)
	}
}

// CheckIn records that the user is alive. It cancels any in-progress
// confirmation/countdown (unless already fired).
func (s *Service) CheckIn() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.st.LastCheckIn = now
	s.st.LastReminderAt = now
	if s.st.Phase != PhaseFired {
		s.st.Phase = PhaseNormal
		s.st.ConfirmedAt = time.Time{}
		s.st.ConfirmedBy = ""
		s.st.Confirmations = nil
		s.st.LastWarningAt = time.Time{}
		// A veto invalidates every confirmation link that was sent out: the next
		// waiting cycle gets a new id, so an old link cannot be replayed.
		s.st.CycleID = ""
		// If a partial fire had already sent something, forget it: after a veto
		// the next real firing must deliver every envelope again.
		s.st.Delivered = nil
	}
	log.Printf("dms: check-in recorded (phase now %s)", s.st.Phase)
	s.persist()
}

// Confirm is called when a confirmer attests to the user's death. It starts the
// release countdown. Idempotent if already counting down.
func (s *Service) Confirm(id string) error {
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()
	var name string
	known := false
	for _, c := range s.cfg.Confirmers {
		if c.ID == id {
			known, name = true, c.Name
		}
	}
	if !known {
		return fmt.Errorf("unknown confirmer %q", id)
	}
	if s.st.Phase == PhaseCountdown {
		return nil // already confirmed by enough people
	}
	if s.st.Phase != PhaseAwaiting {
		return fmt.Errorf("not awaiting confirmation (phase %s)", s.st.Phase)
	}
	if !s.st.hasConfirmed(id) {
		s.st.Confirmations = append(s.st.Confirmations, id)
	}
	got, need := len(s.st.Confirmations), s.cfg.ConfirmQuorum

	if got < need {
		// Not enough yet: record it and tell the owner someone attested.
		log.Printf("dms: confirmation %d/%d (by %s)", got, need, id)
		s.persist()
		s.mu.Unlock()
		locked = false
		s.notifyOwner("subj.partial", "body.partial", name, need, got, s.checkinURL())
		return nil
	}

	now := s.now()
	s.st.Phase = PhaseCountdown
	s.st.ConfirmedAt = now
	s.st.ConfirmedBy = id
	s.st.LastWarningAt = time.Time{}
	log.Printf("dms: confirmed by %s (%d/%d); countdown started", id, got, need)
	delay := s.cfg.ReleaseDelay.D()
	s.persist()
	s.mu.Unlock()
	locked = false

	// Sending happens with the lock released: a stalled relay here would
	// otherwise block the owner's check-in, which is the veto.
	s.notifyOwner("subj.confirmed", "body.confirmed", name, durArg(delay), s.checkinURL())
	return nil
}

// Confirmations reports how many distinct confirmers have attested in this
// cycle and how many are needed.
func (s *Service) Confirmations() (got, need int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.st.Confirmations), s.cfg.ConfirmQuorum
}

// Phase returns the current phase (for status / tests).
func (s *Service) Phase() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.Phase
}

// releaseEnvelopes delivers the envelopes that have not gone out yet. It runs
// without the state lock (delivery is network work) and re-reads the phase
// under the lock around every envelope: if the owner vetoes while a slow relay
// is chewing on the first envelope, the rest must not be sent.
func (s *Service) releaseEnvelopes(now time.Time) {
	var sent, failed []string
	vetoed := false

	for _, e := range s.cfg.Envelopes {
		s.mu.Lock()
		skip := s.st.Phase != PhaseCountdown || s.st.wasDelivered(e.ID)
		vetoed = s.st.Phase != PhaseCountdown
		s.mu.Unlock()
		if vetoed {
			break
		}
		if skip {
			continue
		}

		data, err := os.ReadFile(e.Path)
		if err != nil || len(data) == 0 {
			log.Printf("dms: envelope %s unreadable: %v", e.ID, err)
			failed = append(failed, e.ID)
			continue
		}
		// One channel through is enough for a given envelope; an envelope that
		// reached nobody is retried on the next tick, one that got out is not.
		note := ""
		if e.Note != "" {
			note = e.Note + "\n\n"
		}
		delivered, err := s.notifyEnvelope(e, note, string(data))

		s.mu.Lock()
		if s.st.Phase != PhaseCountdown {
			// A check-in landed while this envelope was being delivered.
			s.mu.Unlock()
			vetoed = true
			if delivered > 0 {
				sent = append(sent, e.ID)
			}
			break
		}
		if delivered > 0 {
			s.st.Delivered = append(s.st.Delivered, e.ID)
			sent = append(sent, e.ID)
			log.Printf("dms: envelope %s delivered on %d channel(s)", e.ID, delivered)
		} else {
			reason := e.ID
			if err != nil {
				reason += " (" + err.Error() + ")"
			}
			failed = append(failed, reason)
		}
		s.persist()
		s.mu.Unlock()
	}

	if vetoed {
		l := i18n.Parse(s.cfg.UserLang)
		msg := i18n.S(l, "body.cancelled")
		if len(sent) > 0 {
			msg += i18n.S(l, "body.cancelled.some", strings.Join(sent, ", "))
		}
		s.notifyOwner("subj.cancelled", "body.raw", msg)
		return
	}

	if len(failed) > 0 {
		// Stay in countdown so the next tick retries what is left.
		l := i18n.Parse(s.cfg.UserLang)
		msg := i18n.S(l, "body.sendfail.list", strings.Join(failed, ", "))
		if len(sent) > 0 {
			msg = i18n.S(l, "body.sendfail.sent", strings.Join(sent, ", ")) + msg
		}
		s.notifyOwner("subj.sendfail", "body.raw", msg)
		return
	}

	s.mu.Lock()
	if s.st.Phase != PhaseCountdown {
		s.mu.Unlock()
		return
	}
	s.st.Phase = PhaseFired
	s.st.FiredAt = now
	s.persist()
	s.mu.Unlock()

	log.Printf("dms: FIRED — %d envelope(s) delivered", len(s.cfg.Envelopes))
	s.notifyOwner("subj.sent", "body.sent", strings.Join(envelopeIDs(s.cfg.Envelopes), ", "))
}

// notifyEnvelope sends one envelope, honouring a subject the config set for it
// and otherwise using the catalogue's, in each recipient's language.
func (s *Service) notifyEnvelope(e Envelope, note, ciphertext string) (int, error) {
	if e.Subject != "" {
		return s.notifyWithSubject(e.recipients(), e.Subject, "body.envelope",
			nameOrOwner(s.cfg), note, ciphertext)
	}
	return s.notify(e.recipients(), "subj.envelope", "body.envelope",
		nameOrOwner(s.cfg), note, ciphertext)
}

func envelopeIDs(es []Envelope) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.ID
	}
	return out
}

// probeSignal asks the Signal container whether it is still usable. It is a
// pure network probe: the result is recorded and alerted on by Tick, under the
// lock. A dead Signal never blocks firing — e-mail is the channel the envelope
// actually depends on, and a second channel must not become a second way for
// the whole thing to jam.
func (s *Service) probeSignal() error {
	if s.sig == nil {
		return nil
	}
	err := s.sig.Check()
	if err != nil {
		log.Printf("dms: signal channel degraded: %v", err)
	}
	return err
}

// selfTest checks everything that must hold before the DMS may fire. It takes
// no lock and touches no state, so Tick can run it before locking.
func (s *Service) selfTest() (bool, string) {
	if len(s.cfg.HMACSecret) < 16 {
		return false, "the HMAC secret is missing or too short"
	}
	if err := s.mail.Check(); err != nil {
		return false, "mail transport: " + err.Error()
	}
	for _, e := range s.cfg.Envelopes {
		data, err := os.ReadFile(e.Path)
		if err != nil {
			return false, "envelope " + e.ID + " cannot be read: " + err.Error()
		}
		if len(data) == 0 {
			return false, "envelope " + e.ID + " is empty"
		}
		if !strings.Contains(string(data), "BEGIN PGP MESSAGE") {
			return false, "envelope " + e.ID + " is not an ASCII-armored PGP message"
		}
	}
	if err := stateWritable(s.cfg.StatePath); err != nil {
		return false, "state is not writable: " + err.Error()
	}
	return true, ""
}

func (s *Service) persist() {
	if err := s.st.save(s.cfg.StatePath); err != nil {
		log.Printf("dms: state save failed: %v", err)
	}
}

// ---- links + email bodies ----

func (s *Service) checkinURL() string {
	return s.cfg.PublicBaseURL + "/checkin?token=" + s.token(s.checkinAction())
}

func (s *Service) confirmURL(c Confirmer) string {
	return s.cfg.PublicBaseURL + "/confirm?id=" + c.ID +
		"&token=" + s.token(s.confirmAction(c.ID, s.st.CycleID))
}

// currentCycle reads the cycle id for the HTTP layer, which verifies tokens
// outside the state lock.
func (s *Service) currentCycle() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.CycleID
}

// healthSignalNote is the one line in the health beat that depends on the
// secondary channel, empty when Signal is not configured at all.
func (s *Service) healthSignalNote() string {
	if s.sig == nil {
		return ""
	}
	l := i18n.Parse(s.cfg.UserLang)
	if s.st.SignalOK {
		return i18n.S(l, "body.healthy.signal.ok")
	}
	return i18n.S(l, "body.healthy.signal.bad")
}

// addConfirmRequests queues one request per confirmer, each in their own
// language and with their own single-use link.
func (s *Service) addConfirmRequests(outbox *[]outgoing) {
	for _, c := range s.cfg.Confirmers {
		*outbox = append(*outbox, outgoing{
			to:      []Recipient{c.recipient()},
			subjKey: "subj.confirmreq",
			bodyKey: "body.confirmreq",
			args: []any{
				c.Name, nameOrOwner(s.cfg),
				durArg(s.cfg.ReleaseDelay.D()), s.confirmURL(c),
			},
		})
	}
}

func nameOrOwner(c Config) string {
	if c.UserEmail != "" {
		return c.UserEmail
	}
	return i18n.S(i18n.Parse(c.UserLang), "owner")
}

func humanDur(l i18n.Lang, d time.Duration) string {
	if d >= 24*time.Hour {
		return i18n.S(l, "time.days", int((d+12*time.Hour)/(24*time.Hour)))
	}
	if d >= time.Hour {
		return i18n.S(l, "time.hours", int((d+30*time.Minute)/time.Hour))
	}
	return i18n.S(l, "time.minutes", int((d+30*time.Second)/time.Minute))
}
