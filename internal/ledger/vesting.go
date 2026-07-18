package ledger

import (
	"math"
	"time"

	"github.com/ekansh-exe/navx/internal/store/db"
)

// VestingPeriod is how long it takes a card creator's retained shares to
// fully unlock (§4.3/§6 step 5: "vesting schedule applied to retained
// shares"). A genuine linear schedule rather than a single hard cutoff —
// §4.3's "e.g. can't dump >Y% in the first 24h" is illustrative, not a
// number the code must hit exactly; a 7-day linear unlock is already more
// conservative in the first 24h (~14% unlocked) than most literal readings
// of that example, while matching "vesting schedule" more literally than a
// single threshold would.
const VestingPeriod = 7 * 24 * time.Hour

// unlockedRetainedShares returns how many of a creator's retained shares
// are sellable at now, linearly unlocking from 0 at createdAt to the full
// amount at createdAt+VestingPeriod. Pure and clock-injectable so it's
// directly unit-testable without a database.
func unlockedRetainedShares(retainedShares int64, createdAt, now time.Time) int64 {
	if retainedShares <= 0 {
		return 0
	}
	elapsed := now.Sub(createdAt)
	if elapsed >= VestingPeriod {
		return retainedShares
	}
	if elapsed <= 0 {
		return 0
	}
	fraction := float64(elapsed) / float64(VestingPeriod)
	return int64(math.Floor(float64(retainedShares) * fraction))
}

// creatorSellLimit computes how many shares a card's creator may sell right
// now, and how many of that amount would count against the vesting-
// restricted pool. Sales are assumed to draw from the restricted
// (originally-retained) pool first, then from any shares bought later —
// the simplest way to track "which shares are restricted" without tagging
// individual shares. Once the restricted pool is exhausted, every further
// sale is unrestricted regardless of vesting progress.
func creatorSellLimit(card db.Card, sharesOwned int64, now time.Time) (maxSellable, unlockedFromRestricted int64) {
	remainingRestricted := card.CreatorRetainedShares - card.CreatorRetainedSharesSold
	if remainingRestricted < 0 {
		remainingRestricted = 0
	}
	freelySellable := sharesOwned - remainingRestricted
	if freelySellable < 0 {
		freelySellable = 0
	}
	unlockedFromRestricted = unlockedRetainedShares(card.CreatorRetainedShares, card.CreatedAt, now) - card.CreatorRetainedSharesSold
	if unlockedFromRestricted < 0 {
		unlockedFromRestricted = 0
	}
	return freelySellable + unlockedFromRestricted, unlockedFromRestricted
}
