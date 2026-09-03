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
- **Obnova** → vložíš ≥ K častí (každú na svojom riadku) → zloží key-file.
- **Runbook** → vyplníš „kto-čo-kde" → vygeneruje tlačiteľný návod pre rodinu
  (klik *Vytlačiť / Uložiť ako PDF*). Dokument **neobsahuje žiadne tajomstvá**.

## Ako to zapadá

- Časti sa ryjú na **kovové platničky** (médium musí pojať 23 slov), jedna na držiteľa/lokalitu.
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
