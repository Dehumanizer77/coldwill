package dms

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"coldwill/dms/internal/i18n"
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
// healthy) releasing the envelope. Safe to call on any cadence; it is
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
		s.releaseEnvelope(now)
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

// envelopeName is the file name the heir sees on the attachment.
const envelopeName = "envelope.pdf"

// releaseEnvelope sends the envelope to the primary heir. Delivery is network
// work, so it runs without the state lock, and the phase is read again once the
// send returns: a check-in that lands while a slow relay is still busy with the
// message wins, and the switch does not mark itself fired.
func (s *Service) releaseEnvelope(now time.Time) {
	s.mu.Lock()
	vetoed := s.st.Phase != PhaseCountdown
	s.mu.Unlock()
	if vetoed {
		return
	}

	data, err := os.ReadFile(s.cfg.EnvelopePath)
	if err != nil || !isPDF(data) {
		// The self-test passed moments ago, so the file changed underneath.
		// Stay in the countdown; the next tick's self-test names the problem.
		if err == nil {
			err = fmt.Errorf("not a PDF")
		}
		log.Printf("dms: envelope unusable at release: %v", err)
		s.notifyOwner("subj.sendfail", "body.envfail")
		return
	}
	// One channel getting through is enough; none is retried on the next tick.
	delivered, sendErr := s.send([]Recipient{s.cfg.Heir.recipient()},
		[]Attachment{{Name: envelopeName, ContentType: "application/pdf", Data: data}},
		"subj.envelope", "body.envelope", nameOrOwner(s.cfg))

	s.mu.Lock()
	if s.st.Phase != PhaseCountdown {
		// A check-in landed while the envelope was going out.
		s.mu.Unlock()
		if delivered > 0 {
			s.notifyOwner("subj.cancelled", "body.cancelled", s.heirLabel())
		}
		return
	}
	if delivered == 0 {
		s.mu.Unlock()
		reason := "no channel reached the heir"
		if sendErr != nil {
			reason = sendErr.Error()
		}
		s.notifyOwner("subj.sendfail", "body.sendfail", reason)
		return
	}
	s.st.Phase = PhaseFired
	s.st.FiredAt = now
	s.persist()
	s.mu.Unlock()

	log.Printf("dms: FIRED — envelope delivered to the heir on %d channel(s)", delivered)
	s.notifyOwner("subj.sent", "body.sent", s.heirLabel())
}

// heirLabel is how the owner's own messages name the heir.
func (s *Service) heirLabel() string {
	switch h := s.cfg.Heir; {
	case h.Name != "":
		return h.Name
	case h.Email != "":
		return h.Email
	default:
		return h.Signal
	}
}

// isPDF is a sanity check rather than validation: it catches the wrong file,
// the map saved as text for instance, while the owner can still replace it.
func isPDF(b []byte) bool { return bytes.HasPrefix(b, []byte("%PDF-")) }

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
	data, err := os.ReadFile(s.cfg.EnvelopePath)
	if err != nil {
		return false, "the envelope cannot be read: " + err.Error()
	}
	if len(data) == 0 {
		return false, "the envelope is empty"
	}
	if !isPDF(data) {
		return false, "the envelope is not a PDF"
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
