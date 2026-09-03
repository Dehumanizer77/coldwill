# inh — DMS (dead-man's switch)

Online služba: pravidelne žiada vlastníka o **check-in**; po dlhom tichu požiada
**dôveryhodné osoby** o potvrdenie; po potvrdení + ochrannej lehote pošle
**GPG-zašifrovanú obálku** (ktorú sama nikdy nevie prečítať) technicky zdatnej osobe.

**Kanály:** e-mail (primárny) + voliteľne **Signal** (druhý kanál). Každá správa
ide na všetky kanály, ktoré má daný adresát nastavené.

## Bezpečnostný model

- **Nikdy nedrží plaintext.** Drží len `envelope.asc` (ciphertext zašifrovaný na
  GPG kľúč technicky zdatnej osoby) a pri výstrele ho len pošle.
- **Fail-safe:** ak self-test zlyhá (mail nedostupný, obálka chýba/nie je PGP,
  stav sa nedá zapísať), DMS **alertuje, ale NEODPÁLI**.
- **Dvojkrokové odkazy:** check-in aj confirm sú GET stránka + POST tlačidlo, aby
  ich nespustil automatický „link prefetch" e-mailových skenerov.
- **Tokeny** v odkazoch sú HMAC z `hmac_secret`. Aj keby odkaz unikol, najhorší
  prípad je bezpečný (check-in len oddiali výstrel; confirm aj tak potrebuje
  človeka + lehotu + kovové podiely).
- **Druhý kanál nikdy nezablokuje výstrel.** Nefunkčný Signal = alert e-mailom,
  nie porucha; obálka odíde, ak ju doručí **aspoň jeden** kanál (inak sa výstrel
  neuskutoční a skúsi sa znova pri ďalšom tiku).
- **Lokálny postfix bez TLS, cudzí relay len s TLS.** Na loopbacku sa STARTTLS
  zámerne nepoužíva: bajty nikdy neopustia stroj a postfix nemá (a nemôže mať)
  certifikát na „127.0.0.1" — Go by inak každý e-mail vrátane obálky odmietol
  poslať. Pri ne-loopback `smtp_addr` sa STARTTLS naopak vyžaduje aj s overením
  certifikátu.
- Bankový trezor je nezávislá druhá cesta k obálke — DMS je „best-effort".

## Časová os (default)

```
check-in mesačne → po 60 dňoch ticha: výzva potvrdzovateľom (dôveryhodné osoby)
→ potvrdenie (ktorýkoľvek) → 7-dňový odklad s dennými upozorneniami vlastníkovi
→ výstrel: obálka e-mailom technicky zdatnej osobe.   Check-in kedykoľvek všetko ruší.
DMS → vlastníkovi týždenne „som zdravý"; pri poruche alert.
```

## Vytvorenie obálky (offline, ručne)

Do súboru daj len **passphrase k peňaženke** (+ prípadne krátky pokyn) a zašifruj na
verejný GPG kľúč technicky zdatnej osoby (`keys/friend.asc`; fingerprint si over
nezávisle, nie z toho istého kanála, ktorým kľúč prišiel):

```
gpg --import keys/friend.asc
gpg --armor --encrypt --recipient friend@example.com passphrase.txt
mv passphrase.txt.asc envelope.asc      # toto ide do /data, nie do gitu
shred -u passphrase.txt
```

## Deployment

Na serveri s Dockerom a postfixom to spraví `./deploy.sh` — nič nenasadzuje sám
od seba, každý krok vypíše a na Apache ani postfix nesiahne.

**Compose netreba.** Ak je k dispozícii `docker compose` (v2), použije ho; inak
(a s `--no-compose`) spraví to isté cez čisté `docker build` / `docker run`.
Staré `docker-compose` v1 vedome ignoruje — je EOL a nevie ani
`${VAR:-default}` v compose súbore, takže by to potichu rozbil.

```bash
./deploy.sh --check                              # len preflight, nič nemení
./deploy.sh --envelope ~/envelope.asc            # ostré nasadenie (e-mail)
./deploy.sh --envelope ~/envelope.asc --signal   # + Signal kanál
./deploy.sh --config-only --force-config         # len prepíš config.json
./deploy.sh --envelope ~/envelope.asc --no-compose      # bez compose
./deploy.sh --envelope ~/envelope.asc --test-timings   # skúšobný beh, viď nižšie
```

Čo skript spraví: overí docker/compose/postfix/porty → založí `/opt/inh-dms/data`
(0700) → skopíruje obálku (a odmietne ju, ak to nie je ASCII-armored PGP) →
interaktívne vypýta adresy, čísla a potvrdzovateľov a zapíše `config.json`
(0600, `hmac_secret` z `openssl rand -hex 32`) → **overí config cez `inh-dms
--validate`** → zbuildí a spustí kontajner → skontroluje HTTP, log a `state.json`
→ vypíše Apache vhost a **tvoj check-in odkaz do záložiek**.

