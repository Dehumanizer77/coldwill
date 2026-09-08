# inh: Bitcoin Inheritance System

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
   ČASŤ #1          ČASŤ #2          ČASŤ #3            ZAPEČATENÁ OBÁLKA
 kov, osoba A     kov, osoba B     kov, osoba C       bankový trezor + DMS
      │                │                │                      │
      └────────┬───────┴────────────────┘                      │
               │   stačia ľubovoľné K z N (tu 2 z 3)           │
               ▼                                               │
          KEY-FILE                                             │
               │                                               │
               ▼   otvorí (bez hesla)                          │
        KeePass databáza  ──►  SEED + ostatné heslá            │
                                     │                         │
                                     │                    PASSPHRASE
                                     └───────────┬─────────────┘
                                                 ▼
                                            ₿  BITCOIN
```

- **Kovové časti** (SLIP-39 slová vyryté do kovu) držia rôzni ľudia na rôznych
  miestach. Ktorýchkoľvek **K z N** (v príklade 2 z 3) zloží *key-file*: súbor,
  ktorým sa odomkne KeePass databáza so **seedom** a všetkými ostatnými
  prístupmi. Prah aj počet častí sú nastaviteľné: 3 z 5, 2 z 4, čo ti vyhovuje.
- **Zapečatená obálka** obsahuje **passphrase k peňaženke**. Leží v bankovom
  trezore a navyše ju po tvojej smrti pošle *dead-man's switch* e-mailom.
- **Bitcoin = seed + passphrase.** Kto má len časti, vidí databázu, ale mince
  neminie. Kto má len obálku, má heslo, ktoré bez seedu nie je na nič. Menej než
  K častí je bezcenných.

**Dead-man's switch** je len pohodlie: pravidelne sa ťa pýta „žiješ?“, a keď sa
dlho neozveš a dôveryhodná osoba to potvrdí, po ochrannej lehote pošle obálku
(koľko potvrdení treba, si nastavuješ, štandardne stačí jedno). Istá cesta vedie
cez banku: DMS sa dá kedykoľvek vypnúť a dedičstvu to neublíži.

Podrobnosti nižšie; kto chce len vedieť „ako sa k tomu rodina dostane“, môže
skončiť tu a prečítať si [Recovery](#recovery-postup-pre-netechnického-dediča).

> ⚠️ **BEZPEČNOSTNÉ PRAVIDLO č. 1**
>
> Do tohto repozitára **nikdy** nepatria reálne tajomstvá:
> seed slová (BIP-39), passphrase k peňaženke, key-file ani jeho SLIP-39 časti,
> heslo ku KeePass DB, reálny `.kdbx`, obsah „posmrtnej obálky“.
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
  - [Dead-man's switch (`dms/`)](#dead-mans-switch-dms): obálky, nasadenie, Signal
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
| Gating | **Miernejší**: DB chráni len key-file; passphrase (v obálke) chráni len BTC. |
| Key-file | **160-bit** → 23 slov na SLIP-39 časť. |

### Architektúra: dva faktory

- **Faktor A („posmrtná obálka“):** obsahuje **passphrase k peňaženke**. Uložená
  len tam, kam sa rodina dostane až po smrti vlastníka: zapečatená v **bankovom
  trezore** a doručí ju **DMS** (zašifrovaná na technicky zdatnú osobu). Kým
  vlastník žije, passphrase má len on → **BTC kontroluje on**. Obálok môže byť
  aj viac: tá istá viacerým ľuďom kvôli zálohe, alebo rôzne obálky rôznym ľuďom,
  aby nikto sám nemal všetko (viď [Obálky](#obálky-koľko-ich-je-a-komu-idú)).
- **Faktor B (kovové časti):** key-file ku KeePass DB rozdelený cez **SLIP-39
  (K-z-N)** na slová, vyryté do kovu, časti rozmiestnené po lokalitách.

**KeePass DB** (seed + všetky ostatné heslá a prístupy + návod) je zašifrovaná;
jej **ciphertext (`.kdbx`) môže byť pokojne aj v cloude**, bez key-filu je to
zbytočný balast.

```
KeePass DB sa otvorí  =  key-file (K z N kovových častí)        # -> seed + ostatné heslá
BTC                   =  seed (z DB)  +  passphrase (z obálky)  # -> [K z N častí] + [obálka]

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

