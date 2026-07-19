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
	TransactionTypeRebalance   TransactionType = "REBALANCE"
	TransactionTypeQuestReward TransactionType = "QUEST_REWARD"
)

// QuestType (§7) — the condition a quest tracks progress toward.
type QuestType string

const (
	QuestTypeMakeTrades QuestType = "MAKE_TRADES"
	QuestTypeHoldCard   QuestType = "HOLD_CARD"
	QuestTypeReachRank  QuestType = "REACH_RANK"
)

// QuestResetTime (§7) — only DAILY is meaningfully handled this phase.
type QuestResetTime string

const (
	QuestResetDaily  QuestResetTime = "DAILY"
	QuestResetWeekly QuestResetTime = "WEEKLY"
)
