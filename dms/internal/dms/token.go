package dms

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// token derives an unguessable token for an action from the configured HMAC
// secret. Tokens live only in private e-mails; even if one leaks, the outcomes
// fail safe — a check-in merely delays firing, and a confirmation still needs
// the release delay plus the metal parts to be of any use.
//
// Two actions exist, and neither is a bare constant any more:
//
//	checkin:v<N>        N is checkin_key_version from the config. Bumping it
//	                    and restarting revokes a leaked check-in link.
//	confirm:<id>:<cyc>  cyc is the id of the current waiting cycle, generated
//	                    when the service starts asking and cleared by a
//	                    check-in. A confirmation link therefore dies with its
//	                    cycle and cannot be replayed against a later one.
func (s *Service) token(action string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.HMACSecret))
	mac.Write([]byte(action))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) verifyToken(action, got string) bool {
	want := s.token(action)
	return hmac.Equal([]byte(want), []byte(got))
}

// checkinAction and confirmAction name the two actions; both are derived from
// live state, so callers must not cache them.
func (s *Service) checkinAction() string {
	return fmt.Sprintf("checkin:v%d", s.cfg.CheckinKeyVersion)
}

func (s *Service) confirmAction(id, cycle string) string {
	return "confirm:" + id + ":" + cycle
}

// newCycleID starts a fresh waiting cycle. Confirmation links are bound to it.
func newCycleID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Without randomness we would silently fall back to a predictable
		// cycle; refuse instead — the caller runs at tick time and the
		// service keeps its previous state.
		return ""
	}
	return hex.EncodeToString(b)
}