| Lokalita | Faktor B (kovová časť) | Faktor A (obálka) | Ciphertext `.kdbx` |
|---|---|---|---|
| **lokalita A**, hlavný dedič | časť #1 | – | kópia |
| **bankový trezor** | – | **zapečatená obálka** | kópia |
| **lokalita B**, dôveryhodná osoba | časť #2 | – | kópia |
| **lokalita C**, technicky zdatná osoba | časť #3 | (od DMS po spustení, šifrovaná na ňu) | kópia |
| **Cloud** | – | – | kópia (šifrovaná) |
| **DMS** | – | obálky šifrované na svojich príjemcov | – |

Recovery BTC vyžaduje **K z N častí + obálku** (z banky alebo od DMS) → čiže
„hlavný dedič + jeden dôveryhodný pomocník“. Jedna časť sama o sebe je
bezcenná, preto je riziko u jednotlivých držiteľov nízke.


### Recovery (postup pre netechnického dediča)

1. Otvorí tlačený **runbook** (kópie sú v banke aj u dôveryhodných osôb).
2. Zavolá **technicky zdatnej osobe**, ktorá ho prevedie postupom (aj cez video).
3. Získa **obálku s passphrase**, buď z bankového trezoru, alebo ju už poslal DMS.
4. Pozbiera **K častí** od držiteľov.
5. V **offline nástroji**: SLIP-39 slová → key-file → otvorí KeePass DB
   (štandardnou appkou) → dostane sa k seedu a všetkým prístupom.
6. Na peňaženke obnoví zo **seedu + passphrase** (z obálky) → BTC.
7. (Voliteľné) presunie BTC do vlastnej novej peňaženky.

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
- Obálka v banke neporušená; `.kdbx` kópie sa otvárajú.
- Test check-inu DMS.
- **Raz za rok nanečisto celá obnova** na náhradnom zariadení, aspoň raz aj na
  Windows (tá binárka sa inde otestovať nedá).
- Aktualizácia DB pri zmene prístupov + re-tlač runbooku.

### Čo systém nerieši

Ak zomrú obaja rodičia a deti sú maloleté, **technicky** je to pokryté: dve
dôveryhodné osoby zložia K z N častí a obálku a k mincám sa dostanú. Čo systém
vyriešiť **nemôže**, je kto ich potom drží a spravuje, kým deti dospejú: kto
zloží časti, môže nimi disponovať. Je to otázka dôvery a dedičského práva, nie
kryptografie. Ak sa to má riešiť, patrí to do **závetu** (vlastnoručný, bez
notára). Ten nesmie nikdy obsahovať seed ani passphrase, len odkaz na runbook
a držiteľov častí, lebo závet končí v súdnom spise.

### Princípy životnosti

1. **Otvorené štandardy na kritickej ceste** = softvér je nahraditeľný: SLIP-39
   (s test vektormi), `.kdbx`, GPG. Aj bez tohto kódu sa dá recovery spraviť
   štandardnými nástrojmi, runbook to popisuje.
2. **Self-contained artefakty:** statický Go binár (offline; závisí len na kernel
   ABI a prehliadači), Docker image (online; zmrazený userland).
3. **Minimum závislostí**, pripnuté a reprodukovateľne buildnuteľné.
4. **Archivovať** spolu s dátami: binárky pre viac OS, zdroják, build recept,
   Docker image (tarball) a papierový fallback návod.


## Nástroje

Stack je **Go**: statické binárky a „Go 1 compatibility promise“, jeden jazyk na
obe časti, žiadne cgo ani GUI knižnice. `.kdbx` náš kód **netvorí ani nečíta**;
to robí štandardná KeePass appka (KeePassXC a spol.) s naším key-filom.

### Offline nástroj (`offline/`)

Air-gapped nástroj na (a) vygenerovanie key-filu a jeho rozdelenie na **K-z-N
SLIP-39** časti a (b) zloženie častí späť na key-file pri obnove.

