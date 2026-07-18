package news

import (
	"fmt"
	"math/rand"
	"strings"
)

// EventType is one of §9's fictional geopolitical/commodity event kinds.
type EventType string

const (
	EventFlood     EventType = "FLOOD"
	EventDrought   EventType = "DROUGHT"
	EventWar       EventType = "WAR"
	EventEmbargo   EventType = "EMBARGO"
	EventStrike    EventType = "STRIKE"
	EventDiscovery EventType = "DISCOVERY"
)

var allEventTypes = []EventType{EventFlood, EventDrought, EventWar, EventEmbargo, EventStrike, EventDiscovery}

// Sector mirrors the exact string values migration 000002 seeded onto
// cards.sector — a local type, not a change to domain.Card (which stays a
// plain nullable string).
type Sector string

const (
	SectorAgriculture     Sector = "AGRICULTURE"
	SectorFood            Sector = "FOOD"
	SectorOilGas          Sector = "OIL_GAS"
	SectorSemiconductor   Sector = "SEMICONDUCTOR"
	SectorMetals          Sector = "METALS"
	SectorUtilities       Sector = "UTILITIES"
	SectorShipping        Sector = "SHIPPING"
	SectorMiscCommodities Sector = "MISC_COMMODITIES"
)

// sectorsByEvent is this package's own design (§9 names the event types but
// doesn't map them to sectors) — chosen for plausibility: floods/droughts
// hit food production, wars/embargoes disrupt energy and materials supply
// chains, strikes hit logistics and utilities, discoveries are framed as a
// resource/tech windfall.
var sectorsByEvent = map[EventType][]Sector{
	EventFlood:     {SectorFood, SectorAgriculture},
	EventDrought:   {SectorAgriculture, SectorFood},
	EventWar:       {SectorOilGas, SectorMetals, SectorShipping},
	EventEmbargo:   {SectorOilGas, SectorMetals},
	EventStrike:    {SectorShipping, SectorUtilities},
	EventDiscovery: {SectorSemiconductor, SectorMetals, SectorMiscCommodities},
}

func randomCountry(rng *rand.Rand) Country {
	return Countries[rng.Intn(len(Countries))]
}

func randomEventType(rng *rand.Rand) EventType {
	return allEventTypes[rng.Intn(len(allEventTypes))]
}

// sectorsFor returns the sectors §9's headline generator should cite for
// event. Every EventType in allEventTypes has a non-empty entry in
// sectorsByEvent — enforced by TestSectorsFor_EveryEventTypeIsMapped.
func sectorsFor(event EventType) []Sector {
	return sectorsByEvent[event]
}

// composeHeadline builds the literal format requested for this phase:
// "{Event} in {Country} affects {Sector} markets", with multiple sectors
// joined as "X, Y and Z".
func composeHeadline(event EventType, country Country, sectors []Sector) string {
	displaySectors := make([]string, len(sectors))
	for i, s := range sectors {
		displaySectors[i] = displaySector(s)
	}
	return fmt.Sprintf("%s in %s affects %s markets", displayEvent(event), country.Name, joinSectors(displaySectors))
}

// displayEvent renders "FLOOD" as "Flood".
func displayEvent(e EventType) string {
	s := strings.ToLower(string(e))
	return strings.ToUpper(s[:1]) + s[1:]
}

// displaySector renders "OIL_GAS" as "Oil Gas", "MISC_COMMODITIES" as "Misc
// Commodities".
func displaySector(s Sector) string {
	words := strings.Split(string(s), "_")
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}

// joinSectors renders ["Food"] as "Food", ["Food","Agriculture"] as "Food
// and Agriculture", and 3+ as "A, B and C".
func joinSectors(sectors []string) string {
	switch len(sectors) {
	case 0:
		return ""
	case 1:
		return sectors[0]
	default:
		return strings.Join(sectors[:len(sectors)-1], ", ") + " and " + sectors[len(sectors)-1]
	}
}
