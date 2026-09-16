# coldwill: Bitcoin Inheritance System

A Bitcoin inheritance system your family can actually use: simple enough for a
non-technical heir, resistant to theft, loss and disaster, and fully
self-custodial with no third parties, custodians or notaries. The repository
holds the design and two tools, an offline key generator and recovery tool that
also prints a runbook for the family, and an online dead man's switch (DMS).

*Slovensky: [README_sk.md](README_sk.md).*

## How it works, in a nutshell

**While you are alive**, only you control the Bitcoin. **After you die**, your
family reassembles it from two independent things. Neither is enough on its own.

How many shares exist, and how many are needed to reassemble the key (**K of
N**), is chosen when you generate them. The drawing shows a common choice, 2 of 3.

```
    SHARE #1         SHARE #2         SHARE #3              ENVELOPE
  metal, person A  metal, person B  metal, person C  vault text + PDF from DMS
       │                │                │                      │
       └────────┬───────┴────────────────┘                      │
                │   any K of N (here 2 of 3)                    │
                ▼                                               │
            KEY FILE                                            │
                │                                               │
                ▼   opens (no password)                         ▼
         KeePass database  ──►  SEED + credentials + MAP ──► PASSPHRASE
                                      │                         │
                                      └──────────┬──────────────┘
                                                 ▼
                                            ₿  BITCOIN
```

- **Metal shares.** SLIP-39 words stamped into metal, held by different people
  in different places. Any **K of N** of them reassemble a *key file*, which
  unlocks a KeePass database holding the wallet **seed** and every other
  credential. Threshold and count are yours to pick: 3 of 5, 2 of 4, whatever
  fits the people you trust.
- **The envelope** is an ordinary printed text with the **wallet passphrase**
  hidden in it. The **map** that reads it out is kept in the database, so the
  text on its own gives nothing away. It lies in a bank vault, and after your
  death the *dead man's switch* also sends it to the primary heir as a PDF.
- **Bitcoin = seed + passphrase.** Whoever holds only shares can read the
  database and the map, but has no text to read. Whoever holds only the envelope
  has an article and nothing to read it with. Fewer than K shares are worthless.

