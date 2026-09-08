package i18n

// skMessages is the Slovak translation. The text is the original wording the
// tool shipped with before it was translated, moved here unchanged.
var skMessages = map[string]string{
	"nav.home":   "domov",
	"nav.back":   "späť",
	"nav.again":  "skús znova",
	"lang.label": "Jazyk",
	"app.title":  "inh: offline nástroj",

	"index.h1":        "🔐 inh: offline Setup / Recovery",
	"index.setup.h":   "Nový backup",
	"index.setup.p":   "Vygeneruj key-file a rozdeľ ho na časti.",
	"index.recover.h": "Obnova",
	"index.recover.p": "Zlož časti späť na key-file na otvorenie KeePass DB.",
	"index.runbook.h": "Runbook",
	"index.runbook.p": "Vytvor personalizovaný návod „kto-čo-kde“ pre rodinu (vytlač / ulož ako PDF).",

	"error.title": "Chyba",

	"setup.title":     "Nový backup",
	"setup.intro":     "Vygeneruje sa náhodný 160-bit key-file a rozdelí sa na časti. Štandard pre tento systém je <strong>2-z-3</strong>.",
	"setup.threshold": "Prah: koľko častí treba na obnovu",
	"setup.count":     "Počet častí",
	"setup.submit":    "Vygenerovať",

	"setupres.title":        "Backup vygenerovaný (%s-z-%s)",
	"setupres.selftest.ok":  "✓ Self-test: %s časti správne zložili kľúč späť.",
	"setupres.selftest.bad": "✗ Self-test ZLYHAL. Nepoužívaj tento výstup, zopakuj generovanie.",
	"setupres.shares.h":     "Časti: každú na vlastné médium",
	"setupres.shares.p":     "Každá časť je %s očíslovaných slov; prenes ich na médium presne v tomto poradí. Na obnovu stačia ľubovoľné %s.",
	"setupres.prefix.p":     "Na kov stačí <strong>prvé 4 písmená</strong> (zvýraznené): v SLIP-39 zozname je nimi slovo určené jednoznačne, preto majú kovové médiá väčšinou len 4 pozície na slovo. Obnovovací nástroj si zvyšok doplní sám.",
	"setupres.keyfile.h":    "Key-file",
	"setupres.keyfile.p":    "Týmto súborom zamkneš KeePass DB. Po vytvorení DB ho zmaž, vždy ho znova zložíš z %s častí.",
	"setupres.download":     "⬇ Stiahnuť key-file",
	"setupres.hex":          "hex",
	"setupres.hex.warn":     "⚠️ Toto je obsah key-filu (tých istých 20 bajtov), len na <strong>okamžité vytvorenie DB</strong> a prípadné overenie. <strong>NEZAPISUJ ho do runbooku ani nikam inam</strong>: kľúč má žiť len ako %s časti; jeho uložením kdekoľvek by si obišiel ochranu %s-z-%s.",
	"setupres.next.h":       "Ďalej",
	"setupres.next.1":       "Prenes všetky časti na kovové médium a rozdaj ich podľa plánu.",
	"setupres.next.2":       "V KeePassXC vytvor novú databázu, ako ochranu zvoľ <em>Key file</em> = stiahnutý súbor (<strong>do poľa Heslo nič nezadávaj</strong>).",
	"setupres.next.3":       "Do DB ulož seed a ostatné prístupy. <strong>passphrase k peňaženke do DB NEDÁVAJ</strong>, tá ide do posmrtnej obálky.",
	"setupres.next.4":       "Rozkopíruj zašifrovanú <code>.kdbx</code> (cloud, banka, u všetkých). Key-file z disku zmaž.",

	"recover.title":         "Obnova key-filu",
	"recover.intro":         "Prepíš slová z kovových častí, <strong>každú časť do svojho bloku</strong>. Stačia <strong>prvé 4 písmená</strong> tak, ako sú vyryté; celé slovo sa doplní samo a kurzor skočí na ďalšie pole. Treba aspoň toľko častí, koľko bol prah (štandardne 2).",
	"recover.partcount":     "Počet častí",
	"recover.wordcount":     "Slov v časti",
	"recover.part":          "Časť %s",
	"recover.submit":        "Zložiť key-file",
	"recover.paste.sum":     "Alebo vlož časti naraz ako text",
	"recover.paste.p":       "Každá časť na svoj riadok. Použi, ak máš slová v súbore; inak vypĺňaj polia vyššie.",
	"recover.paste.ph":      "časť 1: slovo slovo slovo …\nčasť 2: slovo slovo slovo …",
	"recover.confirm.words": "Zmena dĺžky vymaže zadané slová. Pokračovať?",
	"recover.confirm.parts": "Odobratím sa vymažú slová zadané v tých častiach. Pokračovať?",

	"recoverres.title": "Kľúč obnovený",
	"recoverres.ok":    "✓ Z %s častí zložený key-file.",
	"recoverres.step":  "Pozor: toto je <strong>1. krok z 3</strong>. Samotný key-file ešte nie je prístup k Bitcoinu: treba ešte databázu <code>.kdbx</code> a passphrase z obálky.",
	"recoverres.chain": `①  časti
       │   nástroj ich poskladá dokopy
       ▼
   key-file: malý súbor v počítači          ← TOTO práve vzniklo
       │   ním sa otvorí databáza hesiel (KeePassXC)
       ▼
②  v databáze: SEED peňaženky + ostatné prístupy
       │   k tomu ešte passphrase z obálky
       ▼
③  obnova peňaženky  →  Bitcoin ₿`,
	"recoverres.next.h": "Ďalej",
	"recoverres.next.1": "Otvor svoju <code>.kdbx</code> v KeePassXC; ako ochranu zvoľ <em>Key file</em> = tento súbor (<strong>do poľa Heslo nič nezadávaj</strong>).",
	"recoverres.next.2": "Dostaneš sa k <strong>seedu</strong> a ďalším informáciám a prístupom; <strong>passphrase k peňaženke</strong> vezmi z obálky (banka / e-mail).",
	"recoverres.next.3": "Na hardvérovej peňaženke obnov zo seedu, potom odomkni passphrase.",

	"rbform.title":        "Runbook: návod pre rodinu",
	"rbform.intro":        "Vyplň údaje; vygeneruje sa <strong>tlačiteľný návod „kto-čo-kde“</strong>. Dokument <strong>neobsahuje žiadne tajomstvá</strong>, len mapu a postup. Aj tak ho drž u dôveryhodných osôb (prezrádza, kde čo hľadať).",
	"rbform.lang":         "Jazyk runbooku",
	"rbform.basics":       "Základ",
	"rbform.author":       "Tvoje meno",
	"rbform.date":         "Dátum",
	"rbform.heir":         "Meno hlavného dediča (komu je list určený)",
	"rbform.scheme":       "Schéma",
	"rbform.threshold":    "Prah (koľko častí treba na obnovu)",
	"rbform.count":        "Počet častí",
	"rbform.holders":      "Časti a kto ich drží",
	"rbform.holders.p":    "Počet riadkov sa riadi podľa <strong>„Počet častí“</strong> vyššie. Pri každej časti uveď, <strong>kto ju drží</strong> a kontakt, a či je <strong>technicky zdatná</strong> (volá sa jej o pomoc, dostane obálku) alebo netechnická.",
	"rbform.ph.name":      "kto drží túto časť: meno",
	"rbform.ph.contact":   "kontakt (tel. / e-mail)",
	"rbform.nontech":      "netechnická",
	"rbform.tech":         "technicky zdatná",
	"rbform.envelope":     "Obálka a kópie",
	"rbform.bank":         "Obálka (passphrase), banka",
	"rbform.ph.bank":      "názov banky, pobočka, číslo schránky",
	"rbform.toolwhere":    "Kde je uložený nástroj <em>inh-offline</em>",
	"rbform.ph.toolwhere": "napr. USB kľúč pri každom kovovom médiu, v banke a v cloude vedľa .kdbx",
	"rbform.kdbx":         "Kópie zašifrovanej .kdbx, kde",
	"rbform.ph.kdbx":      "napr. cloud, doma, bankový trezor, u osôb",
	"rbform.extras":       "Doplnky",
	"rbform.wallet":       "Peňaženka, poznámky",
	"rbform.message":      "Odkaz pre rodinu (voľný text)",
	"rbform.submit":       "Vygenerovať runbook",
	"rbform.rownum":       "Časť %s:",

	"rb.aside": " (%s)",
	"names.or": " alebo ",

	"holder.1": "Prvá časť",
	"holder.2": "Druhá časť",
	"holder.3": "Tretia časť",
	"holder.n": "Časť #%d",

	"word.empty":   "riadok neobsahuje žiadne slová",
	"word.at":      "%d. slovo:",
	"word.at.part": "%d. časť, %d. slovo:",
	"word.short":   "%q je príliš krátke, na kove sú vždy aspoň %d písmená",
	"word.unknown": "%q nie je slovo zo SLIP-39 zoznamu",
	"word.typo":    "%q nie je slovo zo SLIP-39 zoznamu (podľa prvých %d písmen by to malo byť %q)",

	"err.rand":     "Zlyhal generátor náhody: %s",
	"err.params":   "Neplatné parametre: %s",
	"err.noshares": "Nezadal si žiadnu časť.",
	"err.words":    "Nerozumiem zadaným slovám: %s. Slová píš tak, ako sú na kove (stačia prvé 4 písmená), každú časť na svoj riadok.",
	"err.combine":  "Obnova zlyhala: %s. Skontroluj, či sú slová a počet častí správne.",
	"err.form":     "Neplatný formulár: %s",
}
