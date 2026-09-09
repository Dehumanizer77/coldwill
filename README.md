# inh: Bitcoin Inheritance System

A Bitcoin inheritance system your family can actually use: simple enough for a
non-technical heir, resistant to theft, loss and disaster, and fully
self-custodial with no third parties, custodians or notaries. The repository
holds the design and two tools, an offline key generator and recovery tool that
also prints a runbook for the family, and an online dead man's switch.

*Slovensky: [README_sk.md](README_sk.md).*

## How it works, in a nutshell

**While you are alive**, only you control the Bitcoin. **After you die**, your
family reassembles it from two independent things. Neither is enough on its own.

How many shares exist, and how many are needed to reassemble the key (**K of
N**), is chosen when you generate them. The drawing shows a common choice, 2 of 3.

```
    SHARE #1         SHARE #2         SHARE #3            SEALED ENVELOPE
  metal, person A  metal, person B  metal, person C     bank vault + switch
       │                │                │                      │
       └────────┬───────┴────────────────┘                      │
                │   any K of N (here 2 of 3)                    │
                ▼                                               │
            KEY FILE                                            │
                │                                               │
                ▼   opens (no password)                         │
         KeePass database  ──►  SEED + other credentials        │
                                      │                         │
                                      │                   PASSPHRASE
                                      └──────────┬──────────────┘
                                                 ▼
                                            ₿  BITCOIN
```

- **Metal shares.** SLIP-39 words stamped into metal, held by different people
  in different places. Any **K of N** of them reassemble a *key file*, which
  unlocks a KeePass database holding the wallet **seed** and every other
  credential. Threshold and count are yours to pick: 3 of 5, 2 of 4, whatever
  fits the people you trust.
- **A sealed envelope** holds the **wallet passphrase**. It sits in a bank vault,
  and after your death the *dead man's switch* also mails it.
- **Bitcoin = seed + passphrase.** Whoever holds only shares can read the
  database but cannot spend the coins. Whoever holds only the envelope has a
  password that is useless without the seed. Fewer than K shares are worthless.

The **dead man's switch** is a convenience, not a dependency. It asks you
periodically whether you are alive, and once you stop answering and someone you
trust confirms, it mails the envelope after a grace period. How many
confirmations it takes is configurable and defaults to one. The bank vault is
the path that always works: the switch can be turned off at any time without
harming the inheritance.

> **This is not audited software.** It has tests, and the SLIP-39 core is checked
> against the official test vectors, but nobody independent has reviewed it. If
> you are going to trust it with an inheritance, read the code, rehearse a full
> recovery before you rely on it, and keep the manual fallback (below) in reach.
> It is offered under the Apache License 2.0, without warranty of any kind.

## Contents

