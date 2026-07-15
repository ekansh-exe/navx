package domain

// UserType distinguishes human players from bot personas (§2). Bots are real
// rows in the users table and trade through the same trade-execution path as
// humans — no special-cased shortcut (§4.5).
type UserType string

const (
	UserTypeHuman UserType = "HUMAN"
	UserTypeBot   UserType = "BOT"
)

// CardType distinguishes system companies, the NAV5 index, and user-created cards (§2).
type CardType string

const (
	CardTypeSystemCompany CardType = "SYSTEM_COMPANY"
	CardTypeIndex         CardType = "INDEX"
	CardTypeUserCreated   CardType = "USER_CREATED"
)

// SupplyModel determines whether a card has a fixed total_supply or mints/burns
// on buy/sell like an AMM (§4.4).
type SupplyModel string

const (
	SupplyModelFixed     SupplyModel = "FIXED"
	SupplyModelUnlimited SupplyModel = "UNLIMITED"
)

// CardStatus (§2).
type CardStatus string

const (
	CardStatusActive   CardStatus = "ACTIVE"
	CardStatusDelisted CardStatus = "DELISTED"
	CardStatusFrozen   CardStatus = "FROZEN"
)

// TransactionType (§2) — every currency-affecting event goes through one of these.
type TransactionType string

const (
	TransactionTypeBuy         TransactionType = "BUY"
	TransactionTypeSell        TransactionType = "SELL"
	TransactionTypeCardLaunch  TransactionType = "CARD_LAUNCH"
	TransactionTypeDailyReward TransactionType = "DAILY_REWARD"
	TransactionTypeFee         TransactionType = "FEE"
)
