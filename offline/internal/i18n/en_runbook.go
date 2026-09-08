package i18n

// Runbook messages, kept apart from the UI strings because this is a document
// rather than an interface: the values are whole paragraphs, so a translator
// can read them as prose instead of reassembling a page from fragments.
func init() {
	for k, v := range map[string]string{
		"rb.title":     "Runbook: instructions for the family",
		"rb.h1":        "Bitcoin: what to do if I die",
		"rb.author":    "Written by: <strong>%s</strong>. ",
		"rb.date":      "Date: %s.",
		"rb.nosecrets": "This document <strong>contains no secrets</strong> (no seed, password, passphrase or shares), it is only a <strong>map and a procedure</strong>. It does reveal where to look, so do not publish it and keep it only with people you trust.",

		"rb.intro.h":      "Introduction",
		"rb.intro.to":     "For: <strong>%s</strong>",
		"rb.intro.p":      "Call the technically capable person listed below first; they will walk you through the whole thing. Take your time, do it step by step, and give nothing to anyone you do not know.",
		"rb.intro.whatis": "<strong>What a „share“ is:</strong> a list of <strong>23 words</strong> kept on a <strong>metal medium</strong>, so that it survives fire and water. The shape differs from holder to holder: it may be a flat plate, a metal cylinder or capsule, a cassette with sliding letter tiles, a card the size of a bank card, or several strips screwed together. Look for a metal object with words or letters on it. One share on its own is useless; you need <strong>%s of %s</strong> of them to reassemble the „key file“ (the key to the database). The shares are held by different trusted people, listed below.",

		"rb.auto.h": "How the automatic system works",
		"rb.auto.p": "While I am alive I get regular check-in e-mails. If I go quiet for a long time and one of the trusted people confirms it, then after a grace period of a few days the system <strong>automatically mails the envelope with the passphrase to a technically capable person%s</strong>. If that automatic delivery ever fails, the same password is written on paper in the bank vault (section 2).",

		"rb.call.h":      "1) Who to call",
		"rb.call.tech":   "<strong>Technically capable people.</strong> Call them, they will guide you through it:",
		"rb.call.notech": "No technically capable person is listed. Add at least one, or there will be nobody to help.",
		"rb.call.others": "<strong>The other share holders:</strong>",

		"rb.need.h":     "2) What you need to obtain (two things)",
		"rb.need.p":     "Reaching the Bitcoin needs <strong>both</strong>:",
		"rb.need.parts": "<strong>%s of %s shares</strong> (each is 23 words), which reassemble the „key file“.",
		"rb.need.env":   "<strong>The envelope with the passphrase</strong>: the second secret, without which the Bitcoin cannot be reached.",
		"rb.need.note":  "Neither is enough on its own, which is what makes this safe.",

		"rb.map.h":     "3) Map: where everything is",
		"rb.map.item":  "Item",
		"rb.map.where": "Where / with whom",
		"rb.map.part":  "Share %s",
		"rb.map.held":  "held by: %s",
		"rb.map.env":   "Envelope (passphrase), bank",
		"rb.map.kdbx":  "Encrypted database (.kdbx), copies",

		"rb.steps.h":     "4) Step by step",
		"rb.steps.intro": "How the pieces fit together (you need <strong>three things</strong>, not one):",
		"rb.steps.chain": `①  %s of %s shares (words from metal)
       │   the tool reassembles them into a key
       ▼
   the key: a small file on the computer
       │   it opens the password database (the KeePassXC program)
       ▼
②  in the database: the wallet SEED + the other credentials
       │   plus the password (passphrase) from the envelope
       ▼
③  restore the wallet  →  Bitcoin ₿`,

		"rb.step.call":    "Call the technically capable person (section 1). They will help with anything you do not understand.",
		"rb.step.parts":   "Obtain <strong>at least %s shares</strong> (section 3). Copy the words down from the holders on paper, or have them read out to you. <strong>Do not photograph them with your phone</strong>, it is not safe.",
		"rb.step.env":     "Obtain <strong>the envelope with the password (passphrase)</strong>: either sealed from the bank vault, or from the technically capable person%s, to whom the system mails it.",
		"rb.step.pc":      "Prepare a <strong>computer disconnected from the internet</strong> (unplug the cable, turn off wifi). This matters: the words from the shares will be typed into it and must not leave it.<br>The <em>inh-offline</em> tool is on a <strong>USB stick</strong>%s. The stick holds several files, one per kind of computer. <strong>Run the one that matches:</strong>",
		"rb.step.pc.os":   "Computer",
		"rb.step.pc.file": "File",
		"rb.step.pc.win":  "Windows",
		"rb.step.pc.mac1": "Mac (newer, 2020 onwards)",
		"rb.step.pc.mac2": "Mac (older, Intel)",
		"rb.step.pc.lin":  "Linux",
		"rb.step.pc.ask":  "If you are not sure which it is, ask the technically capable person. The wrong file breaks nothing, it simply will not start.",

		"rb.step.warn":     "<strong>The computer will probably object.</strong> This is neither an error nor a virus: it is a program nobody paid Microsoft or Apple to register. Go past the warning:",
		"rb.step.warn.win": "<strong>Windows:</strong> a blue window appears, along the lines of „Windows protected your PC“. Click <em>More info</em> and then <em>Run anyway</em>.",
		"rb.step.warn.mac": "<strong>Mac:</strong> it says the file is from an unidentified developer. Close that, <strong>right-click</strong> the file (or click with two fingers) and choose <em>Open</em>; then <em>Open</em> again in the next window. If that does not work, go to <em>Settings → Privacy &amp; Security</em>, where there will be an <em>Open Anyway</em> button.",
		"rb.step.warn.lin": "<strong>Linux:</strong> if the file will not run, the technically capable person makes it executable with <code>chmod +x inh-offline-linux-amd64</code>.",
		"rb.step.warn.url": "If no browser opens by itself, open one and type in the address the tool prints (usually <code>http://127.0.0.1:8777</code>).",

		"rb.step.run":    "Start <em>inh-offline</em> → <em>Recovery</em> → enter %s shares → you get the <strong>key file</strong>. Save it on that computer.",
		"rb.step.kdbx":   "Open the <code>.kdbx</code> database with <strong>KeePassXC</strong> (<a href=\"https://keepassxc.org/download/\">keepassxc.org/download</a>).<br><strong>Best</strong> is to install KeePassXC on that same disconnected computer and copy the database there, so the seed never leaves a machine without internet.<br>If that is not possible, carry the key file on a USB stick to the computer that has KeePassXC, and <strong>disconnect that computer from the internet</strong> for the duration.<br>In KeePassXC choose <em>Key file</em> as the protection and select the key file. <strong>Leave the password field empty.</strong> You reach the <strong>seed</strong> and the other credentials.",
		"rb.step.wallet": "On the hardware wallet, restore the wallet <strong>from the seed</strong> (those words from the database). The password (passphrase) is <strong>not</strong> entered during the restore; the wallet asks for it afterwards, when unlocking. Without it you see an empty wallet, with it the Bitcoin. An empty wallet therefore does not mean the money is gone, it means the password from the envelope is missing.",
		"rb.step.move":   "<em>(optional, but strongly recommended)</em> Move the funds to a new wallet that you control.",

		"rb.manual.h":      "5) Emergency procedure if the tool does not work",
		"rb.manual.slip39": "The shares are standard <strong>SLIP-39</strong> and any SLIP-39 tool combines them (the reference library <code>shamir-mnemonic</code>, for instance). The result is the <strong>master secret (hex)</strong>, which <strong>is</strong> the content of the key file.",
		"rb.manual.abbrev": "On metal the words are often only <strong>four-letter abbreviations</strong>. That is not a fault: in the SLIP-39 wordlist the first four letters identify a word uniquely. <em>inh-offline</em> completes them itself; if you use another tool that rejects abbreviations, look the full words up in the official SLIP-39 wordlist (1024 words, included in every SLIP-39 tool).",
		"rb.manual.bytes":  "This is where it is easy to go wrong: the file must contain <strong>raw bytes</strong>, not that hex written out as text, or KeePassXC will not open the database. On Linux and macOS:",
		"rb.manual.check":  "Check: the file should be half the length of the hex (40 hex characters → 20 bytes). Then use it in KeePassXC as the <em>Key file</em> and leave the password empty. The <code>.kdbx</code> format is open (KeePassXC, KeePass, KeePassDX, Strongbox and others).",
		"rb.manual.ask":    "If none of this makes sense to you, have the technically capable person%s read it.",

		"rb.safety.h":      "6) Safety: what to watch out for",
		"rb.safety.seed":   "<strong>Only ever type the seed into the hardware wallet.</strong> Never into a computer, a phone, an e-mail or any website, no matter who asks.",
		"rb.safety.words":  "<strong>Only ever type the share words</strong> into <em>inh-offline</em> on a computer disconnected from the internet (or, in the emergency procedure, into another SLIP-39 tool, also without internet).",
		"rb.safety.nobody": "Give the words, the seed or the password from the envelope to nobody outside this document, not even to „technical support“.",
		"rb.safety.wipe":   "<strong>When you are done</strong>, delete the key file from the computer and from the USB stick. Whoever has it can open the database. Burn the pieces of paper with the words, or return them to the holders.",
		"rb.safety.move":   "<em>(optional, but strongly recommended)</em> After recovering, move the funds to a new wallet.",
		"rb.safety.slow":   "Take your time. If something does not add up, stop and ask the technically capable person%s.",

		"rb.notes.wallet": "Wallet: notes",
		"rb.notes.family": "A message",

		"rb.print":   "🖨 Print / Save as PDF",
		"rb.edit":    "✏️ Edit",
		"rb.newform": "← new form",
		"rb.margins": "Leave the margins on „Default“ when printing.",
	} {
		enMessages[k] = v
	}
}