> ⚠️ **Spúšťaj LEN na offline (air-gapped) stroji.** Nástroj počúva výhradne na
> loopbacku (`127.0.0.1`) a odmietne sa spustiť na inej adrese. Nerobí žiadny
> sieťový prístup. Po skončení ho zavri.

#### Build

Potrebné je len Go (≥ 1.24, kvôli `crypto/pbkdf2`). Žiadne externé závislosti.

```
cd offline && go build -o inh-offline .
```

Výsledok je **jeden statický binár** bez závislostí (beží na hocijakom Linuxe
danej architektúry).

Dedič ale nemusí sedieť pri Linuxe, takže do ceremónie patria binárky pre všetky
platformy naraz:

```
cd offline && ./build-all.sh      # -> dist/ + SHA256SUMS
```

Vyrobí Linux, Windows a macOS (Apple Silicon aj Intel), spolu okolo 34 MB. Celý
obsah `dist/` ide na USB kľúč ku každému kovovému médiu aj do bankového trezoru; runbook potom hovorí, ktorý súbor na ktorom počítači spustiť.

Go kríž-kompiluje samo, žiadny ďalší toolchain netreba. Otestovať sa tu dá len
binárka pre tento stroj: **Windows a macOS treba vyskúšať na cieľovom systéme**,
patrí to do ročnej údržby. Binárky nie sú podpísané, takže Windows SmartScreen aj
macOS Gatekeeper zahlásia varovanie. Runbook popisuje, ako ho preklikať.

#### Spustenie

```
./inh-offline                # http://127.0.0.1:8777
./inh-offline --addr 127.0.0.1:9000
```

Automaticky sa otvorí **systémový default browser** (Linux `xdg-open`, Windows
`rundll32`, macOS `open`), nezávisle od toho, ktorý browser je nainštalovaný.
Vypneš to cez `--open=false` (vtedy si otvor vypísanú URL ručne).

Na **Windows** spusti `inh-offline.exe` (dvojklik alebo z `cmd`); otvorí default
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

#### Ako to zapadá

- Časti sa prenášajú na **kovové médium** (musí pojať 23 slov: platnička, valček, kapsula,
  kazeta so zasúvacími písmenkami…), jedna na držiteľa/lokalitu. Prenášajú sa len
  **prvé 4 písmená** každého slova. Setup ich na
  výstupe zvýrazní tučným, aby bolo jasné, čo ide na kov.
- Key-file zamyká **KeePass DB** (KeePassXC: ochrana = *Key file*). KeePassXC
  súbor, ktorý nie je 32 B / 64 hex / KeyFile-XML, deterministicky **zahashuje
  (SHA-256)**, takže náš **20-bajtový** (160-bit) súbor funguje a pri obnove sa
  zloží identicky. Stiahnutý key-file sú **surové bajty**, nie hex text: pri
  ručnom fallbacku ho z hexu vyrob cez `xxd -r -p` (alebo `perl -e 'print pack
  "H*","…"'` / `python3 -c '…bytes.fromhex(…)'`, ak `xxd` na stroji nie je).
  Hex uložený ako text je iný súbor a DB neotvorí.
- Overené na **KeePassXC 2.7.10** (`keepassxc-cli`): key-file vytvorí a otvorí
  `.kdbx` bez hesla, kľúč obnovený z ľubovoľných 2 z 3 častí otvorí tú istú DB,
  nesprávny key-file aj „hex ako text“ ju neotvoria.
- **Passphrase k peňaženke NIE je v DB**, je v posmrtnej obálke (banka / DMS).

#### Korektnosť

SLIP-39 jadro (`internal/slip39`) je overené proti **všetkým 45 oficiálnym
SLIP-39 test vektorom** + round-trip pre 128/192/256-bit. `go test ./...`.

### Dead-man's switch (`dms/`)

