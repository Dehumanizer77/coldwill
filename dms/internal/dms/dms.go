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

// Tick performs one evaluation: self-test, health beat, phase transitions, and
// (when due and healthy) firing. Safe to call on any cadence; it is idempotent
// between meaningful time boundaries.
func (s *Service) Tick() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()

	healthy, reason := s.selfTest()
	s.st.Healthy = healthy
	s.checkSignal(now)
	if !healthy {
		if now.Sub(s.st.LastAlertAt) >= s.cfg.AlertInterval.D() {
			s.notifyOwner("[DMS] PORUCHA — skontroluj", s.bodyFault(reason))
			s.st.LastAlertAt = now
		}
	} else if now.Sub(s.st.LastHealthBeatAt) >= s.cfg.HealthBeatInterval.D() {
		s.notifyOwner("[DMS] v poriadku", s.bodyHealth())
		s.st.LastHealthBeatAt = now
	}

	switch s.st.Phase {
	case PhaseNormal:
		silence := now.Sub(s.st.LastCheckIn)
		switch {
		case silence >= s.cfg.SilenceThreshold.D():
			s.st.Phase = PhaseAwaiting
			s.st.LastConfirmReqAt = now
			s.st.LastReminderAt = now
			s.askConfirmers()
			s.notifyOwner("[DMS] Spustená kontrola po dlhom tichu", s.bodyAwaitingUser(silence))
		case silence >= s.cfg.CheckInInterval.D():
			if now.Sub(s.st.LastReminderAt) >= s.cfg.ReminderInterval.D() {
				s.notifyOwner("[DMS] Ozvi sa — check-in", s.bodyCheckin(false))
				s.st.LastReminderAt = now
			}
		}

	case PhaseAwaiting:
		if now.Sub(s.st.LastReminderAt) >= s.cfg.ReminderInterval.D() {
			s.notifyOwner("[DMS] STÁLE čakám — ozvi sa", s.bodyCheckin(true))
			s.st.LastReminderAt = now
		}
		if now.Sub(s.st.LastConfirmReqAt) >= s.cfg.ReminderInterval.D() {
			s.askConfirmers()
			s.st.LastConfirmReqAt = now
		}

	case PhaseCountdown:
		if now.Sub(s.st.LastWarningAt) >= s.cfg.WarningInterval.D() {
			left := s.cfg.ReleaseDelay.D() - now.Sub(s.st.ConfirmedAt)
			s.notifyOwner("[DMS] Obálka sa čoskoro pošle", s.bodyCountdown(left))
			s.st.LastWarningAt = now
		}
		if healthy && now.Sub(s.st.ConfirmedAt) >= s.cfg.ReleaseDelay.D() {
			s.fire(now)
		}

	case PhaseFired:
		// terminal — nothing to do
	}

	s.persist()
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
	}
	log.Printf("dms: check-in recorded (phase now %s)", s.st.Phase)
	s.persist()
}

// Confirm is called when a confirmer attests to the user's death. It starts the
// release countdown. Idempotent if already counting down.
func (s *Service) Confirm(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.notifyOwner("[DMS] Potvrdené — odpočet beží", s.bodyConfirmed(name))
	s.persist()
	return nil
}

// Phase returns the current phase (for status / tests).
func (s *Service) Phase() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st.Phase
}

func (s *Service) fire(now time.Time) {
	data, err := os.ReadFile(s.cfg.EnvelopePath)
	if err != nil || len(data) == 0 {
		s.notifyOwner("[DMS] CHYBA pri výstrele",
			"Obálka sa nedá načítať, výstrel sa NEUSKUTOČNIL. Skontroluj DMS.")
		return
	}
	// One channel through is enough; the phase only advances on a real delivery,
	// so a total failure is retried on the next tick.
	delivered, err := s.notify([]Recipient{s.friend()},
		"Dôležité — dedičstvo: zašifrovaná obálka", s.bodyEnvelope(string(data)))
	if delivered == 0 {
		msg := "Obálku sa nepodarilo doručiť ani jedným kanálom, výstrel sa NEUSKUTOČNIL (skúsi sa znova)."
		if err != nil {
			msg += "\n\n" + err.Error()
		}
		s.notifyOwner("[DMS] CHYBA: obálku sa nepodarilo odoslať", msg)
		return
	}
	s.st.Phase = PhaseFired
	s.st.FiredAt = now
	log.Printf("dms: FIRED — envelope delivered to friend on %d channel(s)", delivered)
	s.notifyOwner("[DMS] Obálka odoslaná",
		"Obálka bola odoslaná príjemcovi ("+s.cfg.FriendEmail+"). Ak je to omyl, kontaktuj ho.")
}

// checkSignal probes the secondary channel. A dead Signal is reported but never
// blocks firing: e-mail is the channel the envelope actually depends on, and a
// second channel must not become a second way for the whole thing to jam.
func (s *Service) checkSignal(now time.Time) {
	if s.sig == nil {
		return
	}
	err := s.sig.Check()
	s.st.SignalOK = err == nil
	if err == nil {
		return
	}
	log.Printf("dms: signal channel degraded: %v", err)
	if now.Sub(s.st.LastSignalAlertAt) >= s.cfg.AlertInterval.D() {
		// The broken channel cannot carry its own alarm, so this goes by e-mail.
		s.mailOwner("[DMS] Signal nefunguje",
			"Druhý kanál (Signal) neodpovedá: "+err.Error()+
				"\n\nE-mail funguje ďalej a DMS beží normálne — oprav Signal, keď budeš môcť.\n— DMS")
		s.st.LastSignalAlertAt = now
	}
}

func (s *Service) selfTest() (bool, string) {
	if len(s.cfg.HMACSecret) < 16 {
		return false, "HMAC secret chýba alebo je krátky"
	}
	if err := s.mail.Check(); err != nil {
		return false, "mail transport: " + err.Error()
	}
	data, err := os.ReadFile(s.cfg.EnvelopePath)
	if err != nil {
		return false, "obálka sa nedá načítať: " + err.Error()
	}
	if len(data) == 0 {
		return false, "obálka je prázdna"
	}
	if !strings.Contains(string(data), "BEGIN PGP MESSAGE") {
		return false, "obálka nie je ASCII-armored PGP správa"
	}
	if err := s.st.save(s.cfg.StatePath); err != nil {
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
	return s.cfg.PublicBaseURL + "/checkin?token=" + s.token("checkin")
}

func (s *Service) confirmURL(c Confirmer) string {
	return s.cfg.PublicBaseURL + "/confirm?id=" + c.ID + "&token=" + s.token("confirm:"+c.ID)
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

func (s *Service) bodyEnvelope(ciphertext string) string {
	return "Ahoj,\n\nak ti prišla táto správa, " + nameOrOwner(s.cfg) +
		" pravdepodobne zomrel alebo je trvalo neschopný.\n\n" +
		"Nižšie je GPG-zašifrovaná obálka — rozšifruj ju svojím kľúčom. Vnútri je\n" +
		"passphrase k peňaženke a pokyny. Pomôž rodine podľa runbooku. Ďakujem.\n\n" +
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
