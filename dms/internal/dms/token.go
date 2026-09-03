package dms

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// token derives an unguessable, stable token for an action (e.g. "checkin" or
// "confirm:friend") from the configured HMAC secret. Tokens live only in
// private emails; even if leaked, the worst outcomes fail safe (a check-in
// merely delays firing; a confirm still needs the human + 7-day delay + the
// metal shares to be of any use).
func (s *Service) token(action string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.HMACSecret))
	mac.Write([]byte(action))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) verifyToken(action, got string) bool {
	want := s.token(action)
	return hmac.Equal([]byte(want), []byte(got))
}
