package dms

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Duration is a time.Duration that unmarshals from JSON strings like "30d",
// "12h", "90m". Plain Go durations (time.ParseDuration) also work; the extra
// "d" suffix means days.
type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := parseDur(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func (d Duration) D() time.Duration { return time.Duration(d) }

// maxDays bounds the "d" suffix so the multiplication below cannot overflow
// int64 nanoseconds (~292 years) and silently produce a negative duration.
const maxDays = 100000

func parseDur(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid day duration %q: %w", s, err)
		}
		if n < -maxDays || n > maxDays {
			return 0, fmt.Errorf("day duration %q is out of range (max %d days)", s, maxDays)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

type Confirmer struct {
	ID     string `json:"id"`               // short token id, e.g. "friend"
	Name   string `json:"name"`             // display name
	Email  string `json:"email"`            // where the confirmation request is sent
	Signal string `json:"signal,omitempty"` // optional E.164 number, second channel
}

// SignalConfig points at a signal-cli-rest-api container. Signal is the
// secondary channel; leave it out and everything runs on e-mail alone.
type SignalConfig struct {
	APIURL     string   `json:"api_url"`     // e.g. http://127.0.0.1:8080
	FromNumber string   `json:"from_number"` // linked sender number, E.164
	Timeout    Duration `json:"timeout"`     // per-request timeout
}

func (s SignalConfig) Enabled() bool { return s.APIURL != "" && s.FromNumber != "" }

// EnvelopeRecipient is one person an envelope is delivered to. Who can actually
// open it is decided when the envelope is encrypted, offline; this only says
// where the ciphertext is sent.
type EnvelopeRecipient struct {
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
	Signal string `json:"signal,omitempty"`
}

// Envelope is one sealed message with its own recipients. Several envelopes let
// the same secret reach more than one person (redundancy) or different secrets
// reach different people (separation), without the DMS ever reading any of them.
type Envelope struct {
	ID      string              `json:"id"`                // short, stable; used in state and logs
	Path    string              `json:"path"`              // GPG ciphertext on disk
	Subject string              `json:"subject,omitempty"` // optional, defaults below
	Note    string              `json:"note,omitempty"`    // optional line for the recipients
	To      []EnvelopeRecipient `json:"to"`
}

func (e Envelope) subject() string {
	if e.Subject != "" {
		return e.Subject
	}
	return "Dôležité — dedičstvo: zašifrovaná obálka"
}

func (e Envelope) recipients() []Recipient {
	out := make([]Recipient, 0, len(e.To))
	for _, t := range e.To {
		out = append(out, Recipient{Name: t.Name, Email: t.Email, Signal: t.Signal})
	}
	return out
}

type Config struct {
	ListenAddr    string   `json:"listen_addr"`            // local port, behind Apache, e.g. 127.0.0.1:8088
	PublicBaseURL string   `json:"public_base_url"`        // e.g. https://dms.example.com
	SMTPAddr      string   `json:"smtp_addr"`              // local postfix, e.g. 127.0.0.1:25
	SMTPTimeout   Duration `json:"smtp_timeout,omitempty"` // whole conversation; default 30s
	FromEmail     string   `json:"from_email"`
	UserEmail     string   `json:"user_email"`   // me (check-in / health / warnings)
	FriendEmail   string   `json:"friend_email"` // envelope recipient on fire (the friend)

	UserSignal   string       `json:"user_signal,omitempty"`   // my E.164 number, second channel
	FriendSignal string       `json:"friend_signal,omitempty"` // friend's E.164 number, second channel
	Signal       SignalConfig `json:"signal,omitempty"`        // Signal transport (omit = e-mail only)

	Confirmers []Confirmer `json:"confirmers"`

	// ConfirmQuorum is how many different confirmers must attest before the
	// countdown starts. Default 1: one person suffices, which is the deliberate
	// trade — a false confirmation is survivable (the envelope is useless
	// without the metal parts) while nobody confirming is not. Raise it only if
	// you would rather risk the inheritance stalling than a premature release.
	ConfirmQuorum int `json:"confirm_quorum,omitempty"`

	// Envelopes is the general form. envelope_path + friend_email below are the
	// older single-envelope config and still work; applyDefaults folds them in.
	Envelopes []Envelope `json:"envelopes,omitempty"`

	EnvelopePath string `json:"envelope_path,omitempty"` // GPG-encrypted envelope (ciphertext only)
	StatePath    string `json:"state_path"`              // JSON state file
	HMACSecret   string `json:"hmac_secret"`             // random secret for link tokens

	// CheckinKeyVersion revokes a leaked check-in link: bump it and restart,
	// and every previously issued check-in URL stops working. The next reminder
	// or health beat carries the new one.
	CheckinKeyVersion int `json:"checkin_key_version,omitempty"`

	CheckInInterval    Duration `json:"check_in_interval"`    // remind me to check in
	ReminderInterval   Duration `json:"reminder_interval"`    // gap between reminders
	SilenceThreshold   Duration `json:"silence_threshold"`    // silence -> ask confirmers
	ReleaseDelay       Duration `json:"release_delay"`        // confirm -> fire
	HealthBeatInterval Duration `json:"health_beat_interval"` // DMS -> me "healthy"
	WarningInterval    Duration `json:"warning_interval"`     // countdown warnings to me
	AlertInterval      Duration `json:"alert_interval"`       // min gap between fault alerts
	TickInterval       Duration `json:"tick_interval"`        // evaluation cadence
}

func LoadConfig(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	c.applyDefaults()
	return c, c.validate()
}

func (c *Config) applyDefaults() {
	set := func(d *Duration, def time.Duration) {
		if d.D() == 0 {
			*d = Duration(def)
		}
	}
	set(&c.CheckInInterval, 30*24*time.Hour)
	set(&c.ReminderInterval, 7*24*time.Hour)
	set(&c.SilenceThreshold, 60*24*time.Hour)
	set(&c.ReleaseDelay, 7*24*time.Hour)
	set(&c.HealthBeatInterval, 7*24*time.Hour)
	set(&c.WarningInterval, 24*time.Hour)
	set(&c.AlertInterval, 24*time.Hour)
	set(&c.TickInterval, time.Hour)
	if c.SMTPAddr == "" {
		c.SMTPAddr = "127.0.0.1:25"
	}
	if c.ListenAddr == "" {
		c.ListenAddr = "127.0.0.1:8088"
	}
	if c.Signal.Timeout.D() == 0 {
		c.Signal.Timeout = Duration(20 * time.Second)
	}
	if c.SMTPTimeout.D() == 0 {
		c.SMTPTimeout = Duration(DefaultMailTimeout)
	}
	if c.ConfirmQuorum == 0 {
		c.ConfirmQuorum = 1
	}
	// Single-envelope config keeps working: fold it into the general form so the
	// rest of the service only ever deals with a list.
	if len(c.Envelopes) == 0 && c.EnvelopePath != "" {
		c.Envelopes = []Envelope{{
			ID:   "default",
			Path: c.EnvelopePath,
			To:   []EnvelopeRecipient{{Email: c.FriendEmail, Signal: c.FriendSignal}},
		}}
	}
	for i := range c.Envelopes {
		if c.Envelopes[i].ID == "" {
			c.Envelopes[i].ID = fmt.Sprintf("envelope-%d", i+1)
		}
	}
}

func (c Config) validate() error {
	switch {
	case c.PublicBaseURL == "":
		return fmt.Errorf("public_base_url required")
	case c.FromEmail == "":
		return fmt.Errorf("from_email required")
	case c.UserEmail == "":
		return fmt.Errorf("user_email required")
	case c.StatePath == "":
		return fmt.Errorf("state_path required")
	case len(c.HMACSecret) < 16:
		return fmt.Errorf("hmac_secret must be at least 16 chars")
	case c.CheckinKeyVersion < 0:
		return fmt.Errorf("checkin_key_version must not be negative")
	case len(c.Confirmers) == 0:
		return fmt.Errorf("at least one confirmer required")
	case c.ConfirmQuorum < 1:
		return fmt.Errorf("confirm_quorum must be at least 1")
	case c.ConfirmQuorum > len(c.Confirmers):
		return fmt.Errorf("confirm_quorum %d exceeds the %d configured confirmers — nobody could ever release",
			c.ConfirmQuorum, len(c.Confirmers))
	}
	// Every interval must be positive. A negative release_delay would let a
	// confirmation fire on the very next tick with no grace period at all, and a
	// negative tick_interval panics time.NewTicker at startup.
	for _, iv := range []struct {
		name string
		d    Duration
	}{
		{"check_in_interval", c.CheckInInterval},
		{"reminder_interval", c.ReminderInterval},
		{"silence_threshold", c.SilenceThreshold},
		{"release_delay", c.ReleaseDelay},
		{"health_beat_interval", c.HealthBeatInterval},
		{"warning_interval", c.WarningInterval},
		{"alert_interval", c.AlertInterval},
		{"tick_interval", c.TickInterval},
		{"signal.timeout", c.Signal.Timeout},
		{"smtp_timeout", c.SMTPTimeout},
	} {
		if iv.d.D() <= 0 {
			return fmt.Errorf("%s must be positive, got %s", iv.name, iv.d.D())
		}
	}
	if c.SilenceThreshold.D() <= c.CheckInInterval.D() {
		return fmt.Errorf("silence_threshold must exceed check_in_interval")
	}
	if err := c.validateEnvelopes(); err != nil {
		return err
	}
	return c.validateSignal()
}

// validateEnvelopes refuses anything that would fail silently years from now:
// no envelope at all, an envelope nobody receives, or two envelopes sharing an
// id (the id is how delivery is remembered across retries).
func (c Config) validateEnvelopes() error {
	if len(c.Envelopes) == 0 {
		return fmt.Errorf("at least one envelope required (envelopes[], or envelope_path + friend_email)")
	}
	ids := make(map[string]bool, len(c.Envelopes))
	for _, e := range c.Envelopes {
		if ids[e.ID] {
			return fmt.Errorf("duplicate envelope id %q", e.ID)
		}
		ids[e.ID] = true
		if e.Path == "" {
			return fmt.Errorf("envelope %q: path required", e.ID)
		}
		if len(e.To) == 0 {
			return fmt.Errorf("envelope %q: at least one recipient required", e.ID)
		}
		for i, t := range e.To {
			if t.Email == "" && t.Signal == "" {
				return fmt.Errorf("envelope %q: recipient %d has neither email nor signal", e.ID, i+1)
			}
		}
	}
	return nil
}

// validateSignal refuses half-configured Signal: a number without a transport
// would silently drop that channel, which is exactly what must not happen
// quietly in a system nobody looks at for years.
func (c Config) validateSignal() error {
	numbers := map[string]string{"user_signal": c.UserSignal, "friend_signal": c.FriendSignal}
	for _, cf := range c.Confirmers {
		numbers["confirmer "+cf.ID+" signal"] = cf.Signal
	}
	for _, e := range c.Envelopes {
		for i, t := range e.To {
			numbers[fmt.Sprintf("envelope %s recipient %d signal", e.ID, i+1)] = t.Signal
		}
	}
	used := false
	for what, n := range numbers {
		if n == "" {
			continue
		}
		used = true
		if !strings.HasPrefix(n, "+") {
			return fmt.Errorf("%s: %q must be an E.164 number starting with +", what, n)
		}
	}
	if !c.Signal.Enabled() {
		if used {
			return fmt.Errorf("signal numbers configured but signal.api_url/from_number missing")
		}
		if c.Signal.APIURL != "" || c.Signal.FromNumber != "" {
			return fmt.Errorf("signal needs both api_url and from_number")
		}
		return nil
	}
	if !strings.HasPrefix(c.Signal.FromNumber, "+") {
		return fmt.Errorf("signal.from_number: %q must be an E.164 number starting with +", c.Signal.FromNumber)
	}
	if !used {
		return fmt.Errorf("signal configured but no recipient has a signal number")
	}
	return nil
}
