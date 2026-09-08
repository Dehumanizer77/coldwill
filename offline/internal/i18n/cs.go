package i18n

// csMessages is the Czech translation.
//
// Czech and Slovak are close enough that a Slovak reader manages either way,
// which is exactly why this was translated rather than left to fall back: a
// document someone follows under stress should be in their own language, not in
// one they can approximately parse.
var csMessages = map[string]string{
	"nav.home":   "domů",
	"nav.back":   "zpět",
	"nav.again":  "zkusit znovu",
	"lang.label": "Jazyk",
	"app.title":  "inh: offline nástroj",

	"index.h1":        "🔐 inh: offline Setup / Recovery",
	"index.setup.h":   "Nová záloha",
	"index.setup.p":   "Vygeneruj key-file a rozděl ho na části.",
	"index.recover.h": "Obnova",
	"index.recover.p": "Slož části zpět na key-file a otevři jím databázi KeePass.",
	"index.runbook.h": "Runbook",
	"index.runbook.p": "Vytvoř personalizovaný návod „kdo-co-kde“ pro rodinu (vytiskni nebo ulož jako PDF).",

	"error.title": "Chyba",

	"setup.title":     "Nová záloha",
	"setup.intro":     "Vygeneruje se náhodný 160bitový key-file a rozdělí se na části. Obvyklá volba je <strong>2 ze 3</strong>.",
	"setup.threshold": "Práh: kolik částí je potřeba k obnově",
	"setup.count":     "Počet částí",
	"setup.submit":    "Vygenerovat",

	"setupres.title":        "Záloha vygenerována (%s ze %s)",
	"setupres.selftest.ok":  "✓ Autotest: %s části správně složily klíč zpět.",
	"setupres.selftest.bad": "✗ Autotest SELHAL. Tento výstup nepoužívej, vygeneruj znovu.",
	"setupres.shares.h":     "Části: každou na vlastní médium",
	"setupres.shares.p":     "Každá část je %s očíslovaných slov; přenes je na médium přesně v tomto pořadí. K obnově stačí libovolné %s.",
	"setupres.prefix.p":     "Na kov stačí <strong>první 4 písmena</strong> (zvýrazněná): v seznamu SLIP-39 je jimi slovo určeno jednoznačně, proto mívají kovová média jen 4 pozice na slovo. Obnovovací nástroj si zbytek doplní sám.",
	"setupres.keyfile.h":    "Key-file",
	"setupres.keyfile.p":    "Tímto souborem zamkneš databázi KeePass. Po jejím vytvoření ho smaž, vždy ho znovu složíš z %s částí.",
	"setupres.download":     "⬇ Stáhnout key-file",
	"setupres.hex":          "hex",
	"setupres.hex.warn":     "⚠️ Toto je obsah key-filu (těch samých 20 bajtů), jen pro <strong>okamžité vytvoření databáze</strong> a případné ověření. <strong>NEZAPISUJ ho do runbooku ani nikam jinam</strong>: klíč má žít pouze jako %s části; jeho uložením kamkoli bys obešel ochranu %s ze %s.",
	"setupres.next.h":       "Dále",
	"setupres.next.1":       "Přenes všechny části na kovové médium a rozdej je podle plánu.",
	"setupres.next.2":       "V <a href=\"https://keepassxc.org/download/\">KeePassXC</a> vytvoř novou databázi a jako ochranu zvol <em>Key file</em> = stažený soubor (<strong>do pole Heslo nic nezadávej</strong>).",
	"setupres.next.3":       "Do databáze ulož seed a ostatní přístupy. <strong>Passphrase k peněžence do databáze NEDÁVEJ</strong>, ta patří do posmrtné obálky.",
	"setupres.next.4":       "Rozkopíruj zašifrovanou <code>.kdbx</code> (cloud, banka, u všech držitelů). Key-file z disku smaž.",

	"recover.title":         "Obnova key-filu",
	"recover.intro":         "Přepiš slova z kovových částí, <strong>každou část do svého bloku</strong>. Stačí <strong>první 4 písmena</strong> tak, jak jsou vyražená; celé slovo se doplní samo a kurzor skočí na další pole. Potřebuješ alespoň tolik částí, kolik byl práh (obvykle 2).",
	"recover.partcount":     "Počet částí",
	"recover.wordcount":     "Slov v části",
	"recover.part":          "Část %s",
	"recover.submit":        "Složit key-file",
	"recover.paste.sum":     "Nebo vlož části najednou jako text",
	"recover.paste.p":       "Každá část na svůj řádek. Použij, pokud máš slova v souboru; jinak vyplňuj pole výše.",
	"recover.paste.ph":      "část 1: slovo slovo slovo …\nčást 2: slovo slovo slovo …",
	"recover.confirm.words": "Změna délky vymaže zadaná slova. Pokračovat?",
	"recover.confirm.parts": "Odebráním se vymažou slova zadaná v těchto částech. Pokračovat?",

	"recoverres.title": "Klíč obnoven",
	"recoverres.ok":    "✓ Z %s částí složen key-file.",
	"recoverres.step":  "Pozor: toto je <strong>1. krok ze 3</strong>. Samotný key-file ještě není přístup k Bitcoinu: je potřeba ještě databáze <code>.kdbx</code> a passphrase z obálky.",
	"recoverres.chain": `①  části
       │   nástroj je poskládá dohromady
       ▼
   key-file: malý soubor v počítači        ← TOTO právě vzniklo
       │   jím se otevře databáze hesel (KeePassXC)
       ▼
②  v databázi: SEED peněženky + ostatní přístupy
       │   k tomu ještě passphrase z obálky
       ▼
③  obnova peněženky  →  Bitcoin ₿`,
	"recoverres.next.h": "Dále",
	"recoverres.next.1": "Otevři svou <code>.kdbx</code> v <a href=\"https://keepassxc.org/download/\">KeePassXC</a>; jako ochranu zvol <em>Key file</em> = tento soubor (<strong>do pole Heslo nic nezadávej</strong>).",
	"recoverres.next.2": "Dostaneš se k <strong>seedu</strong> a dalším informacím a přístupům; <strong>passphrase k peněžence</strong> vezmi z obálky (banka nebo e-mail).",
	"recoverres.next.3": "Na hardwarové peněžence obnov ze seedu, potom odemkni pomocí passphrase.",

	"rbform.title":        "Runbook: návod pro rodinu",
	"rbform.intro":        "Vyplň údaje; vygeneruje se <strong>tisknutelný návod „kdo-co-kde“</strong>. Dokument <strong>neobsahuje žádná tajemství</strong>, jen mapu a postup. Přesto ho drž u důvěryhodných osob (prozrazuje, kde co hledat).",
	"rbform.lang":         "Jazyk runbooku",
	"rbform.basics":       "Základ",
	"rbform.author":       "Tvoje jméno",
	"rbform.date":         "Datum",
	"rbform.heir":         "Jméno hlavního dědice (komu je dopis určen)",
	"rbform.scheme":       "Schéma",
	"rbform.threshold":    "Práh (kolik částí je potřeba k obnově)",
	"rbform.count":        "Počet částí",
	"rbform.holders":      "Části a kdo je drží",
	"rbform.holders.p":    "Počet řádků se řídí podle <strong>„Počet částí“</strong> výše. U každé části uveď, <strong>kdo ji drží</strong> a kontakt, a zda je <strong>technicky zdatná</strong> (volá se jí o pomoc, dostane obálku) nebo netechnická.",
	"rbform.ph.name":      "kdo drží tuto část: jméno",
	"rbform.ph.contact":   "kontakt (tel. nebo e-mail)",
	"rbform.nontech":      "netechnická",
	"rbform.tech":         "technicky zdatná",
	"rbform.envelope":     "Obálka a kopie",
	"rbform.bank":         "Obálka (passphrase), banka",
	"rbform.ph.bank":      "název banky, pobočka, číslo schránky",
	"rbform.toolwhere":    "Kde je uložen nástroj <em>inh-offline</em>",
	"rbform.ph.toolwhere": "např. USB klíč u každého kovového média, v bance a v cloudu vedle .kdbx",
	"rbform.kdbx":         "Kopie zašifrované .kdbx, kde",
	"rbform.ph.kdbx":      "např. cloud, doma, bankovní trezor, u držitelů",
	"rbform.extras":       "Doplňky",
	"rbform.wallet":       "Peněženka, poznámky",
	"rbform.message":      "Vzkaz pro rodinu (volný text)",
	"rbform.submit":       "Vygenerovat runbook",
	"rbform.rownum":       "Část %s:",

	"rb.aside": " (%s)",
	"names.or": " nebo ",

	"holder.1": "První část",
	"holder.2": "Druhá část",
	"holder.3": "Třetí část",
	"holder.n": "Část #%d",

	"word.empty":   "řádek neobsahuje žádná slova",
	"word.at":      "%d. slovo:",
	"word.at.part": "%d. část, %d. slovo:",
	"word.short":   "%q je příliš krátké, na kovu jsou vždy alespoň %d písmena",
	"word.unknown": "%q není slovo ze seznamu SLIP-39",
	"word.typo":    "%q není slovo ze seznamu SLIP-39 (podle prvních %d písmen by to mělo být %q)",

	"err.rand":     "Selhal generátor náhody: %s",
	"err.params":   "Neplatné parametry: %s",
	"err.noshares": "Nezadal jsi žádnou část.",
	"err.words":    "Nerozumím zadaným slovům: %s. Slova piš tak, jak jsou na kovu (stačí první 4 písmena), každou část na svůj řádek.",
	"err.combine":  "Obnova selhala: %s. Zkontroluj, zda jsou slova a počet částí správně.",
	"err.form":     "Neplatný formulář: %s",
}
