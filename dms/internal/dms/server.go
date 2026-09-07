package dms

import (
	"fmt"
	"net/http"
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
	writePage(w, "inh DMS", `<p>Služba beží.</p>`)
}

func (s *Service) hCheckin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !s.verifyToken(s.checkinAction(), r.FormValue("token")) {
			forbidden(w)
			return
		}
		s.CheckIn()
		writePage(w, "Zaznamenané", `<p class="ok">✓ Ďakujem — zaznamenané, že žiješ. Časovač je vynulovaný.</p>`)
		return
	}
	if !s.verifyToken(s.checkinAction(), r.URL.Query().Get("token")) {
		forbidden(w)
		return
	}
	writePage(w, "Check-in", fmt.Sprintf(`
		<p>Potvrď, že si v poriadku:</p>
		<form method="post" action="/checkin">
			<input type="hidden" name="token" value="%s">
			<button type="submit">Som živý/á — vynulovať časovač</button>
		</form>`, s.token(s.checkinAction())))
}

func (s *Service) hConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		id := r.FormValue("id")
		if !s.verifyToken(s.confirmAction(id, s.currentCycle()), r.FormValue("token")) {
			forbidden(w)
			return
		}
		if err := s.Confirm(id); err != nil {
			writePage(w, "Nedá sa potvrdiť", `<p class="err">`+htmlEscape(err.Error())+`</p>`)
			return
		}
		got, need := s.Confirmations()
		if got < need {
			writePage(w, "Potvrdené", fmt.Sprintf(
				`<p class="ok">✓ Potvrdené. Zatiaľ %d z %d potrebných potvrdení — odpočet sa spustí, keď potvrdia aj ostatní.</p>`,
				got, need))
			return
		}
		writePage(w, "Potvrdené", `<p class="ok">✓ Potvrdené. Obálky sa odošlú po uplynutí ochrannej lehoty, ak vlastník medzitým nepotvrdí, že žije.</p>`)
		return
	}
	id := r.URL.Query().Get("id")
	cycle := s.currentCycle()
	if !s.verifyToken(s.confirmAction(id, cycle), r.URL.Query().Get("token")) {
		forbidden(w)
		return
	}
	writePage(w, "Potvrdenie", fmt.Sprintf(`
		<p><strong>Pozor — vážny krok.</strong> Potvrdením vyhlasuješ, že vlastník zomrel
		alebo je trvalo neschopný. Spustí sa odovzdanie prístupov rodine
		(zašifrovaná obálka po ochrannej lehote). Ak si nie si istý, NEPOTVRDZUJ.</p>
		<form method="post" action="/confirm">
			<input type="hidden" name="id" value="%s">
			<input type="hidden" name="token" value="%s">
			<button type="submit">Potvrdzujem úmrtie / trvalú neschopnosť</button>
		</form>`, htmlEscape(id), s.token(s.confirmAction(id, cycle))))
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

func forbidden(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	writePage(w, "Neplatný odkaz", `<p class="err">Neplatný alebo poškodený odkaz.</p>`)
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
