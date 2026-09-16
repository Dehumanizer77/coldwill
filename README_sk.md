# coldwill: Bitcoin Inheritance System

> Slovenská verzia. Primárna je [anglická](README.md), ktorá sa udržiava ako
> prvá; ak sa obe rozídu, platí anglická.

Systém dedičstva Bitcoinu pre rodinu: **jednoduchý pre netechnického dediča**,
odolný voči krádeži, strate aj katastrofe a **plne self-custody**, teda bez
kustodiánov, notárov a tretích strán. Repozitár obsahuje návrh a dva nástroje:
offline generátor/obnovu kľúča s tlačiteľným runbookom a online dead-man's
switch.

## Ako to funguje (v kocke)

**Za života** ovládaš Bitcoin len ty. **Po smrti** ho rodina zloží z dvoch
nezávislých vecí. Ani jedna z nich sama nestačí.

Koľko častí vznikne a koľko ich treba na zloženie (**K z N**) si volíš pri
generovaní; obrázok ukazuje bežnú voľbu 2 z 3.

```
   ČASŤ #1          ČASŤ #2          ČASŤ #3                OBÁLKA
 kov, osoba A     kov, osoba B     kov, osoba C   text v trezore + PDF z DMS
      │                │                │                      │
      └────────┬───────┴────────────────┘                      │
               │   stačia ľubovoľné K z N (tu 2 z 3)           │
               ▼                                               │
          KEY-FILE                                             │
               │                                               │
               ▼   otvorí (bez hesla)                          ▼
        KeePass databáza  ──►  SEED + heslá + MAPA ──► PASSPHRASE
                                     │                         │
                                     └───────────┬─────────────┘
                                                 ▼
                                            ₿  BITCOIN
```

- **Kovové časti** (SLIP-39 slová vyryté do kovu) držia rôzni ľudia na rôznych
  miestach. Ktorýchkoľvek **K z N** (v príklade 2 z 3) zloží *key-file*: súbor,
  ktorým sa odomkne KeePass databáza so **seedom** a všetkými ostatnými
  prístupmi. Prah aj počet častí sú nastaviteľné: 3 z 5, 2 z 4, čo ti vyhovuje.
- **Obálka** je obyčajný vytlačený text, v ktorom je ukrytá **passphrase
  k peňaženke**. **Mapa**, podľa ktorej sa z neho prečíta, je v databáze, takže
  text sám o sebe nič neprezradí. Leží v bankovom trezore a po tvojej smrti ju
  *dead-man's switch* navyše pošle hlavnému dedičovi ako PDF.
- **Bitcoin = seed + passphrase.** Kto má len časti, vidí databázu aj mapu, ale
  nemá text, z ktorého by čítal. Kto má len obálku, má článok a nemá ho čím
  prečítať. Menej než K častí je bezcenných.

**Dead-man's switch** je len pohodlie: pravidelne sa ťa pýta „žiješ?“, a keď sa
dlho neozveš a dôveryhodná osoba to potvrdí, po ochrannej lehote pošle obálku
hlavnému dedičovi (koľko potvrdení treba, si nastavuješ, štandardne stačí jedno). Istá cesta vedie
cez banku: DMS sa dá kedykoľvek vypnúť a dedičstvu to neublíži.

