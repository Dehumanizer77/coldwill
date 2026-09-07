// Command inh-dms is the online dead-man's switch for the Bitcoin inheritance
// system. It periodically asks the owner to check in; after prolonged silence
// it asks trusted confirmers to attest, and after a confirmation + release
// delay it emails a GPG-encrypted envelope (which it can never read) to the
// friend over every channel it has (e-mail, optionally Signal). It self-tests
// and never fires while unhealthy.
//
// Intended to run in Docker on the owner's server, behind Apache (TLS), bound
// to a loopback port.
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"time"

	"inh/dms/internal/dms"
)

func main() {
	cfgPath := flag.String("config", "/data/config.json", "path to config JSON")
	validate := flag.Bool("validate", false, "load and check the config, then exit (deploy preflight)")
	flag.Parse()

	cfg, err := dms.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if *validate {
		rcpts := 0
		for _, e := range cfg.Envelopes {
			rcpts += len(e.To)
		}
		log.Printf("config OK: %s, %d envelope(s) to %d recipient(s), %d confirmer(s), signal %s",
			cfg.PublicBaseURL, len(cfg.Envelopes), rcpts, len(cfg.Confirmers),
			map[bool]string{true: "on", false: "off"}[cfg.Signal.Enabled()])
		return
	}

	mailer := &dms.SMTPMailer{Addr: cfg.SMTPAddr, From: cfg.FromEmail, Timeout: cfg.SMTPTimeout.D()}

	// Signal is optional and secondary: configured -> second channel, absent ->
	// e-mail only. A nil sender keeps every send path on e-mail.
	var signal dms.SignalSender
	if cfg.Signal.Enabled() {
		signal = dms.NewSignalAPI(cfg.Signal.APIURL, cfg.Signal.FromNumber, cfg.Signal.Timeout.D())
		log.Printf("signal channel enabled via %s (from %s)", cfg.Signal.APIURL, cfg.Signal.FromNumber)
	}

	svc, err := dms.New(cfg, time.Now, mailer, signal)
	if err != nil {
		log.Fatalf("init: %v", err)
	}

	if !isLoopback(cfg.ListenAddr) {
		log.Printf("WARNING: listen_addr %q is not loopback — make sure it is firewalled / only reachable via your reverse proxy.", cfg.ListenAddr)
	}

	// Evaluate immediately, then on the configured cadence.
	svc.Tick()
	go func() {
		t := time.NewTicker(cfg.TickInterval.D())
		defer t.Stop()
		for range t.C {
			svc.Tick()
		}
	}()

	log.Printf("inh-dms listening on %s (public base %s); tick every %s", cfg.ListenAddr, cfg.PublicBaseURL, cfg.TickInterval.D())
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, svc.Handler()))
}

func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
