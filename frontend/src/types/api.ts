// Mirrors internal/api/dto.go and API_ENDPOINTS.md exactly.
// Currency fields are integers in the smallest unit (1 currency = 100 units).
// Timestamps are RFC3339 strings — parse with `new Date(...)` at the call site.

export type UserType = "HUMAN" | "BOT";
export type CardType = "SYSTEM_COMPANY" | "INDEX" | "USER_CREATED";
export type SupplyModel = "FIXED" | "UNLIMITED";
export type CardStatus = "ACTIVE" | "DELISTED" | "FROZEN";
export type TradeType = "BUY" | "SELL";
export type TransactionType = "BUY" | "SELL" | "FEE" | "CARD_LAUNCH" | "QUEST_REWARD";
export type NewsCategory =
  | "FLOOD"
  | "DROUGHT"
  | "WAR"
  | "EMBARGO"
  | "STRIKE"
  | "DISCOVERY"
  | "CARD_LAUNCH"
  | "CIRCUIT_BREAKER"
  | (string & {});

export interface User {
  id: string;
  username: string;
  user_type: UserType;
  currency_balance: number;
  login_streak_count: number;
  last_login_at: string | null;
  created_at: string;
}

export interface Card {
  id: string;
  creator_user_id: string | null;
  symbol: string;
  name: string;
  description: string | null;
  image_url: string | null;
  card_type: CardType;
  supply_model: SupplyModel;
  total_supply: number | null;
  circulating_supply: number;
  creator_retained_shares: number;
  creator_retained_shares_sold: number;
  current_price: number;
  status: CardStatus;
  created_at: string;
}

export interface Transaction {
  id: string;
  type: TransactionType;
  card_id: string | null;
  shares: number | null;
  price_per_share: number | null;
  total_currency_delta: number;
  resulting_balance: number;
  created_at: string;
}

export interface PriceTick {
  price: number;
  volume: number;
  ts: string;
}

export interface NewsEvent {
  id: string;
  headline: string;
  body: string | null;
  category: NewsCategory;
  related_card_id: string | null;
  created_at: string;
}

export interface LeaderboardEntry {
  rank: number;
  user_id: string;
  username: string;
  net_worth: number;
  change_from_last_refresh?: number;
  // GOAT tribute row only (see backend leaderboard.GoatEntry) — a fixed
  // decoration always at position 0, not a real ranked user.
  is_goat?: boolean;
  net_worth_display?: string;
}

// Proposed — internal/domain.Holding exists and is used server-side (ledger,
// quests, bots) but no endpoint exposes it today. Not even listed as
// NOT YET IMPLEMENTED in API_ENDPOINTS.md; treat this shape as speculative
// until a real `GET /api/users/me/holdings` lands.
export interface Holding {
  card_id: string;
  shares_owned: number;
  avg_cost_basis: number;
  first_bought_at: string | null;
}

export interface HoldingsResponse {
  holdings: Holding[];
}

export interface Quest {
  id: string;
  title: string;
  progress: number;
  target_value: number;
  reward_currency: number;
  completed: boolean;
  reset_at: string;
}

export interface ApiError {
  error: string;
}

// ---- Request/response envelopes ----

export interface RegisterRequest {
  username: string;
  password: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
  reward_granted: boolean;
  reward_amount: number;
}

export interface CardListResponse {
  cards: Card[];
  limit: number;
  offset: number;
}

export interface PriceHistoryResponse {
  card_id: string;
  symbol: string;
  ticks: PriceTick[];
  limit: number;
  offset: number;
}

export interface LaunchCardRequest {
  symbol: string;
  name: string;
  description?: string | null;
  image_url?: string | null;
  total_supply: number;
  retained_percent: number;
  idempotency_key: string;
}

export interface LaunchCardResponse {
  card: Card;
  transaction: Transaction;
  user: User;
}

export interface QuoteRequest {
  card_id: string;
  type: TradeType;
  shares: number;
}

export interface QuoteResponse {
  card: Card;
  type: TradeType;
  shares: number;
  estimated_cost: number;
  estimated_fee: number;
  estimated_price_per_share: number;
}

export interface ExecuteTradeRequest {
  card_id: string;
  type: TradeType;
  shares: number;
  idempotency_key: string;
}

export interface ExecuteTradeResponse {
  transaction: Transaction;
  fee_transaction: Transaction;
  user: User;
  card: Card;
}

export interface LeaderboardResponse {
  leaderboard: LeaderboardEntry[];
}

export interface NewsListResponse {
  news: NewsEvent[];
  limit: number;
  offset: number;
}

export interface QuestListResponse {
  quests: Quest[];
}
