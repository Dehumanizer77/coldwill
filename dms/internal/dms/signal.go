package dms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SignalSender delivers a plain-text message to Signal numbers. It is the
// secondary channel: e-mail stays primary, and a broken Signal never blocks
// firing (see Service.Tick).
type SignalSender interface {
	Send(numbers []string, message string) error
	Check() error
}

// SignalAPI talks to a signal-cli-rest-api container (bbernhard/signal-cli-rest-api)
// running next to us on loopback. The container holds the linked device; we only
// ever POST plain text to it.
type SignalAPI struct {
	BaseURL string // e.g. http://127.0.0.1:8080
	From    string // registered/linked sender number, E.164
	Client  *http.Client
}

func NewSignalAPI(baseURL, from string, timeout time.Duration) *SignalAPI {
	return &SignalAPI{
		BaseURL: strings.TrimRight(baseURL, "/"),
		From:    from,
		Client:  &http.Client{Timeout: timeout},
	}
}

func (s *SignalAPI) Send(numbers []string, message string) error {
	if len(numbers) == 0 {
		return nil
	}
	body, err := json.Marshal(map[string]any{
		"message":    message,
		"number":     s.From,
		"recipients": numbers,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, s.BaseURL+"/v2/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("signal send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("signal send: %s: %s", resp.Status, snippet(resp.Body))
	}
	return nil
}

// Check verifies the container answers and still has our sender number linked —
// an unlinked device is the realistic silent failure of this channel.
func (s *SignalAPI) Check() error {
	resp, err := s.Client.Get(s.BaseURL + "/v1/accounts")
	if err != nil {
		return fmt.Errorf("signal api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("signal api: %s: %s", resp.Status, snippet(resp.Body))
	}
	var accounts []string
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return fmt.Errorf("signal api: unreadable account list: %w", err)
	}
	for _, a := range accounts {
		if a == s.From {
			return nil
		}
	}
	return fmt.Errorf("signal api: number %s is not linked (accounts: %v)", s.From, accounts)
}

func snippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 256))
	return strings.TrimSpace(string(b))
}
