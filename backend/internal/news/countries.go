// Package news implements §9's fictional geopolitical/commodity news feed:
// a background job that periodically picks a fictional country, an event
// type, and the sector(s) it plausibly affects, composes a headline, and
// writes it to news_events. This does not (yet) nudge any card's price or
// feed into bot behavior — that wiring is a later phase.
package news

// Country is a fictional near-homophone of a real country (§9.1) — the
// RealWorldAnalog is kept only as an internal reference for maintainers, it
// is never surfaced to players (the whole point is avoiding real-world
// country references in generated news).
type Country struct {
	Name            string
	RealWorldAnalog string
}

// Countries is §9.1's seed list, extended from the spec's 10 examples to
// its own suggested target of "~15-20 entries to comfortably cover news
// variety". Seeded once, never modified at runtime.
var Countries = []Country{
	{"Endia", "India"},
	{"Use", "USA"},
	{"Eran", "Iran"},
	{"Chinar", "China"},
	{"Rusko", "Russia"},
	{"Brazoria", "Brazil"},
	{"Sadia", "Saudi Arabia"},
	{"Nigera", "Nigeria"},
	{"Kanadia", "Canada"},
	{"Straya", "Australia"},
	{"Brittania", "United Kingdom"},
	{"Germania", "Germany"},
	{"Franconia", "France"},
	{"Nippolia", "Japan"},
	{"Mexiga", "Mexico"},
	{"Koreo", "South Korea"},
	{"Egyptaria", "Egypt"},
	{"Turkiz", "Turkey"},
}