Online služba: pravidelne žiada vlastníka o **check-in**; po dlhom tichu požiada
**dôveryhodné osoby** o potvrdenie; po potvrdení + ochrannej lehote pošle
**GPG-zašifrované obálky** (ktoré sama nikdy nevie prečítať) ich príjemcom.

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
6. Po uplynutí lehoty bez veta → odošlú sa obálky (GPG, rozšifruje ich len ten,
   na koho kľúč boli zašifrované).
7. Falošné či zlomyseľné potvrdenie nie je katastrofa: obálka je **bez dostatku
   kovových častí zbytočná**.

Poistky: zaseknutý alebo nepotvrdený DMS **nikdy nezablokuje dedičstvo**, lebo
obálka je aj v bankovom trezore. DMS je **best-effort a odstrániteľný**, istá
cesta vedie cez banku.

#### Bezpečnostný model

- **Nikdy nedrží plaintext.** Drží len ciphertext obálok (zašifrovaných na GPG
  kľúče príjemcov) a pri výstrele ich len pošle. Kto obálku otvorí, sa rozhoduje
  **offline pri jej výrobe**, tým, na ktorý kľúč ju zašifruješ. Config hovorí
  len, kam sa pošle.
- **Fail-safe:** ak self-test zlyhá (mail nedostupný, niektorá obálka chýba
  alebo nie je PGP, stav sa nedá zapísať), DMS **alertuje, ale NEODPÁLI**.
- **Dvojkrokové odkazy:** check-in aj confirm sú GET stránka + POST tlačidlo, aby
  ich nespustil automatický „link prefetch“ e-mailových skenerov.
- **Tokeny** v odkazoch sú HMAC z `hmac_secret`. Aj keby odkaz unikol, najhorší
  prípad je bezpečný (check-in len oddiali výstrel; confirm aj tak potrebuje
  človeka + lehotu + kovové podiely).
- **Druhý kanál nikdy nezablokuje výstrel.** Nefunkčný Signal = alert e-mailom,
  nie porucha; každá obálka odíde, ak ju doručí **aspoň jeden** kanál (čo sa
  nedoručí, skúsi sa znova pri ďalšom tiku).
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
→ výstrel: obálky e-mailom svojim príjemcom.   Check-in kedykoľvek všetko ruší.
DMS → vlastníkovi týždenne „som zdravý“; pri poruche alert.
```

#### Obálky: koľko ich je a komu idú

Obálok môže byť viac a **každá má vlastných príjemcov**:

```json
"envelopes": [
  { "id": "passphrase", "path": "/data/envelope-passphrase.asc",
    "to": [ {"name":"Prvá","email":"prva@…","signal":"+…"},
            {"name":"Druhá","email":"druha@…"} ] },
  { "id": "pristupy",   "path": "/data/envelope-pristupy.asc",
    "note": "Vnútri sú ostatné prístupy, nie passphrase.",
    "to": [ {"name":"Tretia","email":"tretia@…"} ] }
]
```

Pokrýva to dva rôzne zámery:

- **tá istá obálka viacerým ľuďom** = záloha, aby jeden nedostupný človek
  neodrezal celú DMS cestu (napr. keď zomrieš aj ty aj on),
- **rôzne obálky rôznym ľuďom** = rozdelenie znalostí; nikto sám nemá všetko.

Pravidlá, ktoré si služba stráži: obálka bez príjemcu alebo dve s rovnakým `id`
sa odmietnu pri štarte; self-test kontroluje **každý** súbor (chýbajúci alebo
nie-PGP = porucha a **nevystrelí sa vôbec**, nie polovica); pri výstrele si
pamätá, ktoré obálky už odišli, takže sa doposiela len zvyšok a nikomu nepríde
tá istá dvakrát. Check-in (veto) túto pamäť **zmaže**, po ňom musí ísť pri
ďalšom ostrom výstrele von zase všetko.

Starý zápis `envelope_path` + `friend_email` naďalej funguje ako jedna obálka
s jedným príjemcom.

> **Pozor pri rozdeľovaní obsahu:** v bankovom trezore musí byť **všetko**. DMS
> je best-effort, banka je istá cesta. Inak si rozdelením vyrobíš scenár, kde
> jeden nereagujúci príjemca odreže časť dedičstva.

#### Kam patrí verejný GPG kľúč príjemcu

Do `keys/` (adresár je len konvencia, `*.asc` sú gitignorované, aby repo
neprezrádzalo identity). DMS ten kľúč nikdy nevidí, potrebuješ ho len ty pri
výrobe obálky:

```
gpg --export --armor <key-id> > keys/friend.asc
```

#### Vytvorenie obálok (offline, ručne)

Do súboru daj len **passphrase k peňaženke** (+ prípadne krátky pokyn) a zašifruj na
verejný GPG kľúč technicky zdatnej osoby (`keys/friend.asc`; fingerprint si over
nezávisle, nie z toho istého kanála, ktorým kľúč prišiel):

```
gpg --import keys/friend.asc
gpg --armor --encrypt --recipient friend@example.com passphrase.txt
mv passphrase.txt.asc envelope.asc      # toto ide do /data, nie do gitu
shred -u passphrase.txt
```

Ak má tú istú obálku vedieť otvoriť viac ľudí, zašifruj ju na viac kľúčov naraz
(`--recipient A --recipient B`). To je nezávislé od toho, komu sa doručí.

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
./deploy.sh --envelope ~/envelope.asc            # ostré nasadenie (e-mail)
./deploy.sh --envelope passphrase=~/a.asc --envelope pristupy=~/b.asc  # viac obálok
./deploy.sh --envelope ~/envelope.asc --signal   # + Signal kanál
./deploy.sh --config-only --force-config         # len prepíš config.json
./deploy.sh --envelope ~/envelope.asc --no-compose      # bez compose
./deploy.sh --envelope ~/envelope.asc --test-timings   # skúšobný beh, viď nižšie
```