Podrobnosti nižšie; kto chce len vedieť „ako sa k tomu rodina dostane“, môže
skončiť tu a prečítať si [Recovery](#recovery-postup-pre-netechnického-dediča).

> ⚠️ **BEZPEČNOSTNÉ PRAVIDLO č. 1**
>
> Do tohto repozitára **nikdy** nepatria reálne tajomstvá:
> seed slová (BIP-39), passphrase k peňaženke, key-file ani jeho SLIP-39 časti,
> heslo ku KeePass DB, reálny `.kdbx`, obálka ani jej mapa.
>
> Repozitár je len **kód a dokumentácia**. Tajomstvá vznikajú a žijú výhradne
> offline (kov, bankový trezor, šifrovaný `.kdbx`). `.gitignore` je nastavený
> tak, aby bežné tajné súbory nešlo omylom commitnúť, ale spoľahni sa hlavne na
> to, že ich do repa vôbec neprinesieš.


## Obsah

- [Ako to funguje (v kocke)](#ako-to-funguje-v-kocke): celý systém na jednej obrazovke
- [Návrh](#návrh): čo to rieši, architektúra, rozmiestnenie, recovery, údržba
- [Nástroje](#nástroje)
  - [Offline nástroj (`offline/`)](#offline-nástroj-offline): key-file, SLIP-39 časti, runbook
  - [Dead-man's switch (`dms/`)](#dead-mans-switch-dms): obálka, nasadenie, Signal
- [Prevádzkové pravidlá repa](#prevádzkové-pravidlá-repa)

## Návrh

### Čo to rieši

Cieľ je systém, ktorý je:

- **jednoduchý pre netechnického dediča**,
- **bez bezpečnostného kompromisu**,
- **odolný** voči útoku / hacku / strate / katastrofe / vojne,
- **plne self-custody**, čiže žiadne tretie strany ani kustodiáni,
- **maximálne nezávislý od platformy** (prežije OS upgrady aj zmeny knižníc).

Návrhové rozhodnutia, z ktorých všetko ostatné vyplýva:

| | |
|---|---|
| Hrozby | Hlavná priorita = **rodina sa k tomu dostane**; sekundárne odolnosť voči krádeži. |
| Zariadenie | Nepodstatné, záleží len na **seede** (+ passphrase). Obnoviť sa dá na hocijakej kompatibilnej peňaženke. |
| Peňaženka | Existujúca, nová sa **nevytvára**. Jeden seed, s passphrase. |
| Dedičia | Viacero dôveryhodných osôb; aspoň jedna **technicky zdatná**, ostatné môžu byť netechnické. Stačí jeden technický pomocník. |
| Hardvér u dedičov | Nie, maximálne **kovové médium** so slovami. |
| Tretie strany / notár | Nie. |
| Schéma | **K-z-N**, čiže prah K z N častí (ľubovoľný počet; príklady nižšie používajú 2-z-3). |
| Kontrola za života | BTC plne kontroluje **vlastník** (passphrase). DB so seedom a ostatnými heslami vie K-z-N dôveryhodných osôb otvoriť aj za jeho života, ale BTC bez passphrase neminú. |
| Spúšťač | Smrť. Dead-man's switch je voliteľný a kedykoľvek odstrániteľný. |
| Geografia | Časti rozmiestnené po viacerých **lokalitách**, aby ich nezničila jedna udalosť. |
| Údržba | Raz ročne. |
| Ostatné prístupy | Patria do systému, stačí **jedna KeePass DB**. |
| Gating | **Miernejší**: DB chráni len key-file; passphrase (ukrytá v obálke) chráni len BTC. |
| Key-file | **160-bit** → 23 slov na SLIP-39 časť. |

### Architektúra: dva faktory

- **Faktor A (obálka):** obyčajný vytlačený text, v ktorom je ukrytá **passphrase
  k peňaženke**, čitateľná len s mapou uloženou v databáze. Je len tam, kam sa
  rodina dostane až po smrti vlastníka: zapečatená v **bankovom trezore** a ako
  PDF ju hlavnému dedičovi pošle **DMS**. Nič sa nešifruje, lebo text bez
  databázy nemá cenu. Kým vlastník žije, passphrase má len on → **BTC
  kontroluje on**.
- **Faktor B (kovové časti):** key-file ku KeePass DB rozdelený cez **SLIP-39
  (K-z-N)** na slová, vyryté do kovu, časti rozmiestnené po lokalitách.

**KeePass DB** (seed + všetky ostatné heslá a prístupy + mapa k obálke + návod) má u seba
**hlavný dedič** a voliteľne je jedna aj v bankovom trezore. Držitelia častí ju
zámerne nemajú a v cloude nie je. Kto databázu nemá a nevie, kde je, tomu
zložený key-file nepomôže, ani keď sa dá dokopy dosť držiteľov. A keby ju mal
každý držiteľ, pri každej zmene prístupov by bolo treba obísť všetkých.

```
KeePass DB sa otvorí  =  key-file (K z N kovových častí)        # -> seed + ostatné heslá + mapa
BTC                   =  seed (z DB)  +  passphrase (z obálky)  # -> [K z N častí] + [obálka]
Passphrase            =  obálka prečítaná podľa mapy            # samotný text nič nepovie

Miernejší gating: K-z-N častí otvorí DB (aj za života vlastníka), ale BTC bez
passphrase z obálky nikto neminie. Strata obálky => strata len BTC, nie DB.
```

#### Prečo SLIP-39 na key-file (a nie na seed)

SLIP-39 = Shamirovo zdieľanie tajomstva + kódovanie do slov s kontrolným
súčtom. **Nie je viazané na seed peňaženky**, vie rozdeliť ľubovoľný 160-bit
kľúč. Použije sa teda na **náhodný key-file**, ktorým sa zamyká KeePass:

1. vygeneruje sa náhodný 160-bit kľúč `K` (**nie** seed),
2. `K` → SLIP-39 split (K-z-N) → N × 23 slov → na kov,
3. `K` = key-file pre KeePass,
4. `K` sa vymaže, žije len ako N kovových častí.

Bonus: tieto slová **nie sú seed**. Keby ich niekto našiel a naťukal do
peňaženky, dostane prázdno. Implementácia je overená proti **oficiálnym SLIP-39
test vektorom**, žiadna vlastná kryptografia.


### Rozmiestnenie

Príklad pre 2-z-3; častí môže byť ľubovoľný počet:

| Lokalita | Faktor B (kovová časť) | Faktor A (obálka) | Databáza (`.kdbx`) |
|---|---|---|---|
| **lokalita A**, hlavný dedič | časť #1 | (PDF od DMS po spustení) | ✓ |
| **bankový trezor** | voliteľne jedna z častí | **obálka** (vytlačený text) | ✓ (voliteľne) |
| **lokalita B**, dôveryhodná osoba | časť #2 | – | – |
| **lokalita C**, technicky zdatná osoba | časť #3 | – | – |
| **DMS** | – | obálka ako PDF pre hlavného dediča | – |

Recovery BTC vyžaduje **K z N častí + obálku** (z banky alebo od DMS) → čiže
„hlavný dedič + jeden dôveryhodný pomocník“. Jedna časť sama o sebe je
bezcenná, preto je riziko u jednotlivých držiteľov nízke.

Obe voľby pre trezor sú vo formulári runbooku ako zaškrtávacie políčka, takže ich
vytlačená mapa uvedie. Každá z nich ale robí trezor lákavejším: s oboma má ten,
kto sa dostane do schránky, jednu časť, obálku aj databázu, a databázu mu otvorí
už K−1 ďalších častí; mapa v nej potom prečíta passphrase z obálky.

#### Obálka

Passphrase nie je nikde napísaná. Obálka je obyčajný text, napríklad článok
z novín, v ktorého znakoch je passphrase ukrytá, a mapa, podľa ktorej sa z neho
prečíta, je v KeePass databáze ([ako to funguje](#ukrytie-passphrase-v-texte)).
Bez databázy je text bezcenný, a preto môže ležať v trezore ako obyčajný
výtlačok a ísť e-mailom aj Signalom ako obyčajné PDF: nie sú žiadne kľúče, ktoré
treba vyrobiť, uchovať a nestratiť, a dedič nič nerozšifrováva.

Ten istý text existuje dvakrát, vytlačený v trezore a ako PDF v DMS, ktorý ho
pošle hlavnému dedičovi a nikomu inému. Keby ste zomreli obaja naraz, táto kópia
skončí v schránke, ktorú nikto nečíta, a ostatní použijú výtlačok z trezoru.

#### Ukrytie passphrase v texte

Každý znak passphrase sa vezme z ľubovoľnej pozície slova v obyčajnom texte,
napríklad v článku z novín, alebo z medzery a znamienok za slovom. V trezore je
vytlačený text, v DMS ten istý text ako PDF a mapa pozícií je uložená v databáze.

Offline nástroj **Passphrase v texte** vyberie pozície náhodne, prečíta mapu
späť a overí presnú zhodu s passphrase. Dostaneš mapu a PDF textu. Opakovaný
znak dostane iné miesto, kým má text z čoho. Mapa má pre všetky znaky rovnaký
tvar, takže neoznačuje, ktoré sú veľké písmená, znamienka alebo medzery.

**Passphrase.** Povolené sú všetky tlačiteľné ASCII znaky (0x20–0x7E): malé aj
veľké písmená, číslice 0–9, znamienka a medzera. Diakritika a € nie sú povolené.
Každá medzera sa počíta, aj na začiatku a konci hesla. Znaky `\ | ~ ^ { } [ ] < >`
a spätný apostrof sa v bežnom článku hľadajú ťažko; odporúčame ich vynechať.

**Text.** Poslúži článok, strana z knihy aj vlastný text. Pri mape neuvádzaj
zdroj (noviny, dátum, nadpis), podľa ktorého by sa dal text dohľadať bez trezoru.
Každý znak hesla potrebuje použiteľnú pozíciu: číslo „1874“ poskytne všetky štyri
číslice, veľké písmená nájdeš na začiatkoch viet a v skratkách. Odsek s kontaktom
môže prirodzene doplniť e-mail (`@ _ .`), web (`/ : ? = &`), telefón (`+`),
hashtag (`#`) či cenu (`$`). Pri zmene hesla môže papier ostať, ak obsahuje
použiteľné pozície pre všetky nové znaky; vždy vytvor novú mapu.

Nástroj v celom texte aj nadpise zmení typografické úvodzovky „ “ ” na `"`,
‚ ‘ ’ na `'`, pomlčky – — na `-` a … na `...`. Vyberá len ASCII znaky a nikdy
nepočíta cez „ch“, „dz“ či „dž“ bez ohľadu na veľkosť písmen. Medzeru vyberie
len medzi slovami na tom istom riadku PDF, nikdy na konci riadku či odseku ani
vnútri čísla „4 300“, kde je nezalomiteľná medzera. **Tlač dodané PDF**: jeho
zalomenie riadkov je súčasťou mapy.

**Pravidlá čítania mapy:**

- **Odsek** je blok textu medzi prázdnymi riadkami; nadpis sa nepočíta.
  Pri vložení textu bez prázdnych riadkov nástroj vytvorí z každého riadku odsek.
- **Slová** počítaj v každom odseku od 1, vrátane krátkych slov a čísel.
  Samostatná pomlčka či iné znamienko nie je slovo; „4 300“ je jedno slovo.
- **Znaky slova** počítaj od 1 od začiatku slova vrátane interpunkcie,
  úvodzoviek a medzery v „4 300“; „ch“, „dz“ aj „dž“ sú po dva znaky.
- Pozícia hneď za posledným znakom slova je **medzera** za ním. Ak nasledujú
  samostatné znamienka, pokračuj v počítaní ich znakov aj medzier medzi nimi
  na tom istom vytlačenom riadku.
- Znak odpíš **presne tak, ako je vytlačený**, vrátane veľkých písmen a medzier,
  a znaky spoj v poradí mapy.

Napríklad pre tento prvý odsek:

> Obecné zastupiteľstvo minulý týždeň schválilo 86 tisíc eur na opravu budovy,
> ktorú miestni odjakživa volajú starý mlyn. Stavba pri potoku pamätá ešte rok
> 1874 a posledné roky chátrala.

mapa znie:

```
1. znak hesla:  1. odsek, 18. slovo, 1. znak slova
2. znak hesla:  1. odsek, 24. slovo, 3. znak slova
3. znak hesla:  1. odsek, 11. slovo, 7. znak slova
4. znak hesla:  1. odsek,  7. slovo, 6. znak slova
5. znak hesla:  1. odsek, 17. slovo, 5. znak slova
```

Výsledok je `S7, .`: `S` zo „Stavba“, `7` z „1874“, čiarka z „budovy,“, medzera
za „tisíc“ a bodka z „mlyn.“. Medzera platí, len ak „tisíc“ a nasledujúce slovo
sú na tom istom riadku PDF. V skutočnej mape sú len pozície a pravidlá čítania.

**Mapa je uložená v databáze**, a práve preto je text sám o sebe bezcenný: kto
sa pozrie do schránky alebo prečíta dedičov e-mail, vidí obyčajný článok. Kto
otvorí databázu, má aj mapu, takže s časťou a databázou v trezore chýba k minciam
stále len K−1 častí.

Kópia runbooku v tej istej schránke prezradí, že je v nej obálka, takže článok sa
skryje pred letmým pohľadom, nie pred tým, kto si runbook prečíta. Chráni ho až
databáza.


### Recovery (postup pre netechnického dediča)

1. Otvorí tlačený **runbook** (kópie sú v banke aj u dôveryhodných osôb).
2. Zavolá **technicky zdatnej osobe**, ktorá ho prevedie postupom (aj cez video).
3. Získa **obálku**: vytlačený text z bankového trezoru alebo PDF, ktoré mu už poslal DMS.
4. Pozbiera **K častí** od držiteľov (jedna môže byť v bankovom trezore).
5. V **offline nástroji**: SLIP-39 slová → key-file → otvorí KeePass DB (svoju,
   alebo tú z trezoru) štandardnou appkou → dostane sa k seedu a všetkým prístupom.
6. Podľa mapy z databázy prečíta z obálky **passphrase**.
7. Na peňaženke obnoví zo **seedu + passphrase** → BTC.
8. (Voliteľné) presunie BTC do vlastnej novej peňaženky.

Na kove sú slová spravidla len ako **4-písmenové skratky**, kovové médium viac
pozícií nemá. Nie je to problém: SLIP-39 zoznam je navrhnutý tak, že prvé štyri
písmená určujú slovo jednoznačne (1024 slov, 1024 rôznych prefixov), a
obnovovací nástroj si zvyšok doplní sám. Tri písmená by už jednoznačné neboli,
tie nástroj odmietne a povie, ktoré slovo prepísať.

Runbook obsahuje aj **núdzový postup** bez týchto nástrojov: časti sú štandardný
SLIP-39, zloží ich hocijaký SLIP-39 nástroj, výsledný *master secret* v hexe **je
obsahom key-filu**. Ak taký nástroj skratky neprijme, celé slová sa dohľadajú
v oficiálnom SLIP-39 zozname.


### Ročná údržba

- Čitateľnosť a prítomnosť **všetkých častí** (potvrdiť s držiteľmi).
- Obálka v banke neporušená a databáza sa otvára: dedičova, aj tá v trezore, ak tam je.
- Test check-inu DMS.
- PDF v DMS sa zhoduje s výtlačkom v trezore.
- **Raz za rok nanečisto celá obnova** na náhradnom zariadení, aspoň raz aj na
  Windows (tá binárka sa inde otestovať nedá).
- Aktualizácia DB pri zmene prístupov (aj tej v trezore, ak tam je) + re-tlač runbooku.

### Čo systém nerieši

Ak zomrú obaja rodičia a deti sú maloleté, **technicky** je to pokryté: dve
dôveryhodné osoby zložia K z N častí, vezmú obálku z trezoru a k mincám sa dostanú. Čo systém
vyriešiť **nemôže**, je kto ich potom drží a spravuje, kým deti dospejú: kto
zloží časti, môže nimi disponovať. Je to otázka dôvery a dedičského práva, nie
kryptografie. Ak sa to má riešiť, patrí to do **závetu** (vlastnoručný, bez
notára). Ten nesmie nikdy obsahovať seed ani passphrase, len odkaz na runbook
a držiteľov častí, lebo závet končí v súdnom spise.

### Princípy životnosti

1. **Otvorené štandardy na kritickej ceste** = softvér je nahraditeľný: SLIP-39
   (s test vektormi), `.kdbx`, vytlačený text. Aj bez tohto kódu sa dá recovery spraviť
   štandardnými nástrojmi, runbook to popisuje.
2. **Self-contained artefakty:** statický Go binár (offline; závisí len na kernel
   ABI a prehliadači), Docker image (online; zmrazený userland).
3. **Minimum závislostí**, pripnuté a reprodukovateľne buildnuteľné.
4. **Archivovať** spolu s dátami: binárky pre viac OS, zdroják, build recept,
   Docker image (tarball) a papierový fallback návod.


## Nástroje

Stack je **Go**: statické binárky a „Go 1 compatibility promise“, jeden jazyk na
obe časti, žiadne cgo ani GUI knižnice. `.kdbx` náš kód **netvorí ani nečíta**;
to robí štandardná KeePass appka s naším key-filom: KeePassXC zo stránky
<https://keepassxc.org/download/>, alebo hocijaká iná implementácia formátu
(KeePass, KeePassDX, Strongbox).

### Offline nástroj (`offline/`)

Air-gapped nástroj na (a) vygenerovanie key-filu a jeho rozdelenie na **K-z-N
SLIP-39** časti a (b) zloženie častí späť na key-file pri obnove.

> ⚠️ **Spúšťaj LEN na offline (air-gapped) stroji.** Nástroj počúva výhradne na
> loopbacku (`127.0.0.1`) a odmietne sa spustiť na inej adrese. Nerobí žiadny
> sieťový prístup. Po skončení ho zavri.

#### Build

Potrebné je len Go (≥ 1.24, kvôli `crypto/pbkdf2`). Žiadne externé závislosti.
Písmo pre text na vytlačenie, Liberation Serif pod licenciou SIL Open Font
License, je vložené do binárky a licencia je vedľa neho v `internal/pdf/`.

```
cd offline && go build -o coldwill .
```

Výsledok je **jeden statický binár** bez závislostí (beží na hocijakom Linuxe
danej architektúry).

Dedič ale nemusí sedieť pri Linuxe, takže do ceremónie patria binárky pre všetky
platformy naraz:

```
cd offline && ./build-all.sh      # -> dist/ + SHA256SUMS
```

Vyrobí Linux, Windows a macOS (Apple Silicon aj Intel), spolu okolo 34 MB, do
gitignorovaného `dist/`.

Dedič si ich sťahuje, takže po každej zmene nástroja ich treba zverejniť:

```
cd offline && ./release.sh          # tag podľa dnešného dátumu
cd offline && ./release.sh v1.0.0   # alebo vlastný tag
```

Zbuildí a nahrá GitHub release, kam runbook posiela dediča (`/releases/latest`,
adresa, ktorá funguje ďalej aj po vydaní novších). Binárky sa **buildia u teba a
len sa nahrávajú**. CI build zámerne nie je: znamenal by dôverovať cudziemu
runneru s binárkou, ktorá skladá key-file, a zmysel lokálneho buildu je práve to,
že nemusíš.

Releases namiesto commitnutých súborov preto, že 34 MB pri každom prebuilde by
skončilo v histórii každého, kto si repo naklonuje.

Runbook dedičovi povie, ktorý súbor na ktorom počítači spustiť. Ak máš tie súbory
uložené aj inde, runbook má na to pole a vytlačený návod dediča tam nasmeruje,
keby adresa na stiahnutie prestala fungovať.

Go kríž-kompiluje samo, žiadny ďalší toolchain netreba. Otestovať sa tu dá len
binárka pre tento stroj: **Windows a macOS treba vyskúšať na cieľovom systéme**,
patrí to do ročnej údržby. Binárky nie sú podpísané, takže Windows SmartScreen aj
macOS Gatekeeper zahlásia varovanie. Runbook popisuje, ako ho preklikať.

#### Spustenie

```
./coldwill                # http://127.0.0.1:8777
./coldwill --addr 127.0.0.1:9000
```

Automaticky sa otvorí **systémový default browser** (Linux `xdg-open`, Windows
`rundll32`, macOS `open`), nezávisle od toho, ktorý browser je nainštalovaný.
Vypneš to cez `--open=false` (vtedy si otvor vypísanú URL ručne).

Na **Windows** spusti `coldwill.exe` (dvojklik alebo z `cmd`); otvorí default
browser rovnako. Pri štarte beží **power-on self-test** (známy SLIP-39 vektor +
round-trip); ak zlyhá, nástroj sa nespustí.

- **Nový backup** → vygeneruje 160-bit key-file, rozdelí ho na N častí po 23
  slov s prahom K (default 2-z-3), zobrazí ich na vyrytie, ponúkne stiahnutie
  key-filu a hex.
- **Obnova** → **samostatné pole na každé slovo**; **počet častí** aj **počet
  slov v časti** (20/23/26/33) sa nastavujú číselníkom, ktorý bloky rovno pridá
  alebo odoberie (predvolene 2 časti × 23 slov). Ak by sa odobratím stratili už
  zadané slová, najprv sa opýta. Po štvrtom písmene sa slovo
  doplní celé, pole zozelenie a kurzor skočí na ďalšie; vloženie celej časti zo
  schránky rozhádže slová do polí. Nezmyselné slovo pole očervenie. Pod
  formulárom ostáva aj **vloženie častí ako textu**.
  Slová píšeš **tak, ako sú na kove**: kovové médium má miesto len na **4 písmená**
  a nástroj si zvyšok doplní (v SLIP-39 zozname je slovo prvými štyrmi písmenami
  určené jednoznačne (1024 slov, 1024 rôznych prefixov). Tri písmená sú
  nejednoznačné, tie nástroj odmietne a povie ktoré slovo. Veľkosť písmen,
  číslovanie riadkov a interpunkcia sa ignorujú. Formulár renderuje server, takže
  **funguje aj s vypnutým JavaScriptom**. Skript len dopĺňa slová a posúva
  kurzor; kontrola a zloženie kľúča prebiehajú vždy na serveri.
- **Runbook** → vyplníš „kto-čo-kde“ → vygeneruje tlačiteľný návod pre rodinu
  (klik *Vytlačiť / Uložiť ako PDF*). Dokument **neobsahuje žiadne tajomstvá**.
- **Passphrase v texte** → vložíš text a passphrase → dostaneš zoznam pozícií
  znakov (odsek, slovo, znak slova) do databázy a PDF textu na vytlačenie (viď
  [Ukrytie passphrase v texte](#ukrytie-passphrase-v-texte)). Passphrase sa už
  na žiadnej stránke nezobrazí a PDF nemá žiadne metadáta.

#### Ako to zapadá

- Časti sa prenášajú na **kovové médium** (musí pojať 23 slov: platnička, valček, kapsula,
  kazeta so zasúvacími písmenkami…), jedna na držiteľa/lokalitu. Prenášajú sa len
  **prvé 4 písmená** každého slova. Setup ich na
  výstupe zvýrazní tučným, aby bolo jasné, čo ide na kov.
- Key-file zamyká **KeePass DB** (KeePassXC: ochrana = *Key file*; na stiahnutie
  na <https://keepassxc.org/download/>). KeePassXC
  súbor, ktorý nie je 32 B / 64 hex / KeyFile-XML, deterministicky **zahashuje
  (SHA-256)**, takže náš **20-bajtový** (160-bit) súbor funguje a pri obnove sa
  zloží identicky. Stiahnutý key-file sú **surové bajty**, nie hex text: pri
  ručnom fallbacku ho z hexu vyrob cez `xxd -r -p` (alebo `perl -e 'print pack
  "H*","…"'` / `python3 -c '…bytes.fromhex(…)'`, ak `xxd` na stroji nie je).
  Hex uložený ako text je iný súbor a DB neotvorí.
- Overené na **KeePassXC 2.7.10** (`keepassxc-cli`): key-file vytvorí a otvorí
  `.kdbx` bez hesla, kľúč obnovený z ľubovoľných 2 z 3 častí otvorí tú istú DB,
  nesprávny key-file aj „hex ako text“ ju neotvoria.
- **Passphrase k peňaženke NIE je v DB** ani nikde napísaná: obálka (výtlačok v trezore, PDF v DMS) nesie text a databáza mapu, ktorá ho prečíta.

#### Korektnosť

SLIP-39 jadro (`internal/slip39`) je overené proti **všetkým 45 oficiálnym
SLIP-39 test vektorom** + round-trip pre 128/192/256-bit. `go test ./...`.

### Dead-man's switch (`dms/`)

Online služba: pravidelne žiada vlastníka o **check-in**; po dlhom tichu požiada
**dôveryhodné osoby** o potvrdenie; po potvrdení + ochrannej lehote pošle
**obálku**, PDF bez databázy bezcenné, **hlavnému dedičovi**.

**Kanály:** e-mail (primárny) + voliteľne **Signal** (druhý kanál). Každá správa
ide na všetky kanály, ktoré má daný adresát nastavené.

#### Ako to funguje (tok potvrdenia)

1. **Pravidelný check-in** (klik „žijem“).
2. Vynechanie → **eskalujúce upomienky vlastníkovi** na všetky kanály.
3. Po dlhom tichu DMS **nevystrelí sám**. Pošle **technicky zdatnej osobe a
   ďalšej dôveryhodnej osobe** výzvu na potvrdenie úmrtia/trvalej neschopnosti
   (odkaz platný len pre tento cyklus + potvrdzovacie tlačidlo).
4. Po potvrdení → štart **ochrannej lehoty**, počas ktorej chodí vlastníkovi
   denné upozornenie „obálka sa pošle o X dní“. Štandardne stačí **jedno**
   potvrdenie (`confirm_quorum: 1`): planý poplach je prežiteľný, lebo obálka je
   bez kovových častí bezcenná, kým „nikto nepotvrdil“ by dedičstvo zastavilo.
   Pri `confirm_quorum: 2` počíta DMS **rôznych** potvrdzovateľov, takže jeden
   človek dvakrát sa neráta.
5. **Check-in vlastníka kedykoľvek všetko ruší.** Veto vždy vyhráva, aj počas
   odpočtu.
6. Po uplynutí lehoty bez veta → obálka odíde hlavnému dedičovi a nikomu inému.
7. Falošné či zlomyseľné potvrdenie nie je katastrofa: obálka je **bez dostatku
   kovových častí zbytočná**.

Poistky: zaseknutý alebo nepotvrdený DMS **nikdy nezablokuje dedičstvo**, lebo
obálka je aj v bankovom trezore. DMS je **best-effort a odstrániteľný**, istá
cesta vedie cez banku.

#### Bezpečnostný model

- **Nedrží nič, čo by samo osebe niečo znamenalo.** Obálka je text, ktorý bez
  databázy nič nepovie, takže napadnutý server, prečítaná schránka či zlý
  príjemca na Signale prezradia článok, nie passphrase. Preto sa posiela
  nešifrovaná a nie sú žiadne kľúče na správu.
- **Fail-safe:** ak self-test zlyhá (mail nedostupný, obálka chýba
  alebo nie je PDF, stav sa nedá zapísať), DMS **alertuje, ale NEODPÁLI**.
- **Dvojkrokové odkazy:** check-in aj confirm sú GET stránka + POST tlačidlo, aby
  ich nespustil automatický „link prefetch“ e-mailových skenerov.
- **Tokeny** v odkazoch sú HMAC z `hmac_secret`. Aj keby odkaz unikol, najhorší
  prípad je bezpečný (check-in len oddiali výstrel; confirm aj tak potrebuje
  človeka + lehotu + kovové podiely).
- **Druhý kanál nikdy nezablokuje výstrel.** Nefunkčný Signal = alert e-mailom,
  nie porucha; obálka odíde, ak ju doručí **aspoň jeden** kanál (ak žiadny,
  skúsi sa znova pri ďalšom tiku).
- **Lokálny postfix bez TLS, cudzí relay len s TLS.** Na loopbacku sa STARTTLS
  zámerne nepoužíva: bajty nikdy neopustia stroj a postfix nemá (a nemôže mať)
  certifikát na „127.0.0.1“. Go by inak každý e-mail vrátane obálky odmietol
  poslať. Pri ne-loopback `smtp_addr` sa STARTTLS naopak vyžaduje aj s overením
  certifikátu.
- Bankový trezor je nezávislá druhá cesta k obálke, DMS je „best-effort“.

#### Časová os (default)

```
check-in mesačne → po 60 dňoch ticha: výzva potvrdzovateľom (dôveryhodné osoby)
→ potvrdenie (ktorýkoľvek) → 7-dňový odklad s dennými upozorneniami vlastníkovi
→ výstrel: obálka hlavnému dedičovi.   Check-in kedykoľvek všetko ruší.
DMS → vlastníkovi týždenne „som zdravý“; pri poruche alert.
```

#### Komu ide obálka

Hlavnému dedičovi a nikomu inému:

```json
"heir": { "name": "…", "email": "…", "signal": "+…", "lang": "sk" },
"envelope_path": "/data/envelope.pdf"
```

Ide ako príloha na všetky kanály, na ktoré má dedič adresu, a stačí, keď prejde
jeden. Ak neprejde žiadny, DMS ostane v odpočte a skúsi to znova pri ďalšom
tiku. Self-test číta súbor pri každom tiku a chýbajúci súbor alebo súbor, ktorý
nie je PDF, je porucha, ktorá výstrel zastaví. Zlý súbor (napríklad mapa) sa tak
odhalí, kým sa dá vymeniť, a nie v deň odoslania.

Obálka je PDF z nástroja **Passphrase v texte** v offline nástroji, to isté,
ktoré tlačíš do trezoru. Na serveri patrí do `/data`, nikdy nie do gitu. Keď
meníš text alebo passphrase, vymeň naraz výtlačok, PDF na serveri aj mapu
v databáze.

Config, v ktorom ešte zostali `envelopes`, `friend_email` alebo `friend_signal`,
sa pri štarte odmietne, aby sa DMS nespustil s potichu zahodenými príjemcami.

#### Deployment

Na serveri s Dockerom a postfixom to spraví `dms/deploy.sh` (spúšťa sa
z adresára `dms/`). Nič nenasadzuje sám od seba, každý krok vypíše a na Apache
ani postfix nesiahne.

**Compose netreba.** Ak je k dispozícii `docker compose` (v2), použije ho; inak
(a s `--no-compose`) spraví to isté cez čisté `docker build` / `docker run`.
Staré `docker-compose` v1 vedome ignoruje, lebo je EOL a nevie ani
`${VAR:-default}` v compose súbore, takže by to potichu rozbil.

```bash
./deploy.sh --check                              # len preflight, nič nemení
./deploy.sh --envelope ~/envelope.pdf            # ostré nasadenie (e-mail)
./deploy.sh --envelope ~/envelope.pdf --signal   # + Signal kanál
./deploy.sh --config-only --force-config         # len prepíš config.json
./deploy.sh --envelope ~/envelope.pdf --no-compose      # bez compose
./deploy.sh --envelope ~/envelope.pdf --test-timings   # skúšobný beh, viď nižšie
```

Čo skript spraví: overí docker/compose/postfix/porty → založí `/opt/coldwill-switch/data`
(0700) → skopíruje obálku (a odmietne súbor, ktorý nie je PDF) →
interaktívne vypýta tvoju adresu, adresu hlavného dediča, čísla a potvrdzovateľov a zapíše `config.json`
(0600, `hmac_secret` z `openssl rand -hex 32`) → **overí config cez `coldwill-switch
--validate`** → zbuildí a spustí kontajner → skontroluje HTTP, log a `state.json`
→ vypíše Apache vhost a **tvoj check-in odkaz do záložiek**.

Beží idempotentne: existujúci `config.json` ani obálku neprepíše (na to je
`--force-config`, ktorý starý config zálohuje), takže sa dá pustiť znova.

`--test-timings` je skúšobná inštancia a je **úplne oddelená od ostrej**: iný
dátový adresár (`/opt/coldwill-switch-test`), iné mená kontajnerov (`coldwill-switch-test`,
`coldwill-signal-test`), iné porty (8188 / 8180) aj vlastný compose projekt
(`-p coldwill-switch-test`). Skúška teda nemôže zhodiť ani nahradiť bežiaci ostrý DMS a
proxy naň nezačne smerovať; ak ostrý DMS beží vedľa, skript to pri štarte
vypíše.

K tomu: intervaly v minútach namiesto dní, štart z čistého stavu (starý
`state.json` zmaže), odkazy mieria na `http://127.0.0.1:8188` (cez SSH tunel) a
**obálka aj výzvy potvrdzovateľom idú tebe**, skúška nesmie napísať skutočným
ľuďom, lebo vo fáze čakania sa výzva opakuje každých pár minút. Celý reťazec
(check-in → ticho → potvrdenie → odpočet → výstrel) sa tak dá prejsť za pár
minút bez toho, aby si niekoho vystrašil. Po doskúšaní:

```bash
docker compose -p coldwill-switch-test down     # alebo: docker rm -f coldwill-switch-test
rm -rf /opt/coldwill-switch-test
```

Ručne je to to isté: `mkdir -p /opt/coldwill-switch/data`, `config.json` z
`config.example.json` (`hmac_secret` = `openssl rand -hex 32`), obálku do
`/opt/coldwill-switch/data/` (cesty v configu sú z pohľadu kontajnera, `/data/…`), potom
`COLDWILL_UID=$(id -u) COLDWILL_GID=$(id -g) docker compose up -d` (e-mail) alebo to isté
s `--profile signal` (+ Signal). Bez compose:

```bash
docker build -t coldwill-switch .
docker run -d --name coldwill-switch --restart unless-stopped \
  --user "$(id -u):$(id -g)" --network host \
  -v /opt/coldwill-switch/data:/data coldwill-switch
```

`--user` tam **musí** byť: image beží ako `nonroot` (uid 65532), ale `/data`
patrí tebe a `config.json` je 0600. Bez toho kontajner skončí na
`open /data/config.json: permission denied`. Host sieť je zámer:
`smtp_addr` dosiahne lokálny postfix na `127.0.0.1:25`, Signal API je na
`127.0.0.1:8080` a `listen_addr` sa naviaže na hostiteľský loopback, kam ide
Apache proxy.

Pred službu patrí reverse proxy s TLS (DMS počúva len na loopbacku a HTTP).

Apache (`a2enmod proxy proxy_http headers`, TLS cez certbot):

```apache
<VirtualHost *:443>
  ServerName dms.example.com
  ProxyPreserveHost On
  ProxyPass        / http://127.0.0.1:8088/
  ProxyPassReverse / http://127.0.0.1:8088/
  RequestHeader set X-Forwarded-Proto https
  # TLS direktívy doplní certbot
</VirtualHost>
```

nginx (to isté, TLS tiež cez certbot):

```nginx
server {
    listen 443 ssl;
    server_name dms.example.com;
    # ssl_certificate / ssl_certificate_key doplní certbot

    location / {
        proxy_pass http://127.0.0.1:8088;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

Overenie po štarte (skript to kontroluje sám, ale vedieť to treba):

1. `curl -s https://dms.example.com/` → „Služba beží.“
2. do minúty príde e-mail **[DMS] v poriadku** (a ak je zapnutý Signal, aj správa
   na Signale). To je zároveň dôkaz, že self-test prešiel a obálka je na mieste,
3. klikni v ňom check-in odkaz → „Ďakujem“ a v `state.json` sa zmení `last_check_in`,
4. `docker logs coldwill-switch` neobsahuje `failed`,
5. **check-in odkaz si ulož do záložiek / KeePass DB**. Je stabilný, ale závisí
   od `hmac_secret` (ten si tiež odlož; po jeho zmene platia iné odkazy),
6. do runbooku a do KeePass DB zapíš, že DMS existuje a ako sa vypína
   (`docker compose down` = DMS je preč, dedičstvo tým netrpí).

Údržba: `docker compose pull && docker compose up -d` (Signal kontajner),
`docker save coldwill-switch | gzip > coldwill-switch.tar.gz` do archívu k ostatným artefaktom.

#### Konfigurácia

Viď `config.example.json` (alebo si ho nechaj vygenerovať cez `deploy.sh`).
Trvania prijímajú `30d`, `7d`, `12h`, `90m`. `coldwill-switch --validate` config načíta,
skontroluje a skončí. Hodí sa po ručnej úprave, kým službu reštartneš.
Povinné: `public_base_url`, `from_email`, `user_email`, `state_path`,
`hmac_secret` (≥16 znakov), aspoň 1 `confirmer`, `heir` (s e-mailom alebo
Signal číslom) a `envelope_path`.

Voliteľné, ale dobré vedieť:

| kľúč | čo robí |
|---|---|
| `confirm_quorum` | koľko **rôznych** potvrdzovateľov treba na spustenie odpočtu (default 1) |
| `checkin_key_version` | zvýš a reštartuj, ak ti unikol check-in odkaz, staré odkazy prestanú platiť |
| `smtp_timeout` | strop na celú SMTP konverzáciu (default 30s), aby zaseknutý relay nezablokoval DMS |

Potvrdzovacie odkazy platia **len pre aktuálny cyklus čakania**: check-in ich
zneplatní a v ďalšom cykle sa nedajú použiť znova.

#### Signal (druhý kanál)

Voliteľný. Vynechaj blok `signal` aj všetky `*_signal` čísla → všetko ide len
e-mailom. Konfigurácia je **buď celá, alebo žiadna**: číslo bez `signal.api_url`
(alebo naopak) je chyba pri štarte. Polovičná konfigurácia by potichu zahodila
kanál, čo je presne to, čo sa v systéme, do ktorého roky nikto nepozrie, stať nesmie.

```json
"signal": { "api_url": "http://127.0.0.1:8080", "from_number": "+…" },
"user_signal": "+…", "heir": { "…": "…", "signal": "+…" },
"confirmers": [ { "id": "friend", "…": "…", "signal": "+…" } ]
```

Kontajner (`bbernhard/signal-cli-rest-api`) drží linknuté zariadenie; DMS mu len
POSTuje na `/v2/send` cez loopback text a pri výstrele obálku ako prílohu.
Linkovanie (raz, pri deployi):

```
docker compose up -d signal
# otvor v prehliadači (cez SSH tunel) a naskenuj QR v Signale:
#   Signal → Nastavenia → Prepojené zariadenia → +
xdg-open http://127.0.0.1:8080/v1/qrcodelink?device_name=coldwill-switch
curl -s http://127.0.0.1:8080/v1/accounts     # musí obsahovať from_number
```

Self-test kontroluje presne toto `/v1/accounts`: realistické tiché zlyhanie je
**odlinkované zariadenie**, nie spadnutý kontajner. Prichádzajúce správy
nespracúvame (check-in je odkaz v správe, funguje z oboch kanálov).

Obálka ide pri výstrele aj na Signal, ako PDF príloha k správe. Kanál je jedno,
lebo text je bez databázy bezcenný.

#### Korektnosť

`go test ./...` pokrýva celú časovú os (deterministicky, injektovaný čas + fake
mailer): pripomienky, prechod do čakania, potvrdenie → odpočet → výstrel,
zrušenie check-inom, fail-safe pri poruche/chýbajúcej obálke, a HTTP handlery.
Pre Signal navyše: rozposlanie na oba kanály, výstrel pri spadnutom Signale,
výstrel cez Signal pri spadnutom maile, žiadny kanál → žiadny výstrel + retry,
HTTP klient proti fake API a validácia konfigurácie.

## Jazyky

Všetko, čo číta človek, je v katalógu textov, jeden súbor na jazyk:
`offline/internal/i18n/` pre nástroj a runbook, `dms/internal/i18n/` pre e-maily
a stránky DMS. Angličtina, slovenčina a čeština sú kompletné; angličtina je
zdroj pravdy aj záloha pri chýbajúcom kľúči.

Offline nástroj berie jazyk z URL (`?lang=sk`) a formulár runbooku má vlastný
prepínač, takže sa dá vytlačiť iná jazyková verzia pre každého držiteľa. DMS
berie jazyk **na príjemcovi**: `user_lang` pre teba, `lang` pri každom
potvrdzovateľovi a pri dedičovi.

**Českú verziu si pred ostrou tlačou daj prečítať rodenému Čechovi.**

## Prevádzkové pravidlá repa

- **Reálna mapa do repa nepatrí.** Kto drží ktorú časť, kde je obálka a na akom
  serveri beží DMS, to žije vo **vytlačenom runbooku** a v **KeePass DB**, nie
  v texte v repe. Dokumentácia je zámerne zovšeobecnená (lokalita A/B/C,
  „technicky zdatná osoba“).
- Obálka a vytlačené runbooky (`*.pdf`) sú gitignorované, aby sa do repa
  nedostali.
