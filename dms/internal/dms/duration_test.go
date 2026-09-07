package dms

import (
	"strings"
	"testing"
	"time"
)

// A negative release_delay would let a confirmation fire immediately, and a
// negative tick_interval panics time.NewTicker — neither may pass validation.
func TestNonPositiveIntervalsRejected(t *testing.T) {
	cases := map[string]func(*Config){
		"release_delay":        func(c *Config) { c.ReleaseDelay = Duration(-time.Hour) },
		"tick_interval":        func(c *Config) { c.TickInterval = Duration(-time.Second) },
		"warning_interval":     func(c *Config) { c.WarningInterval = Duration(-24 * time.Hour) },
		"check_in_interval":    func(c *Config) { c.CheckInInterval = Duration(-time.Hour) },
		"health_beat_interval": func(c *Config) { c.HealthBeatInterval = Duration(-time.Hour) },
		"alert_interval":       func(c *Config) { c.AlertInterval = Duration(-time.Hour) },
		"reminder_interval":    func(c *Config) { c.ReminderInterval = Duration(-time.Hour) },
		"signal.timeout":       func(c *Config) { c.Signal.Timeout = Duration(-time.Second) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := testConfig(t)
			mutate(&cfg)
			cfg.applyDefaults()
			err := cfg.validate()
			if err == nil {
				t.Fatalf("negative %s accepted", name)
			}
			if !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "positive") {
				t.Fatalf("err = %v, want it to name %s and say it must be positive", err, name)
			}
		})
	}
}

// Zero still means "use the default" — that is how the example config leaves
// most intervals out.
func TestZeroIntervalsStillTakeDefaults(t *testing.T) {
	cfg := testConfig(t)
	cfg.ReleaseDelay, cfg.TickInterval = 0, 0
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatalf("zero intervals should fall back to defaults: %v", err)
	}
	if cfg.ReleaseDelay.D() != 7*24*time.Hour || cfg.TickInterval.D() != time.Hour {
		t.Fatalf("defaults not applied: release=%s tick=%s", cfg.ReleaseDelay.D(), cfg.TickInterval.D())
	}
}

func TestParseDurationRange(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr string
	}{
		{in: "30d", want: 30 * 24 * time.Hour},
		{in: "12h", want: 12 * time.Hour},
		{in: "90m", want: 90 * time.Minute},
		{in: "-1h", want: -time.Hour}, // parses; validation is what rejects it
		{in: "100001d", wantErr: "out of range"},
		{in: "999999999999999d", wantErr: "out of range"},
		{in: "-100001d", wantErr: "out of range"},
		{in: "abc", wantErr: "invalid"},
	}
	for _, c := range cases {
		got, err := parseDur(c.in)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("parseDur(%q): unexpected error %v", c.in, err)
		case c.wantErr == "" && got != c.want:
			t.Errorf("parseDur(%q) = %s, want %s", c.in, got, c.want)
		case c.wantErr != "" && err == nil:
			t.Errorf("parseDur(%q) = %s, want an error mentioning %q", c.in, got, c.wantErr)
		case c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr):
			t.Errorf("parseDur(%q) error = %v, want it to mention %q", c.in, err, c.wantErr)
		}
	}
	// The overflow the bound protects: without it this wraps to a negative duration.
	if d := time.Duration(maxDays) * 24 * time.Hour; d <= 0 {
		t.Fatalf("maxDays still overflows: %s", d)
	}
}
