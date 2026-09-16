package i18n

// Slovak runbook text, moved here from the template unchanged.
func init() {
	for k, v := range map[string]string{
		"rb.title":     "Runbook: inštrukcie pre rodinu",
		"rb.h1":        "Bitcoin: inštrukcie pre prípad mojej smrti",
		"rb.author":    "Autor: <strong>%s</strong>. ",
		"rb.date":      "Dátum: %s.",
		"rb.nosecrets": "Tento dokument <strong>neobsahuje žiadne tajomstvá</strong> (žiadny seed, heslo, passphrase ani časti), je to len <strong>mapa a postup</strong>. Prezrádza však, kde čo hľadať, preto ho nezverejňuj a drž len u dôveryhodných osôb.",

		"rb.intro.h":         "Úvod",
		"rb.intro.to":        "Adresát: <strong>%s</strong>",
		"rb.intro.p":         "Najprv zavolaj technicky zdatnú osobu (nižšie), ktorá ťa celým postupom prevedie. Neponáhľaj sa, rob to po krokoch a nikomu cudziemu nedávaj nič.",
		"rb.intro.whatis":    "<strong>Čo je „časť“:</strong> zoznam <strong>23 slov</strong> uložený na <strong>kovovom médiu</strong>, aby prežil aj oheň a vodu. Tvar sa u jednotlivých držiteľov môže líšiť: býva to plochá platnička alebo doštička, kovový valček či kapsula, kazeta so zasúvacími písmenkami, kartička veľkosti platobnej karty, alebo aj niekoľko zoskrutkovaných plieškov. Hľadaj kovový predmet so slovami alebo písmenami. Jedna časť samotná je nanič, treba ich <strong>%s z %s</strong>, aby sa z nich zložil „key-file“ (kľúč k databáze). Jednotlivé časti držia rôzne dôveryhodné osoby (nižšie).",
		"rb.intro.bankshare": "Jedna z častí je v <strong>bankovom trezore</strong>, spolu s obálkou.",

		"rb.auto.h": "Ako to funguje (automatický systém)",
		"rb.auto.p": "Kým žijem, chodia mi pravidelné kontrolné e-maily. Ak sa dlhšie neozvem a niektorá z dôveryhodných osôb to potvrdí, systém po ochrannej lehote (niekoľko dní) <strong>automaticky pošle obálku e-mailom hlavnému dedičovi%s</strong>: PDF s tým istým textom, ktorý je uložený v bankovom trezore. Text sám o sebe nič neprezradí; ako z neho prečítať passphrase, je napísané v databáze hesiel. Keby to automatické posielanie zlyhalo, vytlačený text je v bankovom trezore (bod 3).",

		"rb.call.h":      "1) Koho zavolať",
		"rb.call.tech":   "<strong>Technicky zdatné osoby.</strong> Volaj ich, prevedú ťa celým postupom:",
		"rb.call.notech": "Nie je uvedená žiadna technicky zdatná osoba. Doplň aspoň jednu, inak nebude mať kto pomôcť.",
		"rb.call.others": "<strong>Ostatní držitelia častí:</strong>",

		"rb.need.h":     "2) Čo treba získať (dve veci)",
		"rb.need.p":     "Na prístup k Bitcoinu treba <strong>oboje</strong>:",
		"rb.need.parts": "<strong>%s z %s častí</strong> (každá je 23 slov), z nich sa zloží „key-file“.",
		"rb.need.env":   "<strong>Obálku</strong>: vytlačený text s ukrytou passphrase, druhé tajomstvo, bez ktorého sa k Bitcoinu nedá dostať. Ako z neho passphrase prečítať, je napísané v databáze.",
		"rb.need.note":  "Ani jedno samo o sebe nestačí, preto je to bezpečné.",

		"rb.map.h":         "3) Mapa: kde čo je",
		"rb.map.item":      "Položka",
		"rb.map.where":     "Kde / u koho",
		"rb.map.part":      "Časť %s",
		"rb.map.held":      "drží: %s",
		"rb.map.inbank":    "bankový trezor, spolu s obálkou",
		"rb.map.heir":      "hlavný dedič",
		"rb.map.env":       "Obálka (text), banka",
		"rb.map.kdbx":      "Databáza hesiel (.kdbx)",
		"rb.map.kdbx.bank": ", a tiež v bankovom trezore",

		"rb.steps.h":     "4) Postup krok za krokom",
		"rb.steps.intro": "Ako to do seba zapadá (treba <strong>tri veci</strong>, nie jednu):",
		"rb.steps.chain": `①  %s z %s častí (slová z kovu)
       │   nástroj z nich poskladá kľúč
       ▼
   kľúč: malý súbor v počítači
       │   ním sa otvorí databáza hesiel (program KeePassXC)
       ▼
②  v databáze: SEED peňaženky + ostatné prístupy
       │   k tomu ešte heslo (passphrase) z obálky
       ▼
③  obnova peňaženky  →  Bitcoin ₿`,

		"rb.step.call":       "Zavolaj technicky zdatnú osobu (bod 1). Poradí ti so všetkým, čomu nebudeš rozumieť.",
		"rb.step.parts":      "Zožeň <strong>aspoň %s časti</strong> (bod 3). Slová si od držiteľov odpíš na papier alebo si ich nechaj nadiktovať. <strong>Nefoť ich telefónom</strong>, nie je to bezpečné.",
		"rb.step.parts.bank": "Jedna z nich je v bankovom trezore spolu s obálkou, takže obe vyzdvihneš naraz.",
		"rb.step.env":        "Zožeň <strong>obálku</strong>: buď vytlačený text zapečatený v bankovom trezore, alebo PDF, ktoré systém pošle e-mailom hlavnému dedičovi%s. V oboch prípadoch je to ten istý text.",
		"rb.step.pc":         "Priprav <strong>počítač odpojený od internetu</strong> (vytiahni kábel, vypni wifi). Je to dôležité: slová z častí sa naň budú písať a nesmú sa odtiaľ dostať von.<br>Nástroj sa volá <em>coldwill</em>. Na počítači, ktorý internet <em>má</em>, ho stiahni z <strong>%s</strong> a prenes ho na USB kľúči. Súbory sú na tej stránke dole, pod nadpisom <em>Assets</em>.",
		"rb.step.pc.where":   "Ak tá adresa už nefunguje, tie isté súbory sú uložené aj tu: <strong>%s</strong>.",
		"rb.step.pc.pick":    "Súborov je viac, jeden pre každý typ počítača. <strong>Spusti ten, ktorý sedí:</strong>",
		"rb.step.pc.os":      "Počítač",
		"rb.step.pc.file":    "Súbor",
		"rb.step.pc.win":     "Windows",
		"rb.step.pc.mac1":    "Mac (novší, od roku 2020)",
		"rb.step.pc.mac2":    "Mac (starší, Intel)",
		"rb.step.pc.lin":     "Linux",
		"rb.step.pc.ask":     "Ak si nie si istý, ktorý to je, spýtaj sa technicky zdatnej osoby. Zlý súbor nič nepokazí, len sa nespustí.",

		"rb.step.warn":     "<strong>Počítač bude asi protestovať.</strong> Nie je to chyba ani vírus: je to program, za ktorý nikto nezaplatil registráciu u Microsoftu ani Apple. Varovanie pokojne obíď:",
		"rb.step.warn.win": "<strong>Windows:</strong> objaví sa modré okno v štýle „Systém Windows ochránil váš počítač“. Klikni na <em>Ďalšie informácie</em> a potom na <em>Spustiť aj tak</em>.",
		"rb.step.warn.mac": "<strong>Mac:</strong> napíše, že súbor je od neovereného vývojára. Zavri to, klikni na súbor <strong>pravým tlačidlom</strong> (alebo dvoma prstami) a vyber <em>Otvoriť</em>; v ďalšom okne znova <em>Otvoriť</em>. Ak to nejde, otvor <em>Nastavenia → Súkromie a bezpečnosť</em>, kde bude tlačidlo <em>Otvoriť napriek tomu</em>.",
		"rb.step.warn.lin": "<strong>Linux:</strong> ak sa súbor nedá spustiť, technicky zdatná osoba mu dá právo príkazom <code>chmod +x coldwill-linux-amd64</code>.",
		"rb.step.warn.url": "Ak sa neotvorí prehliadač sám, otvor ho a napíš doň adresu, ktorú nástroj vypíše (býva <code>http://127.0.0.1:8777</code>).",

		"rb.step.run":        "Spusti <em>coldwill</em> → <em>Obnova</em> → vlož %s časti → dostaneš <strong>súbor s kľúčom</strong> (key-file). Ulož si ho na ten počítač.",
		"rb.step.kdbx":       "Otvor databázu <code>.kdbx</code> programom <strong>KeePassXC</strong> (<a href=\"https://keepassxc.org/download/\">https://keepassxc.org/download</a>).<br><strong>Najlepšie je</strong> nainštalovať KeePassXC na ten istý odpojený počítač a databázu tam skopírovať. Seed potom nikdy neopustí stroj bez internetu.<br>Ak to nejde, prenes súbor s kľúčom na USB kľúči do počítača, kde KeePassXC máš, a ten počítač na ten čas <strong>odpoj od internetu</strong>.<br>V KeePassXC zvoľ ako ochranu <em>Key file</em> a vyber ten súbor s kľúčom. <strong>Do poľa Heslo nezadávaj nič.</strong> Dostaneš sa k <strong>seedu</strong> a ďalším informáciám a prístupom.",
		"rb.step.kdbx.where": "Databázu má <strong>%s</strong>.",
		"rb.step.kdbx.bank":  "Je aj v <strong>bankovom trezore</strong>.",
		"rb.step.pass":       "V databáze nájdi <strong>mapu k obálke</strong> a podľa nej prečítaj z textu <strong>passphrase</strong>: znak po znaku, v poradí mapy, presne tak, ako je vytlačený, vrátane veľkých písmen a medzier. Text nijako neupravuj, jeho zalomenie riadkov je súčasťou mapy.",
		"rb.step.wallet":     "Na hardvérovej peňaženke obnov peňaženku <strong>zo seedu</strong> (tie slová z databázy). Heslo (passphrase) sa pri obnove <strong>nezadáva</strong>, peňaženka si oň povie až potom, pri odomykaní. Bez neho uvidíš prázdnu peňaženku, s ním Bitcoin. Prázdna peňaženka teda neznamená, že peniaze sú preč, ale že chýba passphrase z obálky.",
		"rb.step.move":       "<em>(voliteľné, ale dôrazne odporúčané)</em> Presuň prostriedky do novej peňaženky, ktorú ovládaš ty.",

		"rb.manual.h":      "5) Núdzový (manuálny) postup, ak nástroj nefunguje",
		"rb.manual.slip39": "Časti sú v štandarde <strong>SLIP-39</strong>, zložia sa hocijakým SLIP-39 nástrojom (napr. referenčná knižnica <code>shamir-mnemonic</code>). Výsledkom je <strong>master secret (hex)</strong>, ktorý <strong>je obsahom key-filu</strong>.",
		"rb.manual.abbrev": "Na kove sú slová často len ako <strong>4-písmenové skratky</strong>, to nie je chyba, v SLIP-39 zozname je slovo prvými štyrmi písmenami určené jednoznačne. Nástroj <em>coldwill</em> ich doplní sám; ak použiješ iný nástroj, ktorý skratky neprijme, dohľadaj celé slová v oficiálnom SLIP-39 zozname (1024 slov, je v každom SLIP-39 nástroji).",
		"rb.manual.bytes":  "Tu sa dá ľahko pomýliť: do súboru patria <strong>surové bajty</strong>, nie ten hex zapísaný ako text, inak KeePassXC databázu neotvorí. Na Linuxe/macOS:",
		"rb.manual.check":  "Kontrola: súbor má mať polovicu dĺžky hexu (40 hex znakov → 20 bajtov). Potom ho v KeePassXC použi ako <em>Key file</em> a heslo nechaj prázdne. Databáza <code>.kdbx</code> je otvorený formát (KeePassXC, KeePass, KeePassDX, Strongbox, …).",
		"rb.manual.ask":    "Ak tomuto nerozumieš, daj to prečítať technicky zdatnej osobe%s.",

		"rb.safety.h":      "6) Bezpečnosť: na čo si dať pozor",
		"rb.safety.seed":   "<strong>Seed zadávaj jedine do hardvérovej peňaženky.</strong> Nikdy ho nepíš do počítača, do mobilu, do e-mailu ani na žiadnu webovú stránku, ani keď o to niekto žiada.",
		"rb.safety.words":  "<strong>Slová z častí</strong> zadávaj jedine do nástroja <em>coldwill</em> na počítači odpojenom od internetu (pri núdzovom postupe do iného SLIP-39 nástroja, tiež bez internetu).",
		"rb.safety.nobody": "Nikomu cudziemu (ani „technickej podpore“) nedávaj slová, seed, mapu ani passphrase.",
		"rb.safety.wipe":   "<strong>Po skončení</strong> zmaž súbor s kľúčom z počítača aj z USB kľúča. Kto ho má, otvorí si databázu. Papieriky so slovami spáľ alebo vráť držiteľom.",
		"rb.safety.move":   "<em>(voliteľné, ale dôrazne odporúčané)</em> Po obnove presuň prostriedky do novej peňaženky.",
		"rb.safety.slow":   "Neponáhľaj sa; ak niečo nesedí, zastav a poraď sa s technicky zdatnou osobou%s.",

		"rb.notes.wallet": "Peňaženka: poznámky",
		"rb.notes.family": "Odkaz",

		"rb.print":   "🖨 Vytlačiť / Uložiť ako PDF",
		"rb.edit":    "✏️ Upraviť",
		"rb.newform": "← nový formulár",
		"rb.margins": "Okraje sú súčasťou dokumentu, takže na tom, čo ponúka tlačový dialóg, nezáleží.",
	} {
		skMessages[k] = v
	}
}
