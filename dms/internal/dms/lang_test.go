package dms

import (
	"strings"
	"testing"
	"time"
)

// The people this service writes to live in different places, so each of them
// gets their own language while the owner keeps his.
func TestEachRecipientGetsTheirOwnLanguage(t *testing.T) {
	cfg := testConfig(t)
	dir := t.TempDir()
	cfg.UserLang = "sk"
	cfg.EnvelopePath, cfg.FriendEmail = "", ""
	cfg.Confirmers = []Confirmer{
		{ID: "friend", Name: "Adam", Email: "adam@example.com", Lang: "en"},
		{ID: "brother", Name: "Boris", Email: "boris@example.com", Lang: "sk"},
	}
	cfg.Envelopes = []Envelope{{ID: "passphrase", Path: writeEnvelope(t, dir, "p.asc"),
		To: []EnvelopeRecipient{
			{Name: "Cyril", Email: "cyril@example.com", Lang: "en"},
			{Name: "Dana", Email: "dana@example.com", Lang: "sk"},
		}}}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatalf("config: %v", err)
	}
	c, fm := newClock(), &fakeMailer{}
	svc, err := New(cfg, c.now, fm, nil)
	if err != nil {
		t.Fatal(err)
	}

	svc.Tick()
	// The owner asked for Slovak, so his own health beat is Slovak.
	if _, ok := fm.sentTo("me@example", "v poriadku"); !ok {
		t.Errorf("the owner's health beat was not in Slovak")
	}

	c.add(61 * 24 * time.Hour)
	svc.Tick()
	if m, ok := fm.sentTo("adam@example.com", "Request for confirmation"); !ok {
		t.Errorf("Adam's request was not in English")
	} else if !strings.Contains(m.body, "Hello Adam") {
		t.Errorf("Adam's request body was not in English: %q", firstLineOf(m.body))
	}
	if m, ok := fm.sentTo("boris@example.com", "Prosba o potvrdenie"); !ok {
		t.Errorf("Boris's request was not in Slovak")
	} else if !strings.Contains(m.body, "Ahoj Boris") {
		t.Errorf("Boris's request body was not in Slovak: %q", firstLineOf(m.body))
	}

	if err := svc.Confirm("friend"); err != nil {
		t.Fatal(err)
	}
	c.add(7 * 24 * time.Hour)
	fm.reset()
	svc.Tick()

	if svc.Phase() != PhaseFired {
		t.Fatalf("phase = %s, want fired", svc.Phase())
	}
	if m, ok := fm.sentTo("cyril@example.com", "envelope"); !ok {
		t.Errorf("Cyril's envelope was not in English")
	} else if !strings.Contains(m.body, "probably died") {
		t.Errorf("Cyril's envelope body was not in English")
	}
	if m, ok := fm.sentTo("dana@example.com", "obálka"); !ok {
		t.Errorf("Dana's envelope was not in Slovak")
	} else if !strings.Contains(m.body, "pravdepodobne zomrel") {
		t.Errorf("Dana's envelope body was not in Slovak")
	}
}

// Durations are spelled out per language, not baked into one string.
func TestDurationsFollowTheLanguage(t *testing.T) {
	if got := humanDur("en", 7*24*time.Hour); got != "7 days" {
		t.Errorf("English duration = %q", got)
	}
	if got := humanDur("sk", 7*24*time.Hour); got != "7 dní" {
		t.Errorf("Slovak duration = %q", got)
	}
	// Czech has no catalogue yet, so it falls back to English rather than blank.
	if got := humanDur("cs", 7*24*time.Hour); got != "7 days" {
		t.Errorf("Czech duration = %q, want the English fallback", got)
	}
}

func firstLineOf(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
