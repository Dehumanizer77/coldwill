# inh — offline Setup/Recovery tool

Air-gapped nástroj na (a) vygenerovanie key-filu a jeho rozdelenie na **K-z-N
SLIP-39** časti a (b) zloženie častí späť na key-file pri obnove.

> ⚠️ **Spúšťaj LEN na offline (air-gapped) stroji.** Nástroj počúva výhradne na
> loopbacku (`127.0.0.1`) a odmietne sa spustiť na inej adrese. Nerobí žiadny
> sieťový prístup. Po skončení ho zavri.

## Build

Potrebné je len Go (≥ 1.24, kvôli `crypto/pbkdf2`). Žiadne externé závislosti.

```
go build -o inh-offline .
```

Výsledok je **jeden statický binár** bez závislostí (beží na hocijakom Linuxe
danej architektúry). Pre Windows/macOS: `GOOS=windows go build` / `GOOS=darwin`.

## Run

```
./inh-offline                # http://127.0.0.1:8777
./inh-offline --addr 127.0.0.1:9000
```

Automaticky sa otvorí **systémový default browser** (Linux `xdg-open`, Windows
`rundll32`, macOS `open`) — nezávisle od toho, ktorý browser je nainštalovaný.
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
  Slová píšeš **tak, ako sú na kove**: platničky majú miesto len na **4 písmená**
  a nástroj si zvyšok doplní (v SLIP-39 zozname je slovo prvými štyrmi písmenami
  určené jednoznačne — 1024 slov, 1024 rôznych prefixov). Tri písmená sú
  nejednoznačné, tie nástroj odmietne a povie ktoré slovo. Veľkosť písmen,
  číslovanie riadkov a interpunkcia sa ignorujú. Formulár renderuje server, takže
  **funguje aj s vypnutým JavaScriptom** — skript len dopĺňa slová a posúva
  kurzor; kontrola a zloženie kľúča prebiehajú vždy na serveri.
- **Runbook** → vyplníš „kto-čo-kde" → vygeneruje tlačiteľný návod pre rodinu
  (klik *Vytlačiť / Uložiť ako PDF*). Dokument **neobsahuje žiadne tajomstvá**.

## Ako to zapadá

- Časti sa ryjú na **kovové platničky** (médium musí pojať 23 slov), jedna na
  držiteľa/lokalitu. Ryjú sa len **prvé 4 písmená** každého slova — setup ich na
  výstupe zvýrazní tučným, aby bolo jasné, čo ide na kov.
- Key-file zamyká **KeePass DB** (KeePassXC: ochrana = *Key file*). KeePassXC
  súbor, ktorý nie je 32 B / 64 hex / KeyFile-XML, deterministicky **zahashuje
  (SHA-256)** — náš **20-bajtový** (160-bit) súbor teda funguje a pri obnove sa
  zloží identicky. Stiahnutý key-file sú **surové bajty**, nie hex text: pri
  ručnom fallbacku ho z hexu vyrob cez `xxd -r -p` (alebo `perl -e 'print pack
  "H*","…"'` / `python3 -c '…bytes.fromhex(…)'`, ak `xxd` na stroji nie je).
  Hex uložený ako text je iný súbor a DB neotvorí.
- Overené na **KeePassXC 2.7.10** (`keepassxc-cli`): key-file vytvorí a otvorí
  `.kdbx` bez hesla, kľúč obnovený z ľubovoľných 2 z 3 častí otvorí tú istú DB,
  nesprávny key-file aj „hex ako text" ju neotvoria.
- **Passphrase k peňaženke NIE je v DB** — je v posmrtnej obálke (banka / DMS).

## Korektnosť

SLIP-39 jadro (`internal/slip39`) je overené proti **všetkým 45 oficiálnym
SLIP-39 test vektorom** + round-trip pre 128/192/256-bit. `go test ./...`.