Čo skript spraví: overí docker/compose/postfix/porty → založí `/opt/inh-dms/data`
(0700) → skopíruje obálky (a odmietne tú, ktorá nie je ASCII-armored PGP) →
interaktívne vypýta adresy, čísla a potvrdzovateľov a zapíše `config.json`
(0600, `hmac_secret` z `openssl rand -hex 32`) → **overí config cez `inh-dms
--validate`** → zbuildí a spustí kontajner → skontroluje HTTP, log a `state.json`
→ vypíše Apache vhost a **tvoj check-in odkaz do záložiek**.

Beží idempotentne: existujúci `config.json` ani obálky neprepíše (na to je
`--force-config`, ktorý starý config zálohuje), takže sa dá pustiť znova.

`--test-timings` je skúšobná inštancia a je **úplne oddelená od ostrej**: iný
dátový adresár (`/opt/inh-dms-test`), iné mená kontajnerov (`inh-dms-test`,
`inh-signal-test`), iné porty (8188 / 8180) aj vlastný compose projekt
(`-p inh-dms-test`). Skúška teda nemôže zhodiť ani nahradiť bežiaci ostrý DMS a
proxy naň nezačne smerovať; ak ostrý DMS beží vedľa, skript to pri štarte
vypíše.

K tomu: intervaly v minútach namiesto dní, štart z čistého stavu (starý
`state.json` zmaže), odkazy mieria na `http://127.0.0.1:8188` (cez SSH tunel) a
**obálky aj výzvy potvrdzovateľom idú tebe**, skúška nesmie napísať skutočným
ľuďom, lebo vo fáze čakania sa výzva opakuje každých pár minút. Celý reťazec
(check-in → ticho → potvrdenie → odpočet → výstrel) sa tak dá prejsť za pár
minút bez toho, aby si niekoho vystrašil. Po doskúšaní:

```bash
docker compose -p inh-dms-test down     # alebo: docker rm -f inh-dms-test
rm -rf /opt/inh-dms-test
```

Ručne je to to isté: `mkdir -p /opt/inh-dms/data`, `config.json` z
`config.example.json` (`hmac_secret` = `openssl rand -hex 32`), obálky do
`/opt/inh-dms/data/` (cesty v configu sú z pohľadu kontajnera, `/data/…`), potom
`INH_UID=$(id -u) INH_GID=$(id -g) docker compose up -d` (e-mail) alebo to isté
s `--profile signal` (+ Signal). Bez compose:

