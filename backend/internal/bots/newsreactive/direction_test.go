package newsreactive

import (
	"testing"

	"github.com/ekansh-exe/navx/internal/domain"
	"github.com/ekansh-exe/navx/internal/news"
)

func TestDirection(t *testing.T) {
	tests := []struct {
		event   news.EventType
		wantOK  bool
		wantTyp domain.TransactionType
	}{
		{news.EventDiscovery, true, domain.TransactionTypeBuy},
		{news.EventFlood, true, domain.TransactionTypeSell},
		{news.EventDrought, true, domain.TransactionTypeSell},
		{news.EventWar, true, domain.TransactionTypeSell},
		{news.EventEmbargo, true, domain.TransactionTypeSell},
		{news.EventStrike, false, ""},
	}
	for _, tt := range tests {
		t.Run(string(tt.event), func(t *testing.T) {
			got, ok := Direction(tt.event)
			if ok != tt.wantOK {
				t.Fatalf("Direction(%q) ok = %v, want %v", tt.event, ok, tt.wantOK)
			}
			if ok && got != tt.wantTyp {
				t.Errorf("Direction(%q) = %v, want %v", tt.event, got, tt.wantTyp)
			}
		})
	}
}
