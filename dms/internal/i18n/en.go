package i18n

// enMessages is the source of truth for everything the switch sends.
//
// Subjects are deliberately blunt: these arrive months or years after they were
// written, into an inbox that has forgotten this system exists.
var enMessages = map[string]string{
	// --- subjects ---
	"subj.fault":      "[DMS] FAULT, please check",
	"subj.healthy":    "[DMS] all good",
	"subj.checkin":    "[DMS] check in, please",
	"subj.stillwait":  "[DMS] STILL waiting, please check in",
	"subj.awaiting":   "[DMS] silence check started",
	"subj.confirmreq": "[DMS] Request for confirmation",
	"subj.confirmed":  "[DMS] Confirmed, countdown running",
	"subj.partial":    "[DMS] Confirmation received, waiting for more",
	"subj.countdown":  "[DMS] The envelope goes out shortly",
	"subj.sendfail":   "[DMS] ERROR: the envelope could not be sent",
	"subj.sent":       "[DMS] The envelope has been sent",
	"subj.cancelled":  "[DMS] Sending cancelled by your check-in",
	"subj.signaldown": "[DMS] Signal is not working",
	"subj.envelope":   "Important, inheritance: the envelope",

	// --- bodies to the owner ---
	"body.checkin":            "Click to confirm you are all right:\n\n%s\n\nIf you do not respond, the process of handing your credentials to the family begins.\n— DMS",
	"body.checkin.urgent":     "You have not been heard from for a long time. If you are alive, confirm IMMEDIATELY:\n\n%s\n\nIf you do not respond, the process of handing your credentials to the family begins.\n— DMS",
	"body.awaiting":           "You have not been heard from for %s. I have asked the trusted people to confirm.\n\nIf you are alive, CANCEL the process IMMEDIATELY:\n\n%s\n— DMS",
	"body.confirmed":          "%s has confirmed. The envelope goes out in %s.\n\nIf this is a mistake and you are alive, CANCEL it now:\n\n%s\n— DMS",
	"body.partial":            "%s has confirmed. Starting the countdown takes %d confirmations; there are %d so far.\n\nIf you are alive, CANCEL it now:\n\n%s\n— DMS",
	"body.countdown":          "The envelope goes out in roughly %s.\n\nIf you are alive, CANCEL it:\n\n%s\n— DMS",
	"body.healthy":            "The switch is running and the self-tests pass.%s\n\nYour check-in link, if you want to confirm you are alive right now:\n%s\n— DMS",
	"body.healthy.signal.ok":  "\nSignal channel: working.",
	"body.healthy.signal.bad": "\nSignal channel: NOT WORKING (e-mail carries on).",
	"body.fault":              "The switch's self-test FAILED: %s\n\nIt will NOT release anything until this is fixed. Check the service on the server.\n— DMS",
	"body.signaldown":         "The second channel (Signal) is not answering: %s\n\nE-mail carries on and the switch is running normally. Fix Signal when you can.\n— DMS",
	"body.envfail":            "The envelope file is unreadable or not a PDF, so nothing was sent. It will be retried; check the switch.",
	"body.sendfail":           "The envelope could not be delivered on any channel, so nothing was released: %s\nIt will be retried on the next tick.",
	"body.sent":               "The envelope has been sent to the primary heir (%s).\nIf this is a mistake, let them know.",
	"body.cancelled":          "Your check-in arrived while the envelope was going out, and it had already reached the primary heir (%s). It cannot be recalled, so let them know it was a false alarm.",
	"body.cycleid":            "Could not generate a cycle id (the random number generator failed); the confirmation request was NOT sent. It will be retried.",

	// --- to a confirmer ---
	"body.confirmreq": "Hello %s,\n\nthis is an automatic message. %s has not been heard from for some time.\n\nIF you can confirm that they have died or are permanently incapacitated, open the link and confirm with the button.\nThe envelope is then sent to the primary heir after %s.\nIF you cannot confirm this, do nothing.\n\n%s\n— DMS",

	// --- to the primary heir ---
	"body.envelope": "Hello,\n\nif this message has reached you, %s has probably died or is permanently incapacitated.\n\nThe attached PDF is the envelope the runbook talks about: the same text that is kept in the bank vault. On its own it gives nothing away. How to read the passphrase out of it is written in the password database.\n\nTake the runbook and call the technically capable person listed in it, who will help you with the rest.\n— DMS",

	// --- HTTP pages ---
	"page.running":              "The service is running.",
	"page.badlink.title":        "Invalid link",
	"page.badlink":              "Invalid or damaged link.",
	"page.checkin.title":        "Check-in",
	"page.checkin.p":            "Confirm that you are all right:",
	"page.checkin.btn":          "I am alive, reset the timer",
	"page.checkin.done.title":   "Recorded",
	"page.checkin.done":         "✓ Thank you, it is recorded that you are alive. The timer is reset.",
	"page.confirm.title":        "Confirmation",
	"page.confirm.warn":         "<strong>Careful, this is a serious step.</strong> By confirming you declare that the owner has died or is permanently incapacitated. Handing the credentials to the family begins (the envelope goes to the primary heir after a grace period). If you are not certain, DO NOT CONFIRM.",
	"page.confirm.btn":          "I confirm the death or permanent incapacity",
	"page.confirm.done.title":   "Confirmed",
	"page.confirm.done":         "✓ Confirmed. The envelope goes out to the primary heir after the grace period, unless the owner confirms in the meantime that they are alive.",
	"page.confirm.partial":      "✓ Confirmed. %d of the %d required confirmations so far; the countdown starts when the others confirm too.",
	"page.confirm.failed.title": "Cannot confirm",

	// --- misc ---
	"body.raw": "%s\n— DMS",

	"time.days":    "%d days",
	"time.hours":   "%d hours",
	"time.minutes": "%d minutes",
	"owner":        "the owner",
}
