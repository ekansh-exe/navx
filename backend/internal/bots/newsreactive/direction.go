package newsreactive

import (
	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/news"
)

// Direction maps a news event to the trade direction the news-reactive
// persona takes on the sectors it affects (§4.5's last bullet): discovery
// is framed as a positive windfall (buy), flood/drought/war/embargo are
// disruptive (sell). Strike isn't classified as clearly positive or
// negative, so the bot takes no action on it — ok is false.
func Direction(event news.EventType) (txType domain.TransactionType, ok bool) {
	switch event {
	case news.EventDiscovery:
		return domain.TransactionTypeBuy, true
	case news.EventFlood, news.EventDrought, news.EventWar, news.EventEmbargo:
		return domain.TransactionTypeSell, true
	default:
		return "", false
	}
}
