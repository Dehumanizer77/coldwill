# inh — Bitcoin Inheritance System

Systém dedičstva Bitcoinu pre rodinu: **jednoduchý pre netechnického dediča**,
odolný voči krádeži, strate aj katastrofe, a **plne self-custody** — bez
kustodiánov, notárov a tretích strán. Repozitár obsahuje návrh a dva nástroje:
offline generátor/obnovu kľúča s tlačiteľným runbookom a online dead-man's
switch.

> ⚠️ **BEZPEČNOSTNÉ PRAVIDLO č. 1**
>
> Do tohto repozitára **nikdy** nepatria reálne tajomstvá:
> seed slová (BIP-39), passphrase k peňaženke, key-file ani jeho SLIP-39 časti,
> heslo ku KeePass DB, reálny `.kdbx`, obsah „posmrtnej obálky".
>
> Repozitár je len **kód a dokumentácia**. Tajomstvá vznikajú a žijú výhradne
> offline (kov, bankový trezor, šifrovaný `.kdbx`). `.gitignore` je nastavený
> tak, aby bežné tajné súbory nešlo omylom commitnúť — ale spoľahni sa hlavne na
> to, že ich do repa vôbec neprinesieš.

## Čo to rieši

Cieľ je systém, ktorý je:

- **jednoduchý pre netechnického dediča**,
- **bez bezpečnostného kompromisu**,
- **odolný** voči útoku / hacku / strate / katastrofe / vojne,
- **plne self-custody** — žiadne tretie strany, žiadni kustodiáni,
- **maximálne nezávislý od platformy** (prežije OS upgrady aj zmeny knižníc).

Návrhové rozhodnutia, z ktorých všetko ostatné vyplýva:

| | |
|---|---|
| Hrozby | Hlavná priorita = **rodina sa k tomu dostane**; sekundárne odolnosť voči krádeži. |
| Zariadenie | Nepodstatné — záleží len na **seede** (+ passphrase). Obnoviť sa dá na hocijakej kompatibilnej peňaženke. |
| Peňaženka | Existujúca, nová sa **nevytvára**. Jeden seed, s passphrase. |
| Dedičia | Viacero dôveryhodných osôb; aspoň jedna **technicky zdatná**, ostatné môžu byť netechnické. Stačí jeden technický pomocník. |
| Hardvér u dedičov | Nie — maximálne **kovové platničky** so slovami. |
| Tretie strany / notár | Nie. |
| Schéma | **K-z-N** — prah K z N častí (ľubovoľný počet; príklady nižšie používajú 2-z-3). |
| Kontrola za života | BTC plne kontroluje **vlastník** (passphrase). DB so seedom a ostatnými heslami vie K-z-N dôveryhodných osôb otvoriť aj za jeho života, ale BTC bez passphrase neminú. |
| Spúšťač | Smrť. Dead-man's switch je voliteľný a kedykoľvek odstrániteľný. |
| Geografia | Časti rozmiestnené po viacerých **lokalitách**, aby ich nezničila jedna udalosť. |
| Údržba | Raz ročne. |
| Ostatné prístupy | Patria do systému — stačí **jedna KeePass DB**. |
| Gating | **Miernejší**: DB chráni len key-file; passphrase (v obálke) chráni len BTC. |
| Key-file | **160-bit** → 23 slov na SLIP-39 časť. |

## Architektúra: dva faktory

- **Faktor A — „posmrtná obálka":** obsahuje **passphrase k peňaženke**. Uložená
  len tam, kam sa rodina dostane až po smrti vlastníka: zapečatená v **bankovom
  trezore** a doručí ju **DMS** (zašifrovaná na technicky zdatnú osobu). Kým
  vlastník žije, passphrase má len on → **BTC kontroluje on**.
- **Faktor B — kovové časti:** key-file ku KeePass DB rozdelený cez **SLIP-39
  (K-z-N)** na slová, vyryté do kovu, časti rozmiestnené po lokalitách.

**KeePass DB** (seed + všetky ostatné heslá a prístupy + návod) je zašifrovaná;
jej **ciphertext (`.kdbx`) môže byť pokojne aj v cloude** — bez key-filu je to
zbytočný balast.