Beží idempotentne: existujúci `config.json` ani obálku neprepíše (na to je
`--force-config`, ktorý starý config zálohuje), takže sa dá pustiť znova.

`--test-timings` je skúšobná inštancia: intervaly v minútach namiesto dní,
vlastný dátový adresár (`/opt/inh-dms-test`), štart z čistého stavu (starý
`state.json` zmaže) a **obálka aj výzvy potvrdzovateľom idú tebe** — skúška
nesmie napísať skutočným ľuďom, lebo vo fáze čakania sa výzva opakuje každých
pár minút — celý reťazec (check-in → ticho → potvrdenie → odpočet → výstrel) sa
tak dá prejsť za pár minút bez toho, aby si niekoho vystrašil. Po doskúšaní
`docker compose down && rm -rf /opt/inh-dms-test`.

Ručne je to to isté: `mkdir -p /opt/inh-dms/data`, `config.json` z
`config.example.json` (`hmac_secret` = `openssl rand -hex 32`), obálku do
`/opt/inh-dms/data/envelope.asc`, potom
`INH_UID=$(id -u) INH_GID=$(id -g) docker compose up -d` (e-mail) alebo to isté
s `--profile signal` (+ Signal). Bez compose:

```bash
docker build -t inh-dms .
docker run -d --name inh-dms --restart unless-stopped \
  --user "$(id -u):$(id -g)" --network host \
  -v /opt/inh-dms/data:/data inh-dms
```

`--user` tam **musí** byť: image beží ako `nonroot` (uid 65532), ale `/data`
patrí tebe a `config.json` je 0600 — bez toho kontajner skončí na
`open /data/config.json: permission denied`. Host sieť je zámer:
`smtp_addr` dosiahne lokálny postfix na `127.0.0.1:25`, Signal API je na
`127.0.0.1:8080` a `listen_addr` sa naviaže na hostiteľský loopback, kam ide
Apache proxy.

Apache ako reverse proxy (`a2enmod proxy proxy_http headers`, TLS cez certbot):

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

Overenie po štarte (skript to kontroluje sám, ale vedieť to treba):

1. `curl -s https://dms.example.com/` → „Služba beží."
2. do minúty príde e-mail **[DMS] v poriadku** (a ak je zapnutý Signal, aj správa
   na Signale) — to je zároveň dôkaz, že self-test prešiel a obálka je čitateľná,
3. klikni v ňom check-in odkaz → „Ďakujem" a v `state.json` sa zmení `last_check_in`,
4. `docker logs inh-dms` neobsahuje `failed`,
5. **check-in odkaz si ulož do záložiek / KeePass DB** — je stabilný, ale závisí
   od `hmac_secret` (ten si tiež odlož; po jeho zmene platia iné odkazy),
6. do runbooku a do KeePass DB zapíš, že DMS existuje a ako sa vypína
   (`docker compose down` = DMS je preč, dedičstvo tým netrpí).

Údržba: `docker compose pull && docker compose up -d` (Signal kontajner),
`docker save inh-dms | gzip > inh-dms.tar.gz` do archívu k ostatným artefaktom.

## Konfigurácia

Viď `config.example.json` (alebo si ho nechaj vygenerovať cez `deploy.sh`).
Trvania prijímajú `30d`, `7d`, `12h`, `90m`. `inh-dms --validate` config načíta,
skontroluje a skončí — hodí sa po ručnej úprave, kým službu reštartneš.
Povinné: `public_base_url`, `from_email`, `user_email`, `friend_email`,
`envelope_path`, `state_path`, `hmac_secret` (≥16 znakov), aspoň 1 `confirmer`.

## Signal (druhý kanál)

Voliteľný. Vynechaj blok `signal` aj všetky `*_signal` čísla → všetko ide len
e-mailom. Konfigurácia je **buď celá, alebo žiadna**: číslo bez `signal.api_url`
(alebo naopak) je chyba pri štarte — polovičná konfigurácia by potichu zahodila
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

Self-test kontroluje presne toto `/v1/accounts` — realistické tiché zlyhanie je
**odlinkované zariadenie**, nie spadnutý kontajner. Prichádzajúce správy
nespracúvame (check-in je odkaz v správe, funguje z oboch kanálov).

Obálka ide pri výstrele aj na Signal (je to ciphertext, kanál je jedno). Ak by
bola príliš dlhá na jednu Signal správu, Signal ju odmietne a doručí ju e-mail —
preto je e-mail primárny a obálka má obsahovať len passphrase + krátky pokyn.

## Korektnosť

`go test ./...` pokrýva celú časovú os (deterministicky, injektovaný čas + fake
mailer): pripomienky, prechod do čakania, potvrdenie → odpočet → výstrel,
zrušenie check-inom, fail-safe pri poruche/chýbajúcej obálke, a HTTP handlery.
Pre Signal navyše: rozposlanie na oba kanály, výstrel pri spadnutom Signale,
výstrel cez Signal pri spadnutom maile, žiadny kanál → žiadny výstrel + retry,
HTTP klient proti fake API a validácia konfigurácie.
