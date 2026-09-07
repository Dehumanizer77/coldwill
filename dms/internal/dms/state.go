package dms

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Phase of the dead-man's switch.
const (
	PhaseNormal    = "normal"                // alive; counting toward check-in/silence
	PhaseAwaiting  = "awaiting_confirmation" // silent too long; confirmers asked
	PhaseCountdown = "countdown"             // confirmed; release delay running
	PhaseFired     = "fired"                 // envelope released
)

// State is the persisted runtime state. It contains NO secrets.
type State struct {
	Phase            string    `json:"phase"`
	LastCheckIn      time.Time `json:"last_check_in"`
	ConfirmedAt      time.Time `json:"confirmed_at,omitempty"`
	ConfirmedBy      string    `json:"confirmed_by,omitempty"`
	FiredAt          time.Time `json:"fired_at,omitempty"`
	LastReminderAt   time.Time `json:"last_reminder_at,omitempty"`
	LastConfirmReqAt time.Time `json:"last_confirm_req_at,omitempty"`
	LastWarningAt    time.Time `json:"last_warning_at,omitempty"`
	LastHealthBeatAt time.Time `json:"last_health_beat_at,omitempty"`
	LastAlertAt      time.Time `json:"last_alert_at,omitempty"`
	Healthy          bool      `json:"healthy"`

	// CycleID identifies the current waiting cycle; confirmation links are
	// bound to it, so they die when a check-in clears it or a new cycle starts.
	CycleID string `json:"cycle_id,omitempty"`

	// Envelopes already delivered, by id. A partial fire retries only the rest,
	// so nobody gets the same envelope twice while another is still stuck.
	Delivered []string `json:"delivered,omitempty"`

	// Secondary channel health; degraded Signal alerts but never blocks firing.
	SignalOK          bool      `json:"signal_ok"`
	LastSignalAlertAt time.Time `json:"last_signal_alert_at,omitempty"`
}

// loadState reads the state file, or initialises a fresh "alive now" state if
// the file does not exist. now is used to seed LastCheckIn on first run.
func loadState(path string, now time.Time) (State, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return State{Phase: PhaseNormal, LastCheckIn: now, Healthy: true}, nil
	}
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	if s.Phase == "" {
		s.Phase = PhaseNormal
	}
	return s, nil
}

// save writes the state atomically (temp file + rename).
func (s State) save(path string) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s State) wasDelivered(id string) bool {
	for _, d := range s.Delivered {
		if d == id {
			return true
		}
	}
	return false
}

// stateWritable checks that the state directory still accepts writes, without
// touching the state file itself — the self-test runs outside the state lock.
func stateWritable(path string) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".probe-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

func ensureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}
