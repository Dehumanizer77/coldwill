package dms

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
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
	subject  string
	body     string
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
	add := func(subject, body string, to ...Recipient) {
		outbox = append(outbox, outgoing{to: to, subject: subject, body: body})
	}

	if s.sig != nil {
		s.st.SignalOK = sigErr == nil
		if sigErr != nil && now.Sub(s.st.LastSignalAlertAt) >= s.cfg.AlertInterval.D() {
			// The broken channel cannot carry its own alarm, so this goes by e-mail.
			outbox = append(outbox, outgoing{
				to:      []Recipient{s.owner()},
				subject: "[DMS] Signal nefunguje",
				body: "Druhý kanál (Signal) neodpovedá: " + sigErr.Error() +
					"\n\nE-mail funguje ďalej a DMS beží normálne — oprav Signal, keď budeš môcť.\n— DMS",
				mailOnly: true,
			})
			s.st.LastSignalAlertAt = now
		}
	}

	if !healthy {
		if now.Sub(s.st.LastAlertAt) >= s.cfg.AlertInterval.D() {
			add("[DMS] PORUCHA — skontroluj", s.bodyFault(reason), s.owner())
			s.st.LastAlertAt = now
		}
	} else if now.Sub(s.st.LastHealthBeatAt) >= s.cfg.HealthBeatInterval.D() {
		add("[DMS] v poriadku", s.bodyHealth(), s.owner())
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
				add("[DMS] PORUCHA — skontroluj",
					"Nepodarilo sa vygenerovať id cyklu (chyba generátora náhody); "+
						"výzva potvrdzovateľom sa NEODOSLALA. Skúsi sa znova.", s.owner())
				break
			}
			s.st.Phase = PhaseAwaiting
			s.st.CycleID = cycle
			s.st.LastConfirmReqAt = now
			s.st.LastReminderAt = now
			for _, c := range s.cfg.Confirmers {
				add("[DMS] Prosba o potvrdenie", s.bodyConfirmReq(c), c.recipient())
			}
			add("[DMS] Spustená kontrola po dlhom tichu", s.bodyAwaitingUser(silence), s.owner())
		case silence >= s.cfg.CheckInInterval.D():
			if now.Sub(s.st.LastReminderAt) >= s.cfg.ReminderInterval.D() {
				add("[DMS] Ozvi sa — check-in", s.bodyCheckin(false), s.owner())
				s.st.LastReminderAt = now
			}
		}

	case PhaseAwaiting:
		if now.Sub(s.st.LastReminderAt) >= s.cfg.ReminderInterval.D() {
			add("[DMS] STÁLE čakám — ozvi sa", s.bodyCheckin(true), s.owner())
			s.st.LastReminderAt = now
		}
		if now.Sub(s.st.LastConfirmReqAt) >= s.cfg.ReminderInterval.D() {
			for _, c := range s.cfg.Confirmers {
				add("[DMS] Prosba o potvrdenie", s.bodyConfirmReq(c), c.recipient())
			}
			s.st.LastConfirmReqAt = now
		}

	case PhaseCountdown:
		if now.Sub(s.st.LastWarningAt) >= s.cfg.WarningInterval.D() {
			left := s.cfg.ReleaseDelay.D() - now.Sub(s.st.ConfirmedAt)
			add("[DMS] Obálka sa čoskoro pošle", s.bodyCountdown(left), s.owner())
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
			s.mailOwner(m.subject, m.body)
			continue
		}
		s.notify(m.to, m.subject, m.body)
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
		return nil // already confirmed
	}
	if s.st.Phase != PhaseAwaiting {
		return fmt.Errorf("not awaiting confirmation (phase %s)", s.st.Phase)
	}
	now := s.now()
	s.st.Phase = PhaseCountdown
	s.st.ConfirmedAt = now
	s.st.ConfirmedBy = id
	s.st.LastWarningAt = time.Time{}
	log.Printf("dms: confirmed by %s; countdown started", id)
	body := s.bodyConfirmed(name)
	s.persist()
	s.mu.Unlock()
	locked = false

	// Sending happens with the lock released: a stalled relay here would
	// otherwise block the owner's check-in, which is the veto.
	s.notifyOwner("[DMS] Potvrdené — odpočet beží", body)
	return nil
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
			failed = append(failed, e.ID+" (nedá sa načítať)")
			continue
		}
		// One channel through is enough for a given envelope; an envelope that
		// reached nobody is retried on the next tick, one that got out is not.
		delivered, err := s.notify(e.recipients(), e.subject(), s.bodyEnvelope(e, string(data)))

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
		msg := "Počas odosielania prišiel tvoj check-in, takže sa zvyšok obálok NEODOSLAL."
		if len(sent) > 0 {
			msg += "\nEšte predtým stihli odísť: " + strings.Join(sent, ", ") +
				". Tie sa už vziať späť nedajú — daj príjemcom vedieť, že ide o planý poplach."
		}
		s.notifyOwner("[DMS] Odosielanie zrušené check-inom", msg)
		return
	}

	if len(failed) > 0 {
		// Stay in countdown so the next tick retries what is left.
		msg := "Nepodarilo sa doručiť: " + strings.Join(failed, ", ") + "\nSkúsi sa znova pri ďalšom tiku."
		if len(sent) > 0 {
			msg = "Odoslané: " + strings.Join(sent, ", ") + "\n" + msg
		}
		s.notifyOwner("[DMS] CHYBA: obálku sa nepodarilo odoslať", msg)
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
	s.notifyOwner("[DMS] Obálky odoslané",
		"Odoslané obálky: "+strings.Join(envelopeIDs(s.cfg.Envelopes), ", ")+
			".\nAk je to omyl, kontaktuj príjemcov.")
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
		return false, "HMAC secret chýba alebo je krátky"
	}
	if err := s.mail.Check(); err != nil {
		return false, "mail transport: " + err.Error()
	}
	for _, e := range s.cfg.Envelopes {
		data, err := os.ReadFile(e.Path)
		if err != nil {
			return false, "obálka " + e.ID + " sa nedá načítať: " + err.Error()
		}
		if len(data) == 0 {
			return false, "obálka " + e.ID + " je prázdna"
		}
		if !strings.Contains(string(data), "BEGIN PGP MESSAGE") {
			return false, "obálka " + e.ID + " nie je ASCII-armored PGP správa"
		}
	}
	if err := stateWritable(s.cfg.StatePath); err != nil {
		return false, "stav sa nedá zapísať: " + err.Error()
	}
	return true, ""
}

