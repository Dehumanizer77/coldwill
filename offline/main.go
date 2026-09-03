// Command inh-offline is the air-gapped Setup/Recovery tool for the Bitcoin
// inheritance system. It serves a small web UI on a loopback address only and
// never touches the network.
//
//   - Setup:    generate a 128-bit key-file and split it into 2-of-3 SLIP-39
//     metal shares.
//   - Recovery: combine 2 shares back into the key-file (to open the KeePass DB).
//
// Run it on an OFFLINE machine; it opens your default browser automatically
// (use --open=false to disable). Close the tool when done.
package main

import (
	"embed"
	"flag"
	"html/template"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"inh/offline/internal/server"
)

//go:embed web/*.html
var webFS embed.FS

//go:embed web/style.css
var styleCSS []byte

func main() {
	addr := flag.String("addr", "127.0.0.1:8777", "loopback listen address (must be 127.0.0.1/::1/localhost)")
	open := flag.Bool("open", true, "automatically open the default web browser")
	flag.Parse()

	if err := server.SelfTest(); err != nil {
		log.Fatalf("SLIP-39 self-test FAILED — refusing to run: %v", err)
	}
	if !loopbackOnly(*addr) {
		log.Fatalf("refusing to listen on non-loopback address %q (air-gap safety)", *addr)
	}

	tmpl := template.Must(template.ParseFS(webFS, "web/*.html"))
	srv := server.New(tmpl, styleCSS)

	// Bind first so the socket is ready before we launch the browser.
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %s: %v", *addr, err)
	}
	url := "http://" + ln.Addr().String() + "/"

	log.Printf("SLIP-39 self-test OK.")
	log.Printf("inh offline tool ready — %s (open this on the AIR-GAPPED machine).", url)
	log.Printf("Press Ctrl+C when finished.")

	if *open {
		if err := openBrowser(url); err != nil {
			log.Printf("could not auto-open a browser (%v) — open %s manually.", err, url)
		}
	}

	log.Fatal(http.Serve(ln, srv))
}

func loopbackOnly(addr string) bool {
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

// openBrowser opens url in the OS-default browser, independent of which browser
// is installed. Best-effort: the error is returned so the caller can fall back
// to printing the URL.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default: // linux, *bsd, ...
		return exec.Command("xdg-open", url).Start()
	}
}