- [How it works, in a nutshell](#how-it-works-in-a-nutshell): the whole system on one screen
- [Design](#design): what it solves, architecture, distribution, recovery, upkeep
- [Tools](#tools)
  - [Offline tool (`offline/`)](#offline-tool-offline): key file, SLIP-39 shares, runbook
  - [Dead man's switch (`dms/`)](#dead-mans-switch-dms): envelopes, deployment, Signal
- [Running this yourself](#running-this-yourself)
- [Licence](#licence)

## Design

### What it solves

The goal is a system that is:

- **simple for a non-technical heir**,
- **no weaker than what it replaces**,
- **resilient** against attack, loss, disaster and war,
- **fully self-custodial**, with no third parties and no custodians,
- **as independent of any platform as possible**, surviving OS upgrades and
  library churn.

The decisions everything else follows from:

| | |
|---|---|
| Threat model | First priority: **the family gets in**. Theft resistance second. |
| Device | Irrelevant. Only the **seed** (plus passphrase) matters, and it restores on any compatible wallet. |
| Wallet | An existing one. No new wallet is created. One seed, with a passphrase. |
| Heirs | Several trusted people, at least one **technically capable**, the rest need not be. One technical helper is enough. |
| Hardware at the heirs | None, at most a **metal backup** holding words. |
| Third parties, notary | No. |
| Scheme | **K of N**, any threshold and count. Examples below use 2 of 3. |
| Control while alive | The owner controls the BTC through the passphrase. K of N trusted people can open the database even while he lives, but cannot spend without the passphrase. |
| Trigger | Death. The dead man's switch is optional and removable at any time. |
| Geography | Shares spread across several **locations**, so no single event destroys them. |
| Upkeep | Once a year. |
| Other credentials | Part of the system. **One KeePass database** is enough. |
| Gating | **Lenient.** The database is protected by the key file alone; the passphrase (in the envelope) protects only the BTC. |
| Key file | **160-bit**, giving 23 words per SLIP-39 share. |

### Architecture: two factors

- **Factor A, the sealed envelope.** It holds the **wallet passphrase** and lives
  only where the family gets to it after the owner's death: sealed in a **bank
  vault**, and delivered by the **dead man's switch**, encrypted to a technical
  helper. While the owner lives, only he has the passphrase, so only he controls
  the coins. There can be several envelopes: the same one to several people as a
  backup, or different envelopes to different people so that nobody holds
  everything (see [Envelopes](#envelopes-how-many-and-to-whom)).
- **Factor B, the metal shares.** The key file to the KeePass database, split
  with **SLIP-39 (K of N)** into words, stamped into metal, spread across
  locations.

The **KeePass database** holds the seed, every other credential and a written
procedure. It is encrypted, so its **ciphertext (`.kdbx`) can sit in the cloud**
quite safely: without the key file it is ballast.

```
Database opens  =  key file (K of N metal shares)          # -> seed + credentials
Bitcoin         =  seed (from the database) + passphrase   # -> [K of N shares] + [envelope]

Lenient gating: K of N shares open the database, even while the owner lives, but
nobody spends the coins without the passphrase from the envelope. Losing the
envelope costs you the BTC, not the database.
```

#### Why SLIP-39 on the key file and not on the seed

SLIP-39 is Shamir secret sharing plus a checksummed word encoding. It is **not
tied to a wallet seed**: it splits any 160-bit key. So it is applied to a
**random key file** that locks the KeePass database:

1. generate a random 160-bit key `K`, which is **not** a seed,
2. `K` → SLIP-39 split (K of N) → N × 23 words → into metal,
3. `K` is the key file for KeePass,
4. `K` is then erased and lives only as N metal shares.

A useful side effect: those words **are not a seed**. Anyone who finds them and
types them into a wallet gets an empty one. The implementation is verified
against the **official SLIP-39 test vectors**, so there is no homegrown
cryptography on the critical path.

### Distribution

An example for 2 of 3. The number of shares is up to you:

| Location | Factor B (metal share) | Factor A (envelope) | `.kdbx` ciphertext |
|---|---|---|---|
| **location A**, primary heir | share #1 | – | copy |
| **bank vault** | – | **sealed envelope** | copy |
| **location B**, trusted person | share #2 | – | copy |
| **location C**, technical helper | share #3 | (from the switch once triggered, encrypted to them) | copy |
| **Cloud** | – | – | copy (encrypted) |
| **Switch** | – | envelopes encrypted to their recipients | – |

Recovering the Bitcoin needs **K of N shares plus an envelope**, from the vault
or from the switch, which in practice means the primary heir plus one trusted
helper. A single share on its own is worthless, which is what keeps the risk to
each individual holder low.

### Recovery, as the heir experiences it

1. Open the printed **runbook**, copies of which are at the bank and with the
   trusted people.
2. Call the **technical helper**, who walks them through it.
3. Get the **envelope with the passphrase**, from the bank vault or from the
   switch, which has already mailed it.
4. Collect **K shares** from their holders.
5. In the **offline tool**: SLIP-39 words → key file → open the KeePass database
   with a standard app → the seed and every other credential.
6. Restore the wallet from the **seed**, then unlock it with the **passphrase**.
7. Optionally, and recommended, move the coins to a fresh wallet the heir
   controls.

Words on metal are usually stamped as **four-letter abbreviations**, because a
metal backup has no room for more. That is not a problem: the SLIP-39 wordlist is
built so that the first four letters identify a word uniquely (1024 words, 1024
distinct prefixes), and the recovery tool completes them. Three letters would be
ambiguous, so the tool refuses those and names the word to re-read.

The runbook also carries a **manual fallback** that needs none of these tools:
the shares are standard SLIP-39, any SLIP-39 implementation combines them, and
the resulting *master secret* in hex **is** the content of the key file. If such
a tool rejects abbreviations, the full words are in the official wordlist.

### Annual maintenance

- All shares present and legible, confirmed with their holders.
- Envelope in the vault intact; the `.kdbx` copies still open.
- Test the switch's check-in.
- **A full dry-run recovery**, once a year, on a spare machine. At least once
  that should be on Windows, since that binary cannot be tested anywhere else.
- Update the database when credentials change, and reprint the runbook.

### What the system does not solve

If both parents die and the children are minors, the **technical** path is
covered: two trusted people reassemble K of N shares plus the envelope and reach
the coins. What no system can settle is who then holds and manages them until the
children are adults, because whoever reassembles the shares can spend them. That
is a question of trust and inheritance law, not cryptography. If it is to be
settled, it belongs in a will, which must never contain the seed or the
passphrase, only a pointer to the runbook and the share holders, because a will
ends up in a court file.

### Longevity principles

1. **Open standards on the critical path**, which makes the software replaceable:
   SLIP-39 with its test vectors, `.kdbx`, GPG. Recovery is possible with standard
   tools even if this code is gone, and the runbook says how.
2. **Self-contained artifacts.** A static Go binary offline, depending only on the
   kernel ABI and a browser; a Docker image online, with a frozen userland.
3. **Minimal dependencies**, pinned and reproducibly buildable.
4. **Archive alongside the data**: binaries for several operating systems, the
   source, the build recipe, the Docker image as a tarball, and the paper
   fallback.

## Tools

The stack is **Go**: static binaries, the Go 1 compatibility promise, one
language for both halves, no cgo and no GUI toolkit. The `.kdbx` file is neither
created nor read by this code. A standard KeePass application does that, using
the key file this tool produces: KeePassXC, from
<https://keepassxc.org/download/>, or any other implementation of the format
(KeePass, KeePassDX, Strongbox).

### Offline tool (`offline/`)

An air-gapped tool that generates a key file and splits it into **K-of-N SLIP-39**
shares, and reassembles the key file from K shares during recovery. It also
generates the printable runbook.

> **Run it only on an air-gapped machine.** It listens on loopback (`127.0.0.1`)
> and refuses to start on any other address. It makes no network calls at all.
> Close it when you are done.

#### Build

Go is the only requirement (≥ 1.24, for `crypto/pbkdf2`). No external
dependencies.

```
cd offline && go build -o inh-offline .
```

The result is a **single static binary**. Your heir will not necessarily be
sitting at a Linux machine, so the ceremony should produce binaries for every
platform at once:

```
cd offline && ./build-all.sh      # -> dist/ + SHA256SUMS
```

That builds Linux, Windows and macOS (Apple Silicon and Intel), about 34 MB in
total, into a gitignored `dist/`.

The heir downloads them, so after any change to the tool they have to be
published:

```
cd offline && ./release.sh          # tag from today's date
cd offline && ./release.sh v1.0.0   # or an explicit tag
```

That builds and uploads to a GitHub release, which is where the runbook sends
the heir (`/releases/latest`, an address that keeps working as new releases
appear). The binaries are **built on your machine and only uploaded**. There is
no CI build on purpose: a release pipeline would mean trusting somebody else's
runner with the binary that reassembles the key file, and building locally is
what saves you from having to.

Releases rather than committed files, because a 34 MB rebuild in every commit
would end up in the history of everyone who clones this.

The same files also go onto a USB stick beside every metal backup and into the
bank vault, so the inheritance does not depend on GitHub still existing. The
runbook names both paths and tells the heir which file to run on which machine.

Go cross-compiles on its own, with no extra toolchain. Only the binary for the
build machine can be tested there, so **Windows and macOS must be tried on the
real thing**, which belongs in the annual maintenance. The binaries are unsigned,
so Windows SmartScreen and macOS Gatekeeper will both complain; the runbook
explains how to click through.

#### Running it

```
./inh-offline                # http://127.0.0.1:8777
./inh-offline --addr 127.0.0.1:9000
./inh-offline --open=false   # do not launch a browser
```

It opens the system default browser (`xdg-open`, `rundll32`, `open`). At startup
it runs a **power-on self-test**, a known SLIP-39 vector plus a round trip, and
refuses to start if that fails.

- **New backup** generates a 160-bit key file, splits it into N shares of 23
  words with threshold K, shows them for stamping and offers the key file for
  download.
- **Recovery** gives you **one field per word**. The number of shares and the
  words per share (20/23/26/33) are set with spinners that add and remove blocks
  directly. Typing the four stamped letters completes the word, marks the field
  and jumps to the next; pasting a whole share into one field distributes it
  across the following ones; an impossible word turns the field red immediately.
  Case, line numbering and punctuation are ignored. The fields are rendered
  server-side, so the form **works with JavaScript disabled**: the script only
  completes and navigates, while validation and reassembly always happen on the
  server. A paste-everything textarea is still there under a fold.
- **Runbook** takes the who-has-what map and produces a printable set of
  instructions for the family. It contains **no secrets**.

#### How the pieces fit

- Shares are transferred to a **metal backup** that holds 23 words: a plate, a
  cylinder or capsule, a cassette with sliding letter tiles, and so on. One per
  holder or location. Only the **first four letters** of each word are
  transferred, and the setup page highlights them so it is clear what goes onto
  the metal.
- The key file locks the **KeePass database** (in KeePassXC, protection = *Key
  file*; download it from <https://keepassxc.org/download/>). KeePassXC hashes
  any file that is not 32 bytes, 64 hex characters or KeyFile-XML with
  **SHA-256**, deterministically, so a **20-byte** (160-bit) file works and
  reassembles identically. The downloaded key file is **raw
  bytes**, not hex text: during a manual fallback, build it from the hex with
  `xxd -r -p` (or `perl -e 'print pack "H*","…"'`, or
  `python3 -c '…bytes.fromhex(…)'` where `xxd` is missing). Hex saved as text is
  a different file and will not open the database.
- Verified against **KeePassXC 2.7.10** (`keepassxc-cli`): the key file creates
  and reopens a password-less `.kdbx`, a key reassembled from any 2 of 3 shares
  opens the same database, and both a wrong key file and "the hex as text" are
  refused.
- The **wallet passphrase is not in the database.** It lives in the sealed
  envelope, in the vault and with the switch.

#### Correctness

The SLIP-39 core (`internal/slip39`) is verified against **all 45 official
SLIP-39 test vectors** plus round trips for 128, 192 and 256-bit secrets.
`go test ./...`.

### Dead man's switch (`dms/`)

An online service that asks the owner to **check in** periodically, asks
**trusted people** to confirm after a long silence, and after confirmation plus a
grace period mails **GPG-encrypted envelopes**, which it can never read itself,
to their recipients.

**Channels:** e-mail (primary) plus optionally **Signal** (secondary). Every
message goes out on every channel the recipient has an address for.

#### The confirmation flow

1. **Regular check-in** (click "I am alive").
2. Miss one and **escalating reminders** go to the owner on every channel.
3. After a long silence the switch **does not fire on its own**. It asks the
   technical helper and another trusted person to confirm death or permanent
   incapacity, with a link valid only for this cycle and a confirmation button.
4. Once confirmed, the **grace period** starts, during which the owner gets a
   daily warning that the envelope goes out in X days. One confirmation is enough
   by default (`confirm_quorum: 1`): a false alarm is survivable, because the
   envelope is worthless without the metal shares, whereas "nobody confirmed"
   would stall the inheritance. With `confirm_quorum: 2` the switch counts
   **distinct** confirmers, so the same person twice does not count twice.
5. **The owner's check-in cancels everything, at any time**, including during the
   countdown.
6. After the grace period with no veto, the envelopes go out, decryptable only by
   whoever they were encrypted to.
7. A false or malicious confirmation is not a catastrophe: the envelope is
   **useless without enough metal shares**.

Safety nets: a stuck or unconfirmed switch **never blocks the inheritance**,
because the envelope is also in the bank vault. The switch is best-effort and
removable, and the vault is the path that always works.

#### Security model

- **It never holds plaintext.** It holds only envelope ciphertext, encrypted to
  the recipients' GPG keys, and on release it merely sends it. Who can open an
  envelope is decided **offline when you encrypt it**, by which key you encrypt
  to; the config only says where it is sent.
- **Fail-safe:** if the self-test fails (mail unreachable, an envelope missing or
  not PGP, state not writable), the switch **alerts but does not fire**.
- **Two-step links:** check-in and confirm are a GET page plus a POST button, so
  that automatic link prefetching by mail scanners cannot trigger them.
- **Confirmation links are bound to one waiting cycle.** A check-in invalidates
  them, and a link kept from an earlier cycle cannot be replayed in a later one.
  The check-in credential is separately revocable through `checkin_key_version`.
- **Tokens** in the links are HMACs of `hmac_secret`. Even a leaked link fails
  safe: a check-in only delays release, and a confirmation still needs a human,
  the grace period and the metal shares.
- **The secondary channel never blocks release.** A broken Signal is an e-mail
  alert, not a fault; each envelope goes out if **at least one** channel delivers
  it, and whatever fails is retried on the next tick.
- **Plaintext to a local postfix, TLS to anything else.** STARTTLS is
  deliberately skipped on loopback: the bytes never leave the machine, and
  postfix has no certificate for `127.0.0.1` and cannot have one, so Go would
  otherwise refuse to send every message including the envelope. For a
  non-loopback `smtp_addr`, STARTTLS is required and the certificate verified.
- Network work never happens while the state lock is held, so a relay that
  accepts a connection and then stops answering cannot block the owner's veto.
  The whole SMTP conversation is bounded by `smtp_timeout`.

#### Default timeline

```
monthly check-in → after 60 days of silence: ask the confirmers
→ confirmation (any of them) → 7-day delay with daily warnings to the owner
→ release: envelopes mailed to their recipients.   A check-in cancels everything.
switch → owner, weekly "healthy"; on a fault, an alert.
```

#### Envelopes: how many and to whom

There can be several, and **each has its own recipients**:

```json
"envelopes": [
  { "id": "passphrase", "path": "/data/envelope-passphrase.asc",
    "to": [ {"name":"First","email":"first@…","signal":"+…"},
            {"name":"Second","email":"second@…"} ] },
  { "id": "credentials", "path": "/data/envelope-credentials.asc",
    "note": "These are the remaining credentials, not the wallet passphrase.",
    "to": [ {"name":"Third","email":"third@…"} ] }
]
```

That covers two different intentions:

- **the same envelope to several people**, as a backup, so that one unreachable
  person does not cut off the whole path (for instance if you and they die
  together),
- **different envelopes to different people**, so that nobody holds everything.

What the service enforces: an envelope with no recipients, or two sharing an
`id`, is refused at startup; the self-test checks **every** file, and a missing
or non-PGP one is a fault that stops the release **entirely** rather than
delivering half of it; on release it remembers which envelopes went out, so only
the rest is retried and nobody receives the same one twice. A check-in clears
that memory, so a later real release delivers everything again.

The older `envelope_path` plus `friend_email` form still works, as a single
envelope with a single recipient.

> **When splitting the contents, the bank vault must hold everything.** The
> switch is best-effort and the vault is the sure path. Otherwise you have built
> a case where one unresponsive recipient cuts off part of the inheritance.

#### Where the recipient's public GPG key goes

Into `keys/`, which is only a convention; `*.asc` is gitignored so the repository
does not reveal who is involved. The switch never sees that key. You need it only
when you make the envelope:

```
gpg --export --armor <key-id> > keys/friend.asc
```

#### Making the envelopes, offline and by hand

Put only the **wallet passphrase** in the file, plus a short instruction if you
like, and encrypt it to the recipient's public GPG key. Verify the fingerprint
independently, not through the same channel the key arrived on.

```
gpg --import keys/friend.asc
gpg --armor --encrypt --recipient friend@example.com passphrase.txt
mv passphrase.txt.asc envelope.asc      # this goes to /data, never to git
shred -u passphrase.txt
```

If several people should be able to open the same envelope, encrypt it to
several keys at once (`--recipient A --recipient B`). That is independent of who
it gets delivered to.

#### Deployment

On a server with Docker and postfix, `dms/deploy.sh` does it, run from the `dms/`
directory. It deploys nothing behind your back: it prints every step and never
touches Apache or postfix.

**Compose is not required.** If `docker compose` (v2) is available it drives
that; otherwise, and with `--no-compose`, it does the same through plain
`docker build` and `docker run`. The old `docker-compose` v1 is deliberately
ignored, because it is end-of-life and cannot even parse `${VAR:-default}` in a
compose file, so accepting it would break a deployment half way through.

```bash
./deploy.sh --check                              # preflight only, changes nothing
./deploy.sh --envelope ~/envelope.asc            # real deployment (e-mail)
./deploy.sh --envelope passphrase=~/a.asc --envelope credentials=~/b.asc
./deploy.sh --envelope ~/envelope.asc --signal   # + Signal channel
./deploy.sh --config-only --force-config         # rewrite config.json only
./deploy.sh --envelope ~/envelope.asc --no-compose
./deploy.sh --envelope ~/envelope.asc --test-timings   # rehearsal, see below
```

What it does: check docker, compose, postfix and ports → create
`/opt/inh-dms/data` (0700) → copy the envelopes, refusing any that is not
ASCII-armored PGP → ask for addresses, numbers and confirmers and write
`config.json` (0600, with `hmac_secret` from `openssl rand -hex 32`) → **validate
the config through `inh-dms --validate`** → build and start the container → check
HTTP, the log and `state.json` → print the reverse proxy config and **your
check-in link to bookmark**.

It is idempotent: an existing `config.json` or envelope is left alone, so it is
safe to run again. `--force-config` rewrites the config after backing it up.

`--test-timings` is a rehearsal instance, **fully isolated from the real one**:
its own data directory (`/opt/inh-dms-test`), container names (`inh-dms-test`,
`inh-signal-test`), ports (8188 and 8180) and Compose project
(`-p inh-dms-test`). A rehearsal therefore cannot take down or replace a running
production switch, and the proxy will not start pointing at it; if a production
instance is running alongside, the script says so at startup.

On top of that: intervals in minutes instead of days, a clean start (the old
`state.json` is deleted), links pointing at `http://127.0.0.1:8188` over an SSH
tunnel, and **envelopes and confirmation requests all addressed to you**. A
rehearsal must never write to real people, because in the waiting phase the
request repeats every few minutes. The whole chain, check-in through silence,
confirmation, countdown and release, can be walked in a few minutes without
alarming anyone. Afterwards:

```bash
docker compose -p inh-dms-test down     # or: docker rm -f inh-dms-test
rm -rf /opt/inh-dms-test
```

By hand it is the same: `mkdir -p /opt/inh-dms/data`, a `config.json` from
`config.example.json` (`hmac_secret` = `openssl rand -hex 32`), envelopes into
`/opt/inh-dms/data/` (paths in the config are container-side, `/data/…`), then
`INH_UID=$(id -u) INH_GID=$(id -g) docker compose up -d`, or the same with
`--profile signal`. Without compose:

```bash
docker build -t inh-dms .
docker run -d --name inh-dms --restart unless-stopped \
  --user "$(id -u):$(id -g)" --network host \
  -v /opt/inh-dms/data:/data inh-dms
```

`--user` is not optional: the image runs as `nonroot` (uid 65532) but `/data`
belongs to you and `config.json` is 0600, so without it the container dies on
`open /data/config.json: permission denied`. Host networking is deliberate:
`smtp_addr` reaches the local postfix on `127.0.0.1:25`, the Signal API is on
`127.0.0.1:8080`, and `listen_addr` binds to the host loopback where the reverse
proxy points.

The service speaks plain HTTP on loopback, so it needs a reverse proxy with TLS
in front of it.

Apache (`a2enmod proxy proxy_http headers`, TLS through certbot):

```apache
<VirtualHost *:443>
  ServerName dms.example.com
  ProxyPreserveHost On
  ProxyPass        / http://127.0.0.1:8088/
  ProxyPassReverse / http://127.0.0.1:8088/
  RequestHeader set X-Forwarded-Proto https
  # certbot fills in the TLS directives
</VirtualHost>
```

nginx, the same thing:

```nginx
server {
    listen 443 ssl;
    server_name dms.example.com;
    # certbot fills in ssl_certificate / ssl_certificate_key

    location / {
        proxy_pass http://127.0.0.1:8088;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

Verification after startup, which the script also does, but which is worth
knowing:

1. `curl -s https://dms.example.com/` answers that the service is running.
2. Within a minute an **"all good"** e-mail arrives, and a Signal message too if
   that channel is on. That is also proof the self-test passed and every envelope
   is readable.
3. Click the check-in link in it, and `last_check_in` changes in `state.json`.
4. `docker logs inh-dms` contains no `failed`.
5. **Bookmark the check-in link and put it in the KeePass database.** It is
   stable, but it depends on `hmac_secret`, so store that too; changing it makes
   different links.
6. Record in the runbook and in the database that the switch exists and how to
   turn it off (`docker compose down` removes it, and the inheritance does not
   suffer).

Upkeep: `docker compose pull && docker compose up -d` for the Signal container,
and `docker save inh-dms | gzip > inh-dms.tar.gz` into the archive with the other
artifacts.

#### Configuration

See `config.example.json`, or have `deploy.sh` generate one. Durations accept
`30d`, `7d`, `12h`, `90m`. `inh-dms --validate` loads a config, checks it and
exits, which is useful after editing by hand and before restarting.

Required: `public_base_url`, `from_email`, `user_email`, `state_path`,
`hmac_secret` (16 characters or more), at least one `confirmer`, and at least one
envelope (`envelopes[]`, or the older `envelope_path` plus `friend_email`).

Optional, but worth knowing:

| key | what it does |
|---|---|
| `confirm_quorum` | how many **distinct** confirmers start the countdown (default 1) |
| `checkin_key_version` | increment and restart if your check-in link leaks; the old links stop working |
| `smtp_timeout` | bound on the whole SMTP conversation (default 30s), so a stalled relay cannot block the switch |

#### Signal, the second channel

Optional. Leave out the `signal` block and every `*_signal` number and everything
goes by e-mail. The configuration is **all or nothing**: a number without
`signal.api_url`, or the reverse, is a startup error, because a half-configured
channel would be silently dropped, which is exactly what must not happen in a
system nobody looks at for years.

```json
"signal": { "api_url": "http://127.0.0.1:8080", "from_number": "+…" },
"user_signal": "+…", "friend_signal": "+…",
"confirmers": [ { "id": "friend", "…": "…", "signal": "+…" } ]
```

A `bbernhard/signal-cli-rest-api` container holds the linked device, and the
switch only POSTs text to `/v2/send` over loopback. Linking, once, at deployment:

```
docker compose --profile signal up -d
# open over an SSH tunnel and scan the QR code in Signal:
#   Signal → Settings → Linked devices → +
xdg-open http://127.0.0.1:8080/v1/qrcodelink?device_name=inh-dms
curl -s http://127.0.0.1:8080/v1/accounts     # must list your from_number
```

The self-test checks exactly that `/v1/accounts`, because the realistic silent
failure of this channel is an **unlinked device**, not a dead container.
Incoming messages are not processed: the check-in is a link in the message and
works from either channel.

Note that linking makes the container a full Signal device on your number, which
means it receives all your messages, not only the ones the switch sends. That is
the price of messages coming from your own number, which is what makes a
"confirm that he died" request credible to the recipient rather than looking like
a scam.

Envelopes go out over Signal too, since they are ciphertext and the channel does
not matter. If one is too long for a single Signal message, Signal rejects it and
e-mail delivers it, which is why e-mail is primary and an envelope should hold
only the passphrase and a short instruction.

#### Correctness

`go test ./...` covers the whole timeline deterministically, with injected time
and a fake mailer: reminders, the transition into waiting, confirmation through
countdown to release, cancellation by check-in, fail-safe on a fault or a missing
envelope, and the HTTP handlers. For Signal: fan-out to both channels, release
with Signal down, release over Signal with mail down, no channel meaning no
release plus a retry, the HTTP client against a fake API, and config validation.
The suite also runs clean under `-race`.

## Languages

Everything a person reads is in a message catalogue, one file per language, so
adding a language means adding one file and touching no template:

| | |
|---|---|
| `offline/internal/i18n/` | the tool's interface and the runbook |
| `dms/internal/i18n/` | the switch's e-mails and its two web pages |

English is the source of truth and the fallback. A key missing from a
translation renders in English rather than blank, and a key missing everywhere
renders as `[[key.name]]`, so a gap is visible instead of silent. Three tests
enforce that: a translation must cover the English key set, must not invent keys
of its own, and must carry the same format placeholders, which is what stops a
`%!s(MISSING)` reaching a reader.

English, Slovak and Czech are complete. The Czech translation has not been read
by a native speaker, and a runbook is a document someone follows under stress, so
have one check it before printing a Czech copy.

Where the language comes from differs by tool, because the readers do:

- The offline tool takes it from the URL (`?lang=sk`), and the runbook form has
  its own selector, since you may want to print a Slovak copy for one holder and
  an English one for another.
- The switch takes it **per recipient**: `user_lang` for the owner, and `lang` on
  each confirmer and each envelope recipient. One notification is rendered
  separately for each person, so an English confirmer and a Slovak one each get
  their own.

To add a language: copy `en.go` in both packages, translate the values, register
the tag in `i18n.go`, and run the tests.

## Running this yourself

If you fork this for your own inheritance, the one rule that matters:

**Real secrets never go into the repository.** Not the seed words, the wallet
passphrase, the key file or its SLIP-39 shares, the database password, a real
`.kdbx`, or the contents of an envelope. Secrets are born and live offline, on
metal, in a bank vault and in the encrypted database. The `.gitignore` here stops
the usual files from being committed by accident, but do not rely on it: the real
defence is never bringing them near the repository.

**The map does not belong there either.** Who holds which share, where the
envelope is and which machine runs the switch belong in the **printed runbook**
and the **KeePass database**, not in text in a repository, private or otherwise.
The documentation here is deliberately generic (location A/B/C, "technical
helper") for that reason, and `keys/*.asc` and `*.pdf` are gitignored so the
repository never reveals who is involved.

## Licence

Apache License 2.0. See [LICENSE](LICENSE).

Copyright 2026 The inh authors.