func (s *Service) askConfirmers() {
	for _, c := range s.cfg.Confirmers {
		s.notify([]Recipient{c.recipient()}, "[DMS] Prosba o potvrdenie", s.bodyConfirmReq(c))
	}
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

func (s *Service) bodyCheckin(urgent bool) string {
	lead := "Klikni a potvrď, že si v poriadku:"
	if urgent {
		lead = "Už dlho si sa neozval. Ak žiješ, OKAMŽITE potvrď:"
	}
	return lead + "\n\n" + s.checkinURL() +
		"\n\nAk sa neozveš, spustí sa proces odovzdania prístupov rodine.\n— DMS"
}

func (s *Service) bodyAwaitingUser(silence time.Duration) string {
	return "Neozval si sa " + humanDur(silence) + ". Požiadal som dôveryhodné osoby o potvrdenie.\n\n" +
		"Ak žiješ, OKAMŽITE zruš proces:\n\n" + s.checkinURL() + "\n— DMS"
}

func (s *Service) bodyConfirmReq(c Confirmer) string {
	return "Ahoj " + c.Name + ",\n\n" +
		"toto je automatická správa. " + nameOrOwner(s.cfg) + " sa dlhšie neozval.\n\n" +
		"AK vieš potvrdiť, že zomrel alebo je trvalo neschopný, otvor odkaz a potvrď tlačidlom.\n" +
		"Tým sa po " + humanDur(s.cfg.ReleaseDelay.D()) + " odošle zašifrovaná obálka s pokynmi.\n" +
		"AK to potvrdiť nevieš, nerob nič.\n\n" + s.confirmURL(c) + "\n— DMS"
}

func (s *Service) bodyConfirmed(name string) string {
	return name + " potvrdil. Obálka sa pošle o " + humanDur(s.cfg.ReleaseDelay.D()) +
		".\n\nAk je to omyl a žiješ, ZRUŠ to teraz:\n\n" + s.checkinURL() + "\n— DMS"
}

func (s *Service) bodyCountdown(left time.Duration) string {
	if left < 0 {
		left = 0
	}
	return "Obálka sa pošle približne o " + humanDur(left) + ".\n\n" +
		"Ak žiješ, ZRUŠ to:\n\n" + s.checkinURL() + "\n— DMS"
}

func (s *Service) bodyEnvelope(e Envelope, ciphertext string) string {
	note := ""
	if e.Note != "" {
		note = e.Note + "\n\n"
	}
	return "Ahoj,\n\nak ti prišla táto správa, " + nameOrOwner(s.cfg) +
		" pravdepodobne zomrel alebo je trvalo neschopný.\n\n" + note +
		"Nižšie je GPG-zašifrovaná obálka — rozšifruj ju svojím kľúčom.\n" +
		"Pomôž rodine podľa runbooku. Ďakujem.\n\n" +
		"-----\n" + ciphertext
}

func (s *Service) bodyHealth() string {
	sig := ""
	if s.sig != nil {
		sig = "\nSignal kanál: v poriadku."
		if !s.st.SignalOK {
			sig = "\nSignal kanál: NEFUNGUJE (e-mail beží ďalej)."
		}
	}
	return "DMS beží a self-testy prešli." + sig +
		"\n\nTvoj check-in odkaz (ak chceš rovno potvrdiť, že žiješ):\n" +
		s.checkinURL() + "\n— DMS"
}

func (s *Service) bodyFault(reason string) string {
	return "Self-test DMS ZLYHAL: " + reason + "\n\n" +
		"DMS NEODPÁLI, kým sa to neopraví. Skontroluj službu na serveri.\n— DMS"
}

func nameOrOwner(c Config) string {
	if c.UserEmail != "" {
		return c.UserEmail
	}
	return "vlastník"
}

func humanDur(d time.Duration) string {
	if d >= 24*time.Hour {
		days := int((d + 12*time.Hour) / (24 * time.Hour))
		return fmt.Sprintf("%d dní", days)
	}
	if d >= time.Hour {
		return fmt.Sprintf("%d hodín", int((d+30*time.Minute)/time.Hour))
	}
	return fmt.Sprintf("%d minút", int((d+30*time.Second)/time.Minute))
}