The **dead man's switch** is a convenience, not a dependency. It asks you
periodically whether you are alive, and once you stop answering and someone you
trust confirms, it sends the envelope to the primary heir after a grace period. How many
confirmations it takes is configurable and defaults to one. The bank vault is
the path that always works: the DMS can be turned off at any time without
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
  - [Dead man's switch (`dms/`)](#dead-mans-switch-dms): the envelope, deployment, Signal
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
| Gating | **Lenient.** The database is protected by the key file alone; the passphrase (hidden in the envelope) protects only the BTC. |
| Key file | **160-bit**, giving 23 words per SLIP-39 share. |

### Architecture: two factors

- **Factor A, the envelope.** An ordinary printed text with the **wallet
  passphrase** hidden in it, readable only with the map kept in the database. It
  lives only where the family gets to it after the owner's death: sealed in a
  **bank vault**, and sent by the **DMS** to the primary heir as a
  PDF. Nothing is encrypted, because the text without the database is worthless.
  While the owner lives, only he has the passphrase, so only he controls the
  coins.
- **Factor B, the metal shares.** The key file to the KeePass database, split
  with **SLIP-39 (K of N)** into words, stamped into metal, spread across
  locations.

The **KeePass database** holds the seed, every other credential, the map for
the envelope and a written procedure. It stays with the **primary heir**, and optionally one more sits in the
bank vault. It deliberately does not go to the share holders or into the cloud.
Holders who neither have the database nor know where it is gain nothing by
reassembling the key file, even if enough of them get together. And a database
kept by every holder would mean visiting all of them whenever a credential
changes.

```
Database opens  =  key file (K of N metal shares)          # -> seed + credentials + map
Bitcoin         =  seed (from the database) + passphrase   # -> [K of N shares] + [envelope]
Passphrase      =  the envelope, read with the map          # the text alone says nothing

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

| Location | Factor B (metal share) | Factor A (envelope) | Database (`.kdbx`) |
|---|---|---|---|
| **location A**, primary heir | share #1 | (PDF from the DMS once triggered) | ✓ |
| **bank vault** | optionally one of the shares | **envelope** (the printed text) | ✓ (optional) |
| **location B**, trusted person | share #2 | – | – |
| **location C**, technical helper | share #3 | – | – |
| **DMS** | – | the envelope as a PDF, for the primary heir | – |

Recovering the Bitcoin needs **K of N shares plus an envelope**, from the vault
or from the DMS, which in practice means the primary heir plus one trusted
helper. A single share on its own is worthless, which is what keeps the risk to
each individual holder low.

Both vault options are checkboxes on the runbook form, so the printed map shows
whichever you chose. Each makes the vault a bigger prize: with both, whoever gets
into the box has a share, the envelope and the database, and K−1 more shares open
the database, whose map reads the passphrase out of the envelope.

#### The envelope

The passphrase is written down nowhere. The envelope is an ordinary text, such
as a newspaper article, with the passphrase hidden in its characters, and the
map that reads it out is kept in the KeePass database
([how it works](#hiding-the-passphrase-in-a-text)). Without the database the
text is worthless, which is what lets it lie in the vault as a plain printout
and travel by e-mail and Signal as a plain PDF: there are no keys to make, keep
or lose, and nothing for the heir to decrypt.

The same text exists twice, printed in the vault and as a PDF in the DMS,
which sends it to the primary heir and nobody else. If you and the heir die
together, that copy lands in a mailbox nobody reads, and the others use the
printout in the vault.

#### Hiding the passphrase in a text

Each passphrase character comes from any position in a word of an ordinary
text, such as a newspaper article, or from the space and marks after the word.
The vault holds the printed text, the DMS holds the same text as a PDF,
and the map of positions is kept in the database.

The offline tool **Passphrase in a text** picks positions at random, reads the
map back and checks that it matches the passphrase exactly. It gives you the
map and a PDF of the text. Repeated characters use different positions while
any remain available. Every map row has the same format, so it does not label
which characters are uppercase, punctuation or spaces.

**The passphrase.** All printable ASCII characters (0x20–0x7E) are allowed:
lowercase and uppercase letters, digits 0–9, punctuation and spaces. Accented
letters and € are not allowed. Every space counts, including leading and
trailing spaces. Avoid `\ | ~ ^ { } [ ] < >` and backticks: they are hard to
find in an ordinary article.

**The text.** Use an article, a book page or your own text. Do not name the
source next to the map (paper, date, headline), which could let someone find
the text without visiting the vault. Every passphrase character needs a usable
position: “1874” provides all four digits, while sentence beginnings and
abbreviations provide capitals. A contact paragraph can naturally supply an
email (`@ _ .`), website (`/ : ? = &`), phone (`+`), hashtag (`#`) or price (`$`).
When changing the passphrase, the paper can stay if it has usable positions for
all the new characters; always create a new map.

The tool converts typographic quotes „ “ ” to `"`, ‚ ‘ ’ to `'`, dashes – — to
`-` and … to `...` throughout the text and headline. It selects only ASCII
characters and never counts through “ch”, “dz” or “dž”, regardless of case.
Spaces are selected only between words on the same PDF line, never at a line
or paragraph end, or inside “4 300”, which uses a nonbreaking space.
**Print the supplied PDF**: its line breaks are part of the map.

**Rules for reading the map:**

- A **paragraph** is a block of text between empty lines; the headline does
  not count. When pasted text has no empty lines, the tool makes each line a
  separate paragraph.
- Count **words** from 1 in each paragraph, including short words and numbers.
  A standalone dash or another mark is not a word; “4 300” is one word.
- Count **characters** from 1 at the start of the word, including punctuation,
  quotes and the space inside “4 300”; “ch”, “dz” and “dž” each count as two
  characters.
- The position immediately after the last character is the **space** after
  the word. If standalone marks follow, continue counting their characters
  and the spaces between them on the same printed line.
- Copy each character **exactly as printed**, preserving uppercase and spaces,
  and join them in map order.

For example, for this first paragraph:

> The council approved 86 thousand euros to repair the old mill, which dates
> from 1874. Work starts soon.

use this map:

```
passphrase character 1:  paragraph 1, word 16, character 1 of the word
passphrase character 2:  paragraph 1, word 15, character 3 of the word
passphrase character 3:  paragraph 1, word 11, character 5 of the word
passphrase character 4:  paragraph 1, word 5, character 9 of the word
passphrase character 5:  paragraph 1, word 15, character 5 of the word
```

This gives `W7, .`: `W` from “Work”, `7` from “1874.”, the comma from “mill,”,
the space after “thousand”, and the period from “1874.”. The space is valid
only if “thousand” and the following word are on the same PDF line. The actual
map contains only positions and counting rules.

**The map is kept in the database**, which is what makes the text worthless on
its own: whoever looks into the box, or reads the heir's e-mail, sees an
ordinary article. Whoever opens the database has the map too, so with a share
and the database in the vault, the box is still K−1 shares from the coins.

A runbook copy in the same box says the envelope is there, so the article hides
from a glance, not from someone who reads the runbook. What protects it is the
database.

### Recovery, as the heir experiences it

1. Open the printed **runbook**, copies of which are at the bank and with the
   trusted people.
2. Call the **technical helper**, who walks them through it.
3. Get the **envelope**: the printed text from the bank vault, or the PDF the
   DMS has already sent to the primary heir.
4. Collect **K shares** from their holders, one of which may be in the bank vault.
5. In the **offline tool**: SLIP-39 words → key file → open the KeePass database
   (the heir's own, or the one in the vault) with a standard app → the seed and
   every other credential.
6. Read the **passphrase** out of the envelope with the map from the database.
7. Restore the wallet from the **seed**, then unlock it with the **passphrase**.
8. Optionally, and recommended, move the coins to a fresh wallet the heir
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
- Envelope in the vault intact, and the database opens: the heir's, and the one
  in the vault if there is one.
- Test the DMS check-in.
- The PDF in the DMS still matches the printout in the vault.
- **A full dry-run recovery**, once a year, on a spare machine. At least once
  that should be on Windows, since that binary cannot be tested anywhere else.
- Update the database when credentials change, replace the one in the vault if
  there is one, and reprint the runbook.

### What the system does not solve

If both parents die and the children are minors, the **technical** path is
covered: two trusted people reassemble K of N shares plus the envelope from the vault and reach
the coins. What no system can settle is who then holds and manages them until the
children are adults, because whoever reassembles the shares can spend them. That
is a question of trust and inheritance law, not cryptography. If it is to be
settled, it belongs in a will, which must never contain the seed or the
passphrase, only a pointer to the runbook and the share holders, because a will
ends up in a court file.

### Longevity principles

1. **Open standards on the critical path**, which makes the software replaceable:
   SLIP-39 with its test vectors, `.kdbx`, a printed text. Recovery is possible with standard
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
dependencies. The font for the printable text, Liberation Serif under the SIL
Open Font License, is embedded in the binary, with its licence next to it in
`internal/pdf/`.

```
cd offline && go build -o coldwill .
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

The runbook tells the heir which file to run on which machine. If you keep
another copy of the files somewhere, the runbook form has a field for it, and the
printed runbook points there in case the download address stops working.

Go cross-compiles on its own, with no extra toolchain. Only the binary for the
build machine can be tested there, so **Windows and macOS must be tried on the
real thing**, which belongs in the annual maintenance. The binaries are unsigned,
so Windows SmartScreen and macOS Gatekeeper will both complain; the runbook
explains how to click through.

#### Running it

```
./coldwill                # http://127.0.0.1:8777
./coldwill --addr 127.0.0.1:9000
./coldwill --open=false   # do not launch a browser
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
- **Passphrase in a text** takes a text you paste and the wallet passphrase, and
  produces character positions (paragraph, word, character) for the database and a PDF to
  print (see [Hiding the passphrase in a text](#hiding-the-passphrase-in-a-text)).
  No page shows the passphrase again, and the PDF carries no metadata.

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
- The **wallet passphrase is not in the database**, or written down anywhere:
  the envelope (printed in the vault, a PDF in the DMS) holds the text, and
  the database holds the map that reads it.

#### Correctness

The SLIP-39 core (`internal/slip39`) is verified against **all 45 official
SLIP-39 test vectors** plus round trips for 128, 192 and 256-bit secrets.
`go test ./...`.

### Dead man's switch (`dms/`)

An online service that asks the owner to **check in** periodically, asks
**trusted people** to confirm after a long silence, and after confirmation plus a
grace period sends **the envelope**, a PDF that is worthless without the
database, to the **primary heir**.

**Channels:** e-mail (primary) plus optionally **Signal** (secondary). Every
message goes out on every channel the recipient has an address for.

#### The confirmation flow

1. **Regular check-in** (click "I am alive").
2. Miss one and **escalating reminders** go to the owner on every channel.
3. After a long silence the DMS **does not fire on its own**. It asks the
   technical helper and another trusted person to confirm death or permanent
   incapacity, with a link valid only for this cycle and a confirmation button.
4. Once confirmed, the **grace period** starts, during which the owner gets a
   daily warning that the envelope goes out in X days. One confirmation is enough
   by default (`confirm_quorum: 1`): a false alarm is survivable, because the
   envelope is worthless without the metal shares, whereas "nobody confirmed"
   would stall the inheritance. With `confirm_quorum: 2` the DMS counts
   **distinct** confirmers, so the same person twice does not count twice.
5. **The owner's check-in cancels everything, at any time**, including during the
   countdown.
6. After the grace period with no veto, the envelope goes out to the primary
   heir, and to nobody else.
7. A false or malicious confirmation is not a catastrophe: the envelope is
   **useless without enough metal shares**.

Safety nets: a stuck or unconfirmed DMS **never blocks the inheritance**,
because the envelope is also in the bank vault. The DMS is best-effort and
removable, and the vault is the path that always works.

#### Security model

- **It holds nothing usable on its own.** The envelope is a text that says
  nothing without the database, so a compromised server, a read mailbox or a
  wrong Signal recipient gives away an article, not the passphrase. That is why
  it is sent unencrypted, with no keys to manage.
- **Fail-safe:** if the self-test fails (mail unreachable, the envelope missing or
  not a PDF, state not writable), the DMS **alerts but does not fire**.
- **Two-step links:** check-in and confirm are a GET page plus a POST button, so
  that automatic link prefetching by mail scanners cannot trigger them.
- **Confirmation links are bound to one waiting cycle.** A check-in invalidates
  them, and a link kept from an earlier cycle cannot be replayed in a later one.
  The check-in credential is separately revocable through `checkin_key_version`.
- **Tokens** in the links are HMACs of `hmac_secret`. Even a leaked link fails
  safe: a check-in only delays release, and a confirmation still needs a human,
  the grace period and the metal shares.
- **The secondary channel never blocks release.** A broken Signal is an e-mail
  alert, not a fault; the envelope goes out if **at least one** channel delivers
  it, and if none does, it is retried on the next tick.
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
→ release: the envelope sent to the primary heir.  A check-in cancels everything.
DMS → owner, weekly "healthy"; on a fault, an alert.
```

#### Who gets the envelope

The primary heir, and nobody else:

```json
"heir": { "name": "…", "email": "…", "signal": "+…", "lang": "sk" },
"envelope_path": "/data/envelope.pdf"
```

It goes out as an attachment on every channel the heir has an address for, and
one channel getting through is enough. If none does, the DMS stays in the
countdown and tries again on the next tick. The self-test reads the file on
every tick, and a missing file or one that is not a PDF is a fault that stops
the release, so the wrong file (the map, say) is caught while you can still
replace it rather than on the day it is sent.

The envelope is the PDF from **Passphrase in a text** in the offline tool, the
same one you print for the vault. It goes to `/data` on the server and never
into git. When you change the text or the passphrase, replace the printout, the
PDF on the server and the map in the database together.

A config that still has `envelopes`, `friend_email` or `friend_signal` in it is
refused at startup, rather than started with those recipients silently dropped.

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
./deploy.sh --envelope ~/envelope.pdf            # real deployment (e-mail)
./deploy.sh --envelope ~/envelope.pdf --signal   # + Signal channel
./deploy.sh --config-only --force-config         # rewrite config.json only
./deploy.sh --envelope ~/envelope.pdf --no-compose
./deploy.sh --envelope ~/envelope.pdf --test-timings   # rehearsal, see below
```

What it does: check docker, compose, postfix and ports → create
`/opt/coldwill-switch/data` (0700) → copy the envelope, refusing a file that is
not a PDF → ask for your address, the primary heir's, numbers and confirmers and write
`config.json` (0600, with `hmac_secret` from `openssl rand -hex 32`) → **validate
the config through `coldwill-switch --validate`** → build and start the container → check
HTTP, the log and `state.json` → print the reverse proxy config and **your
check-in link to bookmark**.

It is idempotent: an existing `config.json` or envelope is left alone, so it is
safe to run again. `--force-config` rewrites the config after backing it up.

`--test-timings` is a rehearsal instance, **fully isolated from the real one**:
its own data directory (`/opt/coldwill-switch-test`), container names (`coldwill-switch-test`,
`coldwill-signal-test`), ports (8188 and 8180) and Compose project
(`-p coldwill-switch-test`). A rehearsal therefore cannot take down or replace a running
production DMS, and the proxy will not start pointing at it; if a production
instance is running alongside, the script says so at startup.

On top of that: intervals in minutes instead of days, a clean start (the old
`state.json` is deleted), links pointing at `http://127.0.0.1:8188` over an SSH
tunnel, and **the envelope and confirmation requests all addressed to you**. A
rehearsal must never write to real people, because in the waiting phase the
request repeats every few minutes. The whole chain, check-in through silence,
confirmation, countdown and release, can be walked in a few minutes without
alarming anyone. Afterwards:

```bash
docker compose -p coldwill-switch-test down     # or: docker rm -f coldwill-switch-test
rm -rf /opt/coldwill-switch-test
```

By hand it is the same: `mkdir -p /opt/coldwill-switch/data`, a `config.json` from
`config.example.json` (`hmac_secret` = `openssl rand -hex 32`), the envelope into
`/opt/coldwill-switch/data/` (paths in the config are container-side, `/data/…`), then
`COLDWILL_UID=$(id -u) COLDWILL_GID=$(id -g) docker compose up -d`, or the same with
`--profile signal`. Without compose:

```bash
docker build -t coldwill-switch .
docker run -d --name coldwill-switch --restart unless-stopped \
  --user "$(id -u):$(id -g)" --network host \
  -v /opt/coldwill-switch/data:/data coldwill-switch
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
   that channel is on. That is also proof the self-test passed and the envelope
   is in place.
3. Click the check-in link in it, and `last_check_in` changes in `state.json`.
4. `docker logs coldwill-switch` contains no `failed`.
5. **Bookmark the check-in link and put it in the KeePass database.** It is
   stable, but it depends on `hmac_secret`, so store that too; changing it makes
   different links.
6. Record in the runbook and in the database that the DMS exists and how to
   turn it off (`docker compose down` removes it, and the inheritance does not
   suffer).

Upkeep: `docker compose pull && docker compose up -d` for the Signal container,
and `docker save coldwill-switch | gzip > coldwill-switch.tar.gz` into the archive with the other
artifacts.

#### Configuration

See `config.example.json`, or have `deploy.sh` generate one. Durations accept
`30d`, `7d`, `12h`, `90m`. `coldwill-switch --validate` loads a config, checks it and
exits, which is useful after editing by hand and before restarting.

Required: `public_base_url`, `from_email`, `user_email`, `state_path`,
`hmac_secret` (16 characters or more), at least one `confirmer`, the `heir` (with
an e-mail or a Signal number) and `envelope_path`.

Optional, but worth knowing:

| key | what it does |
|---|---|
| `confirm_quorum` | how many **distinct** confirmers start the countdown (default 1) |
| `checkin_key_version` | increment and restart if your check-in link leaks; the old links stop working |
| `smtp_timeout` | bound on the whole SMTP conversation (default 30s), so a stalled relay cannot block the DMS |

#### Signal, the second channel

Optional. Leave out the `signal` block and every `*_signal` number and everything
goes by e-mail. The configuration is **all or nothing**: a number without
`signal.api_url`, or the reverse, is a startup error, because a half-configured
channel would be silently dropped, which is exactly what must not happen in a
system nobody looks at for years.

```json
"signal": { "api_url": "http://127.0.0.1:8080", "from_number": "+…" },
"user_signal": "+…", "heir": { "…": "…", "signal": "+…" },
"confirmers": [ { "id": "friend", "…": "…", "signal": "+…" } ]
```

A `bbernhard/signal-cli-rest-api` container holds the linked device, and the
DMS only POSTs to `/v2/send` over loopback: text, and the envelope as an
attachment. Linking, once, at deployment:

```
docker compose --profile signal up -d
# open over an SSH tunnel and scan the QR code in Signal:
#   Signal → Settings → Linked devices → +
xdg-open http://127.0.0.1:8080/v1/qrcodelink?device_name=coldwill-switch
curl -s http://127.0.0.1:8080/v1/accounts     # must list your from_number
```

The self-test checks exactly that `/v1/accounts`, because the realistic silent
failure of this channel is an **unlinked device**, not a dead container.
Incoming messages are not processed: the check-in is a link in the message and
works from either channel.

Note that linking makes the container a full Signal device on your number, which
means it receives all your messages, not only the ones the DMS sends. That is
the price of messages coming from your own number, which is what makes a
"confirm that he died" request credible to the recipient rather than looking like
a scam.

The envelope goes out over Signal too, as a PDF attached to the message. The
channel does not matter, since the text is worthless without the database.

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
| `dms/internal/i18n/` | the DMS e-mails and its two web pages |

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
- The DMS takes it **per recipient**: `user_lang` for the owner, and `lang` on
  each confirmer and on the heir. One notification is rendered
  separately for each person, so an English confirmer and a Slovak one each get
  their own.

To add a language: copy `en.go` in both packages, translate the values, register
the tag in `i18n.go`, and run the tests.

## Running this yourself

If you fork this for your own inheritance, the one rule that matters:

**Real secrets never go into the repository.** Not the seed words, the wallet
passphrase, the key file or its SLIP-39 shares, the database password, a real
`.kdbx`, or the envelope and its map. Secrets are born and live offline, on
metal, in a bank vault and in the encrypted database. The `.gitignore` here stops
the usual files from being committed by accident, but do not rely on it: the real
defence is never bringing them near the repository.

**The map does not belong there either.** Who holds which share, where the
envelope is and which machine runs the DMS belong in the **printed runbook**
and the **KeePass database**, not in text in a repository, private or otherwise.
The documentation here is deliberately generic (location A/B/C, "technical
helper") for that reason, and `*.pdf` is gitignored so neither the envelope nor
a printed runbook ends up in it.

## Licence

Apache License 2.0. See [LICENSE](LICENSE).

Copyright 2026 The coldwill authors.