```
KeePass DB sa otvorí  =  key-file (K z N kovových častí)        # -> seed + ostatné heslá
BTC                   =  seed (z DB)  +  passphrase (z obálky)  # -> [K z N častí] + [obálka]

Miernejší gating: K-z-N častí otvorí DB (aj za života vlastníka), ale BTC bez
passphrase z obálky nikto neminie. Strata obálky => strata len BTC, nie DB.
```

### Prečo SLIP-39 na key-file (a nie na seed)

SLIP-39 = Shamirovo zdieľanie tajomstva + kódovanie do slov s kontrolným
súčtom. **Nie je viazané na seed peňaženky** — vie rozdeliť ľubovoľný 160-bit
kľúč. Použije sa teda na **náhodný key-file**, ktorým sa zamyká KeePass:

1. vygeneruje sa náhodný 160-bit kľúč `K` (**nie** seed),
2. `K` → SLIP-39 split (K-z-N) → N × 23 slov → na kov,
3. `K` = key-file pre KeePass,
4. `K` sa vymaže — žije len ako N kovových častí.

Bonus: tieto slová **nie sú seed**. Keby ich niekto našiel a naťukal do
peňaženky, dostane prázdno. Implementácia je overená proti **oficiálnym SLIP-39
test vektorom** — žiadna vlastná kryptografia.

## Rozmiestnenie

Príklad pre 2-z-3; častí môže byť ľubovoľný počet:

| Lokalita | Faktor B (kovová časť) | Faktor A (obálka) | Ciphertext `.kdbx` |
|---|---|---|---|
| **lokalita A** – hlavný dedič | časť #1 | — | kópia |
| **bankový trezor** | — | **zapečatená obálka** | kópia |
| **lokalita B** – dôveryhodná osoba | časť #2 | — | kópia |
| **lokalita C** – technicky zdatná osoba | časť #3 | (od DMS po spustení, šifrovaná na ňu) | kópia |
| **Cloud** | — | — | kópia (šifrovaná) |
| **DMS** | — | obálka šifrovaná na technicky zdatnú osobu | — |

Recovery BTC vyžaduje **K z N častí + obálku** (z banky alebo od DMS) → čiže
„hlavný dedič + jeden dôveryhodný pomocník". Jedna časť sama o sebe je
bezcenná, preto je riziko u jednotlivých držiteľov nízke.

## Dead-man's switch

