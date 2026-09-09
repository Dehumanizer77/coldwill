package i18n

// Czech runbook text. This is the document a family member reads while under
// stress, so it is written plainly rather than translated word for word.
func init() {
	for k, v := range map[string]string{
		"rb.title":     "Runbook: instrukce pro rodinu",
		"rb.h1":        "Bitcoin: co dělat, když zemřu",
		"rb.author":    "Autor: <strong>%s</strong>. ",
		"rb.date":      "Datum: %s.",
		"rb.nosecrets": "Tento dokument <strong>neobsahuje žádná tajemství</strong> (žádný seed, heslo, passphrase ani části), je to jen <strong>mapa a postup</strong>. Prozrazuje ale, kde co hledat, proto ho nezveřejňuj a drž jen u důvěryhodných osob.",

		"rb.intro.h":      "Úvod",
		"rb.intro.to":     "Adresát: <strong>%s</strong>",
		"rb.intro.p":      "Nejdřív zavolej technicky zdatnou osobu (níže), která tě celým postupem provede. Nespěchej, dělej to po krocích a nikomu cizímu nic nedávej.",
		"rb.intro.whatis": "<strong>Co je „část“:</strong> seznam <strong>23 slov</strong> uložený na <strong>kovovém médiu</strong>, aby přežil i oheň a vodu. Tvar se u jednotlivých držitelů může lišit: bývá to plochá destička, kovový váleček či kapsle, kazeta se zasouvacími písmenky, kartička velikosti platební karty, nebo i několik sešroubovaných plíšků. Hledej kovový předmět se slovy nebo písmeny. Jedna část sama o sobě je k ničemu, potřebuješ jich <strong>%s ze %s</strong>, aby se z nich složil „key-file“ (klíč k databázi). Jednotlivé části drží různé důvěryhodné osoby (níže).",

		"rb.auto.h": "Jak funguje automatický systém",
		"rb.auto.p": "Dokud žiju, chodí mi pravidelné kontrolní e-maily. Pokud se delší dobu neozvu a některá z důvěryhodných osob to potvrdí, systém po ochranné lhůtě (několik dní) <strong>automaticky pošle obálku s passphrase e-mailem technicky zdatné osobě%s</strong>. Kdyby toto automatické posílání selhalo, stejné heslo je napsané na papíře v bankovním trezoru (bod 2).",

		"rb.call.h":      "1) Komu zavolat",
		"rb.call.tech":   "<strong>Technicky zdatné osoby.</strong> Volej je, provedou tě celým postupem:",
		"rb.call.notech": "Není uvedena žádná technicky zdatná osoba. Doplň alespoň jednu, jinak nebude mít kdo pomoct.",
		"rb.call.others": "<strong>Ostatní držitelé částí:</strong>",

		"rb.need.h":     "2) Co je potřeba získat (dvě věci)",
		"rb.need.p":     "K přístupu k Bitcoinu je potřeba <strong>obojí</strong>:",
		"rb.need.parts": "<strong>%s ze %s částí</strong> (každá je 23 slov), z nich se složí „key-file“.",
		"rb.need.env":   "<strong>Obálku s passphrase</strong>: druhé tajemství, bez kterého se k Bitcoinu nelze dostat.",
		"rb.need.note":  "Ani jedno samo o sobě nestačí, proto je to bezpečné.",

		"rb.map.h":     "3) Mapa: kde co je",
		"rb.map.item":  "Položka",
		"rb.map.where": "Kde / u koho",
		"rb.map.part":  "Část %s",
		"rb.map.held":  "drží: %s",
		"rb.map.env":   "Obálka (passphrase), banka",
		"rb.map.kdbx":  "Zašifrovaná databáze (.kdbx), kopie",

		"rb.steps.h":     "4) Postup krok za krokem",
		"rb.steps.intro": "Jak to do sebe zapadá (jsou potřeba <strong>tři věci</strong>, ne jedna):",
		"rb.steps.chain": `①  %s ze %s částí (slova z kovu)
       │   nástroj z nich poskládá klíč
       ▼
   klíč: malý soubor v počítači
       │   jím se otevře databáze hesel (program KeePassXC)
       ▼
②  v databázi: SEED peněženky + ostatní přístupy
       │   k tomu ještě heslo (passphrase) z obálky
       ▼
③  obnova peněženky  →  Bitcoin ₿`,

		"rb.step.call":    "Zavolej technicky zdatnou osobu (bod 1). Poradí ti se vším, čemu nebudeš rozumět.",
		"rb.step.parts":   "Sežeň <strong>alespoň %s části</strong> (bod 3). Slova si od držitelů opiš na papír nebo si je nech nadiktovat. <strong>Nefoť je telefonem</strong>, není to bezpečné.",
		"rb.step.env":     "Sežeň <strong>obálku s heslem (passphrase)</strong>: buď zapečetěnou z bankovního trezoru, nebo od technicky zdatné osoby%s, které ji systém pošle e-mailem.",
		"rb.step.pc":      "Připrav <strong>počítač odpojený od internetu</strong> (vytáhni kabel, vypni wifi). Je to důležité: slova z částí se do něj budou psát a nesmí se odtud dostat ven.<br>Nástroj se jmenuje <em>inh-offline</em>. Na počítači, který internet <em>má</em>, ho stáhni z <strong>%s</strong> a přenes ho na USB klíči. Soubory jsou na té stránce dole, pod nadpisem <em>Assets</em>.<br>Pokud ta adresa už nefunguje, tytéž soubory jsou na <strong>USB klíči u každého kovového média</strong>, takže ho má každý držitel části, a na jednom v <strong>bankovním trezoru</strong>%s.<br>Souborů je víc, jeden pro každý typ počítače. <strong>Spusť ten, který sedí:</strong>",
		"rb.step.pc.os":   "Počítač",
		"rb.step.pc.file": "Soubor",
		"rb.step.pc.win":  "Windows",
		"rb.step.pc.mac1": "Mac (novější, od roku 2020)",
		"rb.step.pc.mac2": "Mac (starší, Intel)",
		"rb.step.pc.lin":  "Linux",
		"rb.step.pc.ask":  "Pokud si nejsi jistý, který to je, zeptej se technicky zdatné osoby. Špatný soubor nic nepokazí, jen se nespustí.",

		"rb.step.warn":     "<strong>Počítač bude nejspíš protestovat.</strong> Není to chyba ani virus: je to program, za který nikdo nezaplatil registraci u Microsoftu ani Apple. Varování klidně obejdi:",
		"rb.step.warn.win": "<strong>Windows:</strong> objeví se modré okno ve stylu „Systém Windows ochránil váš počítač“. Klikni na <em>Další informace</em> a potom na <em>Přesto spustit</em>.",
		"rb.step.warn.mac": "<strong>Mac:</strong> napíše, že soubor je od neověřeného vývojáře. Zavři to, klikni na soubor <strong>pravým tlačítkem</strong> (nebo dvěma prsty) a vyber <em>Otevřít</em>; v dalším okně znovu <em>Otevřít</em>. Pokud to nejde, otevři <em>Nastavení → Soukromí a zabezpečení</em>, kde bude tlačítko <em>Přesto otevřít</em>.",
		"rb.step.warn.lin": "<strong>Linux:</strong> pokud soubor nejde spustit, technicky zdatná osoba mu dá právo příkazem <code>chmod +x inh-offline-linux-amd64</code>.",
		"rb.step.warn.url": "Pokud se prohlížeč neotevře sám, otevři ho a napiš do něj adresu, kterou nástroj vypíše (bývá <code>http://127.0.0.1:8777</code>).",

		"rb.step.run":    "Spusť <em>inh-offline</em> → <em>Obnova</em> → vlož %s části → dostaneš <strong>soubor s klíčem</strong> (key-file). Ulož si ho do toho počítače.",
		"rb.step.kdbx":   "Otevři databázi <code>.kdbx</code> programem <strong>KeePassXC</strong> (<a href=\"https://keepassxc.org/download/\">https://keepassxc.org/download</a>).<br><strong>Nejlepší je</strong> nainstalovat KeePassXC do toho samého odpojeného počítače a databázi tam zkopírovat. Seed pak nikdy neopustí stroj bez internetu.<br>Pokud to nejde, přenes soubor s klíčem na USB klíči do počítače, kde KeePassXC máš, a ten počítač na tu dobu <strong>odpoj od internetu</strong>.<br>V KeePassXC zvol jako ochranu <em>Key file</em> a vyber ten soubor s klíčem. <strong>Do pole Heslo nic nezadávej.</strong> Dostaneš se k <strong>seedu</strong> a dalším informacím a přístupům.",
		"rb.step.wallet": "Na hardwarové peněžence obnov peněženku <strong>ze seedu</strong> (ta slova z databáze). Heslo (passphrase) se při obnově <strong>nezadává</strong>, peněženka si o ně řekne až potom, při odemykání. Bez něj uvidíš prázdnou peněženku, s ním Bitcoin. Prázdná peněženka tedy neznamená, že peníze jsou pryč, ale že chybí heslo z obálky.",
		"rb.step.move":   "<em>(volitelné, ale důrazně doporučené)</em> Přesuň prostředky do nové peněženky, kterou ovládáš ty.",

		"rb.manual.h":      "5) Nouzový (ruční) postup, pokud nástroj nefunguje",
		"rb.manual.slip39": "Části jsou ve standardu <strong>SLIP-39</strong>, složí je jakýkoli nástroj pro SLIP-39 (např. referenční knihovna <code>shamir-mnemonic</code>). Výsledkem je <strong>master secret (hex)</strong>, který <strong>je obsahem key-filu</strong>.",
		"rb.manual.abbrev": "Na kovu jsou slova často jen jako <strong>čtyřpísmenné zkratky</strong>, to není chyba: v seznamu SLIP-39 je slovo prvními čtyřmi písmeny určeno jednoznačně. Nástroj <em>inh-offline</em> je doplní sám; pokud použiješ jiný nástroj, který zkratky nepřijme, dohledej celá slova v oficiálním seznamu SLIP-39 (1024 slov, je v každém nástroji pro SLIP-39).",
		"rb.manual.bytes":  "Tady se dá snadno splést: do souboru patří <strong>surové bajty</strong>, ne ten hex zapsaný jako text, jinak KeePassXC databázi neotevře. Na Linuxu a macOS:",
		"rb.manual.check":  "Kontrola: soubor má mít poloviční délku hexu (40 hex znaků → 20 bajtů). Potom ho v KeePassXC použij jako <em>Key file</em> a heslo nech prázdné. Formát <code>.kdbx</code> je otevřený (KeePassXC, KeePass, KeePassDX, Strongbox a další).",
		"rb.manual.ask":    "Pokud tomuhle nerozumíš, dej to přečíst technicky zdatné osobě%s.",

		"rb.safety.h":      "6) Bezpečnost: na co si dát pozor",
		"rb.safety.seed":   "<strong>Seed zadávej jedině do hardwarové peněženky.</strong> Nikdy ho nepiš do počítače, do mobilu, do e-mailu ani na žádnou webovou stránku, ani když o to někdo žádá.",
		"rb.safety.words":  "<strong>Slova z částí</strong> zadávej jedině do nástroje <em>inh-offline</em> na počítači odpojeném od internetu (při nouzovém postupu do jiného nástroje pro SLIP-39, také bez internetu).",
		"rb.safety.nobody": "Nikomu cizímu (ani „technické podpoře“) nedávej slova, seed ani heslo z obálky.",
		"rb.safety.wipe":   "<strong>Po skončení</strong> smaž soubor s klíčem z počítače i z USB klíče. Kdo ho má, otevře si databázi. Papírky se slovy spal nebo vrať držitelům.",
		"rb.safety.move":   "<em>(volitelné, ale důrazně doporučené)</em> Po obnově přesuň prostředky do nové peněženky.",
		"rb.safety.slow":   "Nespěchej; pokud něco nesedí, zastav se a poraď se s technicky zdatnou osobou%s.",

		"rb.notes.wallet": "Peněženka: poznámky",
		"rb.notes.family": "Vzkaz",

		"rb.print":   "🖨 Vytisknout / Uložit jako PDF",
		"rb.edit":    "✏️ Upravit",
		"rb.newform": "← nový formulář",
		"rb.margins": "Při tisku nech okraje na „Výchozí“.",
	} {
		csMessages[k] = v
	}
}
