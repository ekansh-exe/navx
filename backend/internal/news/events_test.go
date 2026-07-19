package news

import (
	"math/rand"
	"testing"
)

func TestSectorsFor_EveryEventTypeIsMapped(t *testing.T) {
	for _, event := range allEventTypes {
		sectors := sectorsFor(event)
		if len(sectors) == 0 {
			t.Errorf("event %q has no sectors mapped", event)
		}
	}
}

func TestComposeHeadline_ExactFormat(t *testing.T) {
	tests := []struct {
		name    string
		event   EventType
		country Country
		sectors []Sector
		want    string
	}{
		{
			name:    "single sector",
			event:   EventFlood,
			country: Country{Name: "Endia"},
			sectors: []Sector{SectorFood},
			want:    "Flood in Endia affects Food markets",
		},
		{
			name:    "two sectors",
			event:   EventEmbargo,
			country: Country{Name: "Eran"},
			sectors: []Sector{SectorOilGas, SectorMetals},
			want:    "Embargo in Eran affects Oil Gas and Metals markets",
		},
		{
			name:    "three sectors",
			event:   EventWar,
			country: Country{Name: "Chinar"},
			sectors: []Sector{SectorOilGas, SectorMetals, SectorShipping},
			want:    "War in Chinar affects Oil Gas, Metals and Shipping markets",
		},
		{
			name:    "underscore sector renders with a space",
			event:   EventDiscovery,
			country: Country{Name: "Straya"},
			sectors: []Sector{SectorMiscCommodities},
			want:    "Discovery in Straya affects Misc Commodities markets",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := composeHeadline(tt.event, tt.country, tt.sectors)
			if got != tt.want {
				t.Fatalf("composeHeadline() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRandomCountryAndEventType_StayInRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	countrySeen := make(map[string]bool)
	eventSeen := make(map[EventType]bool)
	for i := 0; i < 500; i++ {
		c := randomCountry(rng)
		found := false
		for _, known := range Countries {
			if known.Name == c.Name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("randomCountry returned %q, not in Countries", c.Name)
		}
		countrySeen[c.Name] = true

		e := randomEventType(rng)
		if _, ok := sectorsByEvent[e]; !ok {
			t.Fatalf("randomEventType returned %q, not a known EventType", e)
		}
		eventSeen[e] = true
	}
	if len(eventSeen) != len(allEventTypes) {
		t.Fatalf("across 500 draws, only saw %d/%d event types", len(eventSeen), len(allEventTypes))
	}
	if len(countrySeen) < 2 {
		t.Fatalf("across 500 draws, only saw %d distinct countries — randomCountry looks broken", len(countrySeen))
	}
}
