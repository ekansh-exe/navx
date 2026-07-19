// Package bots implements §4.5's market-making personas: background
// goroutines that trade through the exact same internal/ledger.ExecuteTrade
// function the HTTP trading handler calls — no special access, no bypass of
// fees, slippage, position caps, or the circuit breaker.
package bots

// Persona distinguishes the trading strategies described in §4.5.
type Persona string

const (
	PersonaMomentum     Persona = "MOMENTUM"
	PersonaContrarian   Persona = "CONTRARIAN"
	PersonaRandomWalker Persona = "RANDOM_WALKER"
	PersonaNewsReactive Persona = "NEWS_REACTIVE"
	PersonaIndexTracker Persona = "INDEX_TRACKER"
)

// SeedAccount pairs a bot username with the persona it should run as.
// Usernames here must exactly match the rows inserted by migration
// 000008_add_bot_support — this list is the single source of truth for
// which persona each seeded bot account plays.
type SeedAccount struct {
	Username string
	Persona  Persona
}

// SeedAccounts is every bot account Run starts a goroutine for.
var SeedAccounts = []SeedAccount{
	{"bot_momentum_1", PersonaMomentum},
	{"bot_momentum_2", PersonaMomentum},
	{"bot_contrarian_1", PersonaContrarian},
	{"bot_contrarian_2", PersonaContrarian},
	{"bot_randomwalker_1", PersonaRandomWalker},
	{"bot_randomwalker_2", PersonaRandomWalker},
	{"bot_newsreactive_1", PersonaNewsReactive},
	{"bot_newsreactive_2", PersonaNewsReactive},
	{"bot_indextracker_1", PersonaIndexTracker},
	{"bot_indextracker_2", PersonaIndexTracker},
}
