package i18n

// enMessages is the source of truth. Every other catalogue is a translation of
// these keys, and anything missing elsewhere falls back to this.
var enMessages = map[string]string{
	// --- shared chrome ---
	"nav.home":   "home",
	"nav.back":   "back",
	"nav.again":  "try again",
	"lang.label": "Language",
	"app.title":  "inh: offline tool",

	// --- index ---
	"index.h1":        "🔐 inh: offline setup and recovery",
	"index.setup.h":   "New backup",
	"index.setup.p":   "Generate a key file and split it into shares.",
	"index.recover.h": "Recovery",
	"index.recover.p": "Reassemble the key file from its shares to open the KeePass database.",
	"index.runbook.h": "Runbook",
	"index.runbook.p": "Produce the personalised who-has-what instructions for the family (print or save as PDF).",

	// --- error ---
	"error.title": "Error",

	// --- setup form ---
	"setup.title":     "New backup",
	"setup.intro":     "A random 160-bit key file is generated and split into shares. The usual choice is <strong>2 of 3</strong>.",
	"setup.threshold": "Threshold: how many shares are needed to recover",
	"setup.count":     "Number of shares",
	"setup.submit":    "Generate",

	// --- setup result ---
	"setupres.title":        "Backup generated (%s of %s)",
	"setupres.selftest.ok":  "✓ Self-test: %s shares correctly reassembled the key.",
	"setupres.selftest.bad": "✗ Self-test FAILED. Do not use this output, generate again.",
	"setupres.shares.h":     "Shares: each onto its own medium",
	"setupres.shares.p":     "Each share is %s numbered words; transfer them to the medium in exactly this order. Any %s of them are enough to recover.",
	"setupres.prefix.p":     "Only the <strong>first 4 letters</strong> (highlighted) go onto the metal: in the SLIP-39 wordlist they identify a word uniquely, which is why metal media usually have room for four. The recovery tool completes the rest.",
	"setupres.keyfile.h":    "Key file",
	"setupres.keyfile.p":    "This file locks the KeePass database. Delete it once the database exists; you can always reassemble it from %s shares.",
	"setupres.download":     "⬇ Download key file",
	"setupres.hex":          "hex",
	"setupres.hex.warn":     "⚠️ This is the content of the key file (the same 20 bytes), only for <strong>creating the database right now</strong> and for checking. <strong>Do not write it into the runbook or anywhere else</strong>: the key should live only as %s shares, and storing it anywhere defeats the %s-of-%s protection.",
	"setupres.next.h":       "Next",
	"setupres.next.1":       "Transfer every share onto a metal medium and hand them out according to the plan.",
	"setupres.next.2":       "In <strong>KeePassXC</strong> (<code>https://keepassxc.org/download</code>) create a new database and choose <em>Key file</em> as the protection, pointing at the downloaded file (<strong>leave the password field empty</strong>).",
	"setupres.next.3":       "Store the seed and the other credentials in the database. <strong>Do not put the wallet passphrase in it</strong>; that goes into the sealed envelope.",
	"setupres.next.4":       "Copy the encrypted <code>.kdbx</code> around (cloud, bank, the holders). Delete the key file from disk.",

	// --- recovery form ---
	"recover.title":         "Recover the key file",
	"recover.intro":         "Copy the words from the metal shares, <strong>each share into its own block</strong>. The <strong>first 4 letters</strong> are enough, exactly as stamped; the word completes itself and the cursor jumps on. You need at least as many shares as the threshold was (2 by default).",
	"recover.partcount":     "Number of shares",
	"recover.wordcount":     "Words per share",
	"recover.part":          "Share %s",
	"recover.submit":        "Reassemble the key file",
	"recover.paste.sum":     "Or paste the shares as text",
	"recover.paste.p":       "One share per line. Use this if you have the words in a file; otherwise fill in the fields above.",
	"recover.paste.ph":      "share 1: word word word …\nshare 2: word word word …",
	"recover.confirm.words": "Changing the length clears the words you have entered. Continue?",
	"recover.confirm.parts": "Removing these will clear the words entered in them. Continue?",

	// --- recovery result ---
	"recoverres.title": "Key recovered",
	"recoverres.ok":    "✓ Key file reassembled from %s shares.",
	"recoverres.step":  "Note: this is <strong>step 1 of 3</strong>. The key file alone is not access to the Bitcoin: you still need the <code>.kdbx</code> database and the passphrase from the envelope.",
	"recoverres.chain": `①  shares
       │   the tool reassembles them
       ▼
   key file: a small file on the computer      ← THIS is what you just got
       │   it opens the password database (KeePassXC)
       ▼
②  in the database: the wallet SEED + other credentials
       │   plus the passphrase from the envelope
       ▼
③  restore the wallet  →  Bitcoin ₿`,
	"recoverres.next.h": "Next",
	"recoverres.next.1": "Open your <code>.kdbx</code> in <strong>KeePassXC</strong> (download it on a machine with internet from <code>https://keepassxc.org/download</code>) and choose <em>Key file</em> as the protection, pointing at this file (<strong>leave the password field empty</strong>).",
	"recoverres.next.2": "You reach the <strong>seed</strong> and the other credentials. Take the <strong>wallet passphrase</strong> from the envelope (bank or e-mail).",
	"recoverres.next.3": "On the hardware wallet, restore from the seed, then unlock with the passphrase.",

	"rbform.title":        "Runbook: instructions for the family",
	"rbform.intro":        "Fill this in and it produces a <strong>printable who-has-what guide</strong>. The document <strong>contains no secrets</strong>, only the map and the procedure. Keep it with trusted people all the same, since it does reveal where to look.",
	"rbform.lang":         "Runbook language",
	"rbform.basics":       "Basics",
	"rbform.author":       "Your name",
	"rbform.date":         "Date",
	"rbform.heir":         "Name of the primary heir (who the letter is addressed to)",
	"rbform.scheme":       "Scheme",
	"rbform.threshold":    "Threshold (how many shares are needed to recover)",
	"rbform.count":        "Number of shares",
	"rbform.holders":      "Shares and who holds them",
	"rbform.holders.p":    "The number of rows follows <strong>Number of shares</strong> above. For each share give <strong>who holds it</strong> and a contact, and whether they are <strong>technically capable</strong> (called for help, receives the envelope) or not.",
	"rbform.ph.name":      "who holds this share: name",
	"rbform.ph.contact":   "contact (phone or e-mail)",
	"rbform.nontech":      "not technical",
	"rbform.tech":         "technically capable",
	"rbform.envelope":     "Envelope and copies",
	"rbform.bank":         "Envelope (passphrase), bank",
	"rbform.ph.bank":      "bank name, branch, box number",
	"rbform.toolurl":      "Where the tool can be downloaded (repository URL)",
	"rbform.toolwhere":    "Where the <em>inh-offline</em> tool is kept",
	"rbform.ph.toolwhere": "e.g. a USB stick beside every metal backup, at the bank, and in the cloud next to the .kdbx",
	"rbform.kdbx":         "Copies of the encrypted .kdbx, where",
	"rbform.ph.kdbx":      "e.g. cloud, at home, bank vault, with the holders",
	"rbform.extras":       "Extras",
	"rbform.wallet":       "Wallet, notes",
	"rbform.message":      "A message for the family (free text)",
	"rbform.submit":       "Generate the runbook",
	"rbform.rownum":       "Share %s:",

	"rb.aside": " (%s)",
	"names.or": " or ",

	// --- holder labels on the setup page ---
	"holder.1": "First share",
	"holder.2": "Second share",
	"holder.3": "Third share",
	"holder.n": "Share #%d",

	// --- wording for a word that could not be resolved ---
	"word.empty":   "the line contains no words",
	"word.at":      "word %d:",
	"word.at.part": "share %d, word %d:",
	"word.short":   "%q is too short, the metal always carries at least %d letters",
	"word.unknown": "%q is not in the SLIP-39 wordlist",
	"word.typo":    "%q is not in the SLIP-39 wordlist (by its first %d letters it should be %q)",

	// --- errors from the server ---
	"err.rand":     "The random number generator failed: %s",
	"err.params":   "Invalid parameters: %s",
	"err.noshares": "You did not enter a share.",
	"err.words":    "I do not understand the words entered: %s. Write them as they are on the metal (the first 4 letters are enough), each share on its own line.",
	"err.combine":  "Recovery failed: %s. Check the words and that you have enough shares.",
	"err.form":     "Invalid form: %s",
}