1. **Pravidelný check-in** (klik „žijem").
2. Vynechanie → **eskalujúce upomienky vlastníkovi** na všetky kanály.
3. Po dlhom tichu DMS **nevystrelí sám** — pošle **technicky zdatnej osobe a
   ďalšej dôveryhodnej osobe** výzvu na potvrdenie úmrtia/trvalej neschopnosti
   (jednorazový odkaz + potvrdzovacie tlačidlo).
4. Po potvrdení (stačí **ktorákoľvek** z nich) → štart **ochrannej lehoty**;
   počas nej chodí vlastníkovi denné upozornenie „obálka sa pošle o X dní".
5. **Check-in vlastníka kedykoľvek všetko ruší** — veto vždy vyhráva, aj počas
   odpočtu.
6. Po uplynutí lehoty bez veta → odošle sa obálka (GPG, rozšifruje ju len
   technicky zdatná osoba).
7. Falošné či zlomyseľné potvrdenie nie je katastrofa — obálka je **bez dostatku
   kovových častí zbytočná**.

Poistky: zaseknutý alebo nepotvrdený DMS **nikdy nezablokuje dedičstvo**, lebo
obálka je aj v bankovom trezore. DMS je **best-effort a odstrániteľný** — istá
cesta vedie cez banku.

## Recovery (postup pre netechnického dediča)

1. Otvorí tlačený **runbook** (kópie sú v banke aj u dôveryhodných osôb).
2. Zavolá **technicky zdatnej osobe**, ktorá ho prevedie postupom (aj cez video).
3. Získa **obálku** — z bankového trezoru, alebo ju už poslal DMS.
4. Pozbiera **K častí** od držiteľov.
5. V **offline nástroji**: SLIP-39 slová → key-file → otvorí KeePass DB
   (štandardnou appkou) → dostane sa k seedu a všetkým prístupom.
6. Na peňaženke obnoví zo **seedu + passphrase** (z obálky) → BTC.
7. (Voliteľné) presunie BTC do vlastnej novej peňaženky.

Na kove sú slová spravidla len ako **4-písmenové skratky** — platnička viac
pozícií nemá. Nie je to problém: SLIP-39 zoznam je navrhnutý tak, že prvé štyri
písmená určujú slovo jednoznačne (1024 slov, 1024 rôznych prefixov), a
obnovovací nástroj si zvyšok doplní sám. Tri písmená by už jednoznačné neboli,
tie nástroj odmietne a povie, ktoré slovo prepísať.

Runbook obsahuje aj **núdzový postup** bez týchto nástrojov: časti sú štandardný
SLIP-39, zloží ich hocijaký SLIP-39 nástroj, výsledný *master secret* v hexe **je
obsahom key-filu**. Ak taký nástroj skratky neprijme, celé slová sa dohľadajú
v oficiálnom SLIP-39 zozname.

## Komponenty

| | |
|---|---|
| [`offline/`](offline/) | Air-gapped nástroj: generovanie 160-bit key-filu, SLIP-39 split na N častí po 23 slov, obnova z K častí, a **generátor runbooku** (tlačiteľný návod pre rodinu, bez tajomstiev). Jeden statický Go binár s web UI, počúva len na loopbacku. |
| [`dms/`](dms/) | Dead-man's switch: check-in, upomienky, potvrdenie, ochranná lehota, výstrel. E-mail + voliteľne Signal. Nikdy nedrží plaintext — len obálku zašifrovanú na príjemcu. Docker, nasadenie cez `dms/deploy.sh`. |
| [`keys/`](keys/) | Miesto pre **verejný** GPG kľúč príjemcu obálky (gitignorované). |

Stack je **Go** — statické binárky a „Go 1 compatibility promise", jeden jazyk
na obe časti, žiadne cgo ani GUI knižnice. `.kdbx` náš kód **netvorí ani
nečíta**; to robí štandardná KeePass appka (KeePassXC a spol.) s naším
key-filom.

Podrobnosti v [`offline/README.md`](offline/README.md) a
[`dms/README.md`](dms/README.md).

## Ročná údržba

- Čitateľnosť a prítomnosť **všetkých častí** (potvrdiť s držiteľmi).
- Obálka v banke neporušená; `.kdbx` kópie sa otvárajú.
- Test check-inu DMS.
- **Raz za rok nanečisto celá obnova** na náhradnom zariadení.
- Aktualizácia DB pri zmene prístupov + re-tlač runbooku.

## Čo systém nerieši

Ak zomrú obaja rodičia a deti sú maloleté, **technicky** je to pokryté — dve
dôveryhodné osoby zložia K z N častí a obálku a k mincám sa dostanú. Čo systém
vyriešiť **nemôže**, je kto ich potom drží a spravuje, kým deti dospejú: kto
zloží časti, môže nimi disponovať. Je to otázka dôvery a dedičského práva, nie
kryptografie. Ak sa to má riešiť, patrí to do **závetu** (vlastnoručný, bez
notára) — a ten nesmie nikdy obsahovať seed ani passphrase, len odkaz na runbook
a držiteľov častí, lebo závet končí v súdnom spise.

## Princípy životnosti

1. **Otvorené štandardy na kritickej ceste** = softvér je nahraditeľný: SLIP-39
   (s test vektormi), `.kdbx`, GPG. Aj bez tohto kódu sa dá recovery spraviť
   štandardnými nástrojmi — runbook to popisuje.
2. **Self-contained artefakty:** statický Go binár (offline; závisí len na kernel
   ABI a prehliadači), Docker image (online; zmrazený userland).
3. **Minimum závislostí**, pripnuté a reprodukovateľne buildnuteľné.
4. **Archivovať** spolu s dátami: binárky pre viac OS, zdroják, build recept,
   Docker image (tarball) a papierový fallback návod.

## Prevádzkové pravidlá repa

- **Reálna mapa do repa nepatrí.** Kto drží ktorú časť, kde je obálka a na akom
  serveri beží DMS — to žije vo **vytlačenom runbooku** a v **KeePass DB**, nie
  v texte v repe. Dokumentácia je zámerne zovšeobecnená (lokalita A/B/C,
  „technicky zdatná osoba").
- Verejné GPG kľúče príjemcov (`keys/*.asc`) a vytlačené runbooky (`*.pdf`) sú
  gitignorované, aby repo neprezrádzalo identity.
