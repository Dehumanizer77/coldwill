package dms

import (
	"fmt"
	"net/http"

	"coldwill/dms/internal/i18n"
)

// Handler returns the HTTP routes. Check-in and confirm are two-step (GET shows
// a page with a button, POST performs the action) so that automatic link
// prefetching by mail scanners cannot trigger them.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.hRoot)
	mux.HandleFunc("/checkin", s.hCheckin)
	mux.HandleFunc("/confirm", s.hConfirm)
	return mux
}

func (s *Service) hRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	l := s.pageLang(r)
	writePage(w, "coldwill switch", "<p>"+string(i18n.T(l, "page.running"))+"</p>")
}

func (s *Service) hCheckin(w http.ResponseWriter, r *http.Request) {
	l := s.pageLang(r)
	if r.Method == http.MethodPost {
		if !s.verifyToken(s.checkinAction(), r.FormValue("token")) {
			forbidden(w, l)
			return
		}
		s.CheckIn()
		writePage(w, i18n.S(l, "page.checkin.done.title"),
			`<p class="ok">`+string(i18n.T(l, "page.checkin.done"))+`</p>`)
		return
	}
	if !s.verifyToken(s.checkinAction(), r.URL.Query().Get("token")) {
		forbidden(w, l)
		return
	}
	writePage(w, i18n.S(l, "page.checkin.title"), fmt.Sprintf(`
		<p>%s</p>
		<form method="post" action="/checkin">
			<input type="hidden" name="token" value="%s">
			<button type="submit">%s</button>
		</form>`, i18n.T(l, "page.checkin.p"), s.token(s.checkinAction()), i18n.T(l, "page.checkin.btn")))
}

func (s *Service) hConfirm(w http.ResponseWriter, r *http.Request) {
	l := s.pageLang(r)
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		if !s.verifyToken(s.confirmAction(id, s.currentCycle()), r.FormValue("token")) {
			forbidden(w, l)
			return
		}
		if err := s.Confirm(id); err != nil {
			writePage(w, i18n.S(l, "page.confirm.failed.title"), `<p class="err">`+htmlEscape(err.Error())+`</p>`)
			return
		}
		got, need := s.Confirmations()
		if got < need {
			writePage(w, i18n.S(l, "page.confirm.done.title"),
				`<p class="ok">`+string(i18n.T(l, "page.confirm.partial", got, need))+`</p>`)
			return
		}
		writePage(w, i18n.S(l, "page.confirm.done.title"),
			`<p class="ok">`+string(i18n.T(l, "page.confirm.done"))+`</p>`)
		return
	}
	id := r.URL.Query().Get("id")
	cycle := s.currentCycle()
	if !s.verifyToken(s.confirmAction(id, cycle), r.URL.Query().Get("token")) {
		forbidden(w, l)
		return
	}
	writePage(w, i18n.S(l, "page.confirm.title"), fmt.Sprintf(`
		<p>%s</p>
		<form method="post" action="/confirm">
			<input type="hidden" name="id" value="%s">
			<input type="hidden" name="token" value="%s">
			<button type="submit">%s</button>
		</form>`, i18n.T(l, "page.confirm.warn"), htmlEscape(id),
		s.token(s.confirmAction(id, cycle)), i18n.T(l, "page.confirm.btn")))
}

func writePage(w http.ResponseWriter, title, bodyHTML string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><html lang="sk"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1"><title>%s</title>
<style>body{font:16px/1.5 system-ui,sans-serif;max-width:640px;margin:40px auto;padding:0 20px}
button{font:inherit;padding:12px 18px;border:0;border-radius:8px;background:#2563eb;color:#fff;cursor:pointer}
.ok{color:#15803d}.err{color:#b91c1c}</style></head><body><h1>%s</h1>%s</body></html>`,
		title, title, bodyHTML)
}

func forbidden(w http.ResponseWriter, l i18n.Lang) {
	w.WriteHeader(http.StatusForbidden)
	writePage(w, i18n.S(l, "page.badlink.title"),
		`<p class="err">`+string(i18n.T(l, "page.badlink"))+`</p>`)
}

// pageLang picks the language for a page. A confirmer's link carries their id,
// so the page can greet them in the language configured for them; everything
// else is the owner's language.
func (s *Service) pageLang(r *http.Request) i18n.Lang {
	if id := r.FormValue("id"); id != "" {
		for _, c := range s.cfg.Confirmers {
			if c.ID == id {
				return i18n.Parse(c.Lang)
			}
		}
	}
	if id := r.URL.Query().Get("id"); id != "" {
		for _, c := range s.cfg.Confirmers {
			if c.ID == id {
				return i18n.Parse(c.Lang)
			}
		}
	}
	return i18n.Parse(s.cfg.UserLang)
}

func htmlEscape(s string) string {
	r := ""
	for _, c := range s {
		switch c {
		case '<':
			r += "&lt;"
		case '>':
			r += "&gt;"
		case '&':
			r += "&amp;"
		case '"':
			r += "&quot;"
		default:
			r += string(c)
		}
	}
	return r
}