```bash
docker build -t inh-dms .
docker run -d --name inh-dms --restart unless-stopped \
  --user "$(id -u):$(id -g)" --network host \
  -v /opt/inh-dms/data:/data inh-dms
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
   na Signale). To je zároveň dôkaz, že self-test prešiel a všetky obálky sú čitateľné,
3. klikni v ňom check-in odkaz → „Ďakujem“ a v `state.json` sa zmení `last_check_in`,
4. `docker logs inh-dms` neobsahuje `failed`,
5. **check-in odkaz si ulož do záložiek / KeePass DB**. Je stabilný, ale závisí
   od `hmac_secret` (ten si tiež odlož; po jeho zmene platia iné odkazy),
6. do runbooku a do KeePass DB zapíš, že DMS existuje a ako sa vypína
   (`docker compose down` = DMS je preč, dedičstvo tým netrpí).

Údržba: `docker compose pull && docker compose up -d` (Signal kontajner),
`docker save inh-dms | gzip > inh-dms.tar.gz` do archívu k ostatným artefaktom.

#### Konfigurácia

Viď `config.example.json` (alebo si ho nechaj vygenerovať cez `deploy.sh`).
Trvania prijímajú `30d`, `7d`, `12h`, `90m`. `inh-dms --validate` config načíta,
skontroluje a skončí. Hodí sa po ručnej úprave, kým službu reštartneš.
Povinné: `public_base_url`, `from_email`, `user_email`, `state_path`,
`hmac_secret` (≥16 znakov), aspoň 1 `confirmer` a aspoň jedna obálka
(`envelopes[]`, alebo starý `envelope_path` + `friend_email`).

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
"user_signal": "+…", "friend_signal": "+…",
"confirmers": [ { "id": "friend", "…": "…", "signal": "+…" } ]
```

Kontajner (`bbernhard/signal-cli-rest-api`) drží linknuté zariadenie; DMS mu len
POSTuje text na `/v2/send` cez loopback. Linkovanie (raz, pri deployi):

```
docker compose up -d signal
# otvor v prehliadači (cez SSH tunel) a naskenuj QR v Signale:
#   Signal → Nastavenia → Prepojené zariadenia → +
xdg-open http://127.0.0.1:8080/v1/qrcodelink?device_name=inh-dms
curl -s http://127.0.0.1:8080/v1/accounts     # musí obsahovať from_number
```

Self-test kontroluje presne toto `/v1/accounts`: realistické tiché zlyhanie je
**odlinkované zariadenie**, nie spadnutý kontajner. Prichádzajúce správy
nespracúvame (check-in je odkaz v správe, funguje z oboch kanálov).

Obálka ide pri výstrele aj na Signal (je to ciphertext, kanál je jedno). Ak by
bola príliš dlhá na jednu Signal správu, Signal ju odmietne a doručí ju e-mail.
Preto je e-mail primárny a obálka má obsahovať len passphrase + krátky pokyn.

#### Korektnosť

`go test ./...` pokrýva celú časovú os (deterministicky, injektovaný čas + fake
mailer): pripomienky, prechod do čakania, potvrdenie → odpočet → výstrel,
zrušenie check-inom, fail-safe pri poruche/chýbajúcej obálke, a HTTP handlery.
Pre Signal navyše: rozposlanie na oba kanály, výstrel pri spadnutom Signale,
výstrel cez Signal pri spadnutom maile, žiadny kanál → žiadny výstrel + retry,
HTTP klient proti fake API a validácia konfigurácie.

## Prevádzkové pravidlá repa

- **Reálna mapa do repa nepatrí.** Kto drží ktorú časť, kde je obálka a na akom
  serveri beží DMS, to žije vo **vytlačenom runbooku** a v **KeePass DB**, nie
  v texte v repe. Dokumentácia je zámerne zovšeobecnená (lokalita A/B/C,
  „technicky zdatná osoba“).
- Verejné GPG kľúče príjemcov (`keys/*.asc`) a vytlačené runbooky (`*.pdf`) sú
  gitignorované, aby repo neprezrádzalo identity.
