package newsreactive

import (
	"reflect"
	"testing"

	"github.com/ekansh-exe/navx/internal/news"
)

func TestParseHeadline_RealGeneratorFormats(t *testing.T) {
	tests := []struct {
		name        string
		headline    string
		wantEvent   news.EventType
		wantSectors []news.Sector
	}{
		{
			name:        "single sector",
			headline:    "Flood in Endia affects Food markets",
			wantEvent:   news.EventFlood,
			wantSectors: []news.Sector{news.SectorFood},
		},
		{
			name:        "two sectors",
			headline:    "Embargo in Eran affects Oil Gas and Metals markets",
			wantEvent:   news.EventEmbargo,
			wantSectors: []news.Sector{news.SectorOilGas, news.SectorMetals},
		},
		{
			name:        "three sectors",
			headline:    "War in Chinar affects Oil Gas, Metals and Shipping markets",
			wantEvent:   news.EventWar,
			wantSectors: []news.Sector{news.SectorOilGas, news.SectorMetals, news.SectorShipping},
		},
		{
			name:        "underscore sector renders with a space",
			headline:    "Discovery in Straya affects Misc Commodities markets",
			wantEvent:   news.EventDiscovery,
			wantSectors: []news.Sector{news.SectorMiscCommodities},
		},
		{
			name:        "task's literal example, all-caps sector",
			headline:    "Flood in Endia affects FOOD markets",
			wantEvent:   news.EventFlood,
			wantSectors: []news.Sector{news.SectorFood},
		},
		{
			name:        "strike, agriculture-adjacent multi-sector",
			headline:    "Strike in Kanadia affects Shipping and Utilities markets",
			wantEvent:   news.EventStrike,
			wantSectors: []news.Sector{news.SectorShipping, news.SectorUtilities},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, sectors, err := ParseHeadline(tt.headline)
			if err != nil {
				t.Fatalf("ParseHeadline(%q) returned error: %v", tt.headline, err)
			}
			if event != tt.wantEvent {
				t.Errorf("event = %q, want %q", event, tt.wantEvent)
			}
			if !reflect.DeepEqual(sectors, tt.wantSectors) {
				t.Errorf("sectors = %v, want %v", sectors, tt.wantSectors)
			}
		})
	}
}

func TestParseHeadline_RejectsUnrecognizedInput(t *testing.T) {
	tests := []string{
		"not a headline at all",
		"Flood in Endia affects Toys markets",
		"Whatever in Endia affects Food markets",
	}
	for _, headline := range tests {
		if _, _, err := ParseHeadline(headline); err == nil {
			t.Errorf("ParseHeadline(%q) = nil error, want an error", headline)
		}
	}
}
