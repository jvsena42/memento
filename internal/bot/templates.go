package bot

import (
	"fmt"
	"math/rand"
)

// Confirmation reply templates. Format verbs: %s = date, %s = requester handle.
var confirmationTemplates = []string{
	"Got it! This one's locked away until %s. See you then, @%s",
	"Saved for the future! I'll resurface this on %s, @%s",
	"Time capsule sealed! Opening on %s, @%s",
	"This memory is safe with me. Bringing it back %s, @%s",
	"Locked in! I'll remind you of this on %s, @%s",
	"Stored! This one comes back on %s. Stay tuned, @%s",
	"Consider it saved. Reopening this on %s, @%s",
	"Done! I'll dig this up on %s for you, @%s",
	"Memory captured. See you on %s, @%s",
	"Noted! Bringing this back to life on %s, @%s",
}

// Quote tweet republish templates. Format verbs: %d = years, %s = requester handle.
var quoteTweetTemplates = []string{
	"%d years ago today... @%s",
	"From %d years ago @%s",
	"A blast from %d years back @%s",
	"Throwback: %d years ago @%s",
	"Remember this? %d years ago @%s",
	"This was %d years ago @%s",
	"%d years later... @%s",
	"Revisiting %d years back @%s",
	"Look what surfaced from %d years ago @%s",
	"Unearthed from %d years ago @%s",
}

// Repost/snapshot templates. Format verbs: %s = requester handle, %d = years, %s = truncated text, %s = tweet ID.
var repostTemplates = []string{
	"@%s saved this memory %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"From @%s's archive, %d years back:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"@%s locked this away %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"A memory @%s kept from %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"@%s captured this %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"Saved by @%s, %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"@%s asked to remember this %d years ago:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
	"Resurfacing for @%s after %d years:\n\n\"%s\"\n\nhttps://x.com/i/status/%s",
}

// Deleted tweet snapshot templates. Format verbs: %s = requester handle, %d = years, %s = truncated text, %s = tweet ID.
var deletedTemplates = []string{
	"@%s saved this memory %d years ago, but the original tweet has been deleted.\n\nIt said: \"%s\"\n\nOriginal link: https://x.com/i/status/%s",
	"This was saved by @%s %d years ago. The original is now gone.\n\nIt read: \"%s\"\n\nhttps://x.com/i/status/%s",
	"@%s kept this from %d years ago, but the tweet no longer exists.\n\nIt said: \"%s\"\n\nhttps://x.com/i/status/%s",
	"A memory from @%s, %d years back. The original was deleted.\n\nIt read: \"%s\"\n\nhttps://x.com/i/status/%s",
	"@%s saved this %d years ago. The tweet has since been removed.\n\nIt said: \"%s\"\n\nhttps://x.com/i/status/%s",
	"Saved by @%s %d years ago. The original tweet is gone now.\n\nIt read: \"%s\"\n\nhttps://x.com/i/status/%s",
	"@%s captured this %d years ago, but the original was deleted.\n\nIt said: \"%s\"\n\nhttps://x.com/i/status/%s",
	"From @%s's archive, %d years ago. The tweet no longer exists.\n\nIt read: \"%s\"\n\nhttps://x.com/i/status/%s",
}

func randomTemplate(templates []string, args ...any) string {
	t := templates[rand.Intn(len(templates))]
	return fmt.Sprintf(t, args...)
}
