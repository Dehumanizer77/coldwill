package i18n

// csMessages is the Czech translation.
//
// Subjects stay blunt for the same reason as the English ones: they arrive
// months or years later, into an inbox that has forgotten this system exists.
var csMessages = map[string]string{
	"subj.fault":      "[DMS] PORUCHA, zkontroluj to",
	"subj.healthy":    "[DMS] v pořádku",
	"subj.checkin":    "[DMS] Ozvi se, prosím",
	"subj.stillwait":  "[DMS] STÁLE čekám, ozvi se",
	"subj.awaiting":   "[DMS] Spuštěna kontrola po dlouhém tichu",
	"subj.confirmreq": "[DMS] Prosba o potvrzení",
	"subj.confirmed":  "[DMS] Potvrzeno, odpočet běží",
	"subj.partial":    "[DMS] Potvrzení přijato, čeká se na další",
	"subj.countdown":  "[DMS] Obálka se brzy odešle",
	"subj.sendfail":   "[DMS] CHYBA: obálku se nepodařilo odeslat",
	"subj.sent":       "[DMS] Obálka odeslána",
	"subj.cancelled":  "[DMS] Odesílání zrušeno tvým check-inem",
	"subj.signaldown": "[DMS] Signal nefunguje",
	"subj.envelope":   "Důležité, dědictví: obálka",

	"body.checkin":        "Klikni a potvrď, že jsi v pořádku:\n\n%s\n\nPokud se neozveš, spustí se proces předání přístupů rodině.\n— DMS",
	"body.checkin.urgent": "Už dlouho ses neozval. Pokud žiješ, OKAMŽITĚ potvrď:\n\n%s\n\nPokud se neozveš, spustí se proces předání přístupů rodině.\n— DMS",
	"body.awaiting":       "Neozval ses %s. Požádal jsem důvěryhodné osoby o potvrzení.\n\nPokud žiješ, OKAMŽITĚ proces zruš:\n\n%s\n— DMS",
	"body.confirmed":      "%s potvrdil. Obálka se odešle za %s.\n\nPokud je to omyl a žiješ, ZRUŠ to teď:\n\n%s\n— DMS",
	"body.partial":        "%s potvrdil. Ke spuštění odpočtu je potřeba %d potvrzení, zatím jsou %d.\n\nPokud žiješ, ZRUŠ to teď:\n\n%s\n— DMS",
	"body.countdown":      "Obálka se odešle přibližně za %s.\n\nPokud žiješ, ZRUŠ to:\n\n%s\n— DMS",
	"body.healthy":        "Služba běží a autotesty prošly.%s\n\nTvůj check-in odkaz (pokud chceš rovnou potvrdit, že žiješ):\n%s\n— DMS",

	"body.healthy.signal.ok":  "\nKanál Signal: v pořádku.",
	"body.healthy.signal.bad": "\nKanál Signal: NEFUNGUJE (e-mail běží dál).",

	"body.fault":      "Autotest služby SELHAL: %s\n\nDokud se to neopraví, nic neodešle. Zkontroluj službu na serveru.\n— DMS",
	"body.signaldown": "Druhý kanál (Signal) neodpovídá: %s\n\nE-mail běží dál a služba funguje normálně. Oprav Signal, až budeš moct.\n— DMS",
	"body.envfail":    "Soubor s obálkou nejde načíst nebo to není PDF, takže se nic neodeslalo. Zkusí se to znovu; zkontroluj službu.",
	"body.sendfail":   "Obálku se nepodařilo doručit ani jedním kanálem, takže se nic neodeslalo: %s\nZkusí se to znovu při dalším tiku.",
	"body.sent":       "Obálka odešla hlavnímu dědici (%s).\nPokud je to omyl, dej mu vědět.",
	"body.cancelled":  "Během odesílání přišel tvůj check-in, ale obálka už stihla odejít hlavnímu dědici (%s). Vzít zpět se nedá, dej mu vědět, že šlo o planý poplach.",
	"body.cycleid":    "Nepodařilo se vygenerovat id cyklu (selhal generátor náhody); výzva potvrzovatelům se NEODESLALA. Zkusí se to znovu.",
	"body.raw":        "%s\n— DMS",

	"body.confirmreq": "Ahoj %s,\n\ntoto je automatická zpráva. %s se delší dobu neozval.\n\nPOKUD můžeš potvrdit, že zemřel nebo je trvale neschopný, otevři odkaz a potvrď tlačítkem.\nTím se po %s odešle obálka hlavnímu dědici.\nPOKUD to potvrdit nemůžeš, nedělej nic.\n\n%s\n— DMS",

	"body.envelope": "Ahoj,\n\npokud ti přišla tato zpráva, %s pravděpodobně zemřel nebo je trvale neschopný.\n\nV příloze je obálka, o které mluví runbook: tentýž text, který je uložený v bankovním trezoru. Sám o sobě nic neprozradí. Jak z něj přečíst passphrase, je napsané v databázi hesel.\n\nVezmi runbook a zavolej technicky zdatné osobě, která je v něm uvedená. Pomůže ti se zbytkem.\n— DMS",

	"page.running":              "Služba běží.",
	"page.badlink.title":        "Neplatný odkaz",
	"page.badlink":              "Neplatný nebo poškozený odkaz.",
	"page.checkin.title":        "Check-in",
	"page.checkin.p":            "Potvrď, že jsi v pořádku:",
	"page.checkin.btn":          "Jsem naživu, vynulovat časovač",
	"page.checkin.done.title":   "Zaznamenáno",
	"page.checkin.done":         "✓ Děkuji, je zaznamenáno, že žiješ. Časovač je vynulovaný.",
	"page.confirm.title":        "Potvrzení",
	"page.confirm.warn":         "<strong>Pozor, vážný krok.</strong> Potvrzením prohlašuješ, že vlastník zemřel nebo je trvale neschopný. Spustí se předání přístupů rodině (obálka půjde hlavnímu dědici po ochranné lhůtě). Pokud si nejsi jistý, NEPOTVRZUJ.",
	"page.confirm.btn":          "Potvrzuji úmrtí nebo trvalou neschopnost",
	"page.confirm.done.title":   "Potvrzeno",
	"page.confirm.done":         "✓ Potvrzeno. Obálka se odešle hlavnímu dědici po uplynutí ochranné lhůty, pokud vlastník mezitím nepotvrdí, že žije.",
	"page.confirm.partial":      "✓ Potvrzeno. Zatím %d z %d potřebných potvrzení; odpočet se spustí, až potvrdí i ostatní.",
	"page.confirm.failed.title": "Nelze potvrdit",

	"time.days":    "%d dní",
	"time.hours":   "%d hodin",
	"time.minutes": "%d minut",
	"owner":        "vlastník",
}
