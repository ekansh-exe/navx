// Package newsreactive implements the pure parsing/sentiment half of §4.5's
// news-reactive trader persona: turning a generated headline back into a
// typed event + affected sectors, and mapping that event to a trade
// direction. DB access and trade-queue orchestration live in the bots
// package (news_reactive.go) — this package has no DB dependency so its
// logic can be unit tested directly against headline strings.
package newsreactive

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ekansh-exe/navx/internal/news"
)

var headlineRegexp = regexp.MustCompile(`(?i)^(\w+) in (.+?) affects (.+) markets$`)

var knownEvents = map[news.EventType]bool{
	news.EventFlood:     true,
	news.EventDrought:   true,
	news.EventWar:       true,
	news.EventEmbargo:   true,
	news.EventStrike:    true,
	news.EventDiscovery: true,
}

var knownSectors = map[news.Sector]bool{
	news.SectorAgriculture:     true,
	news.SectorFood:            true,
	news.SectorOilGas:          true,
	news.SectorSemiconductor:   true,
	news.SectorMetals:          true,
	news.SectorUtilities:       true,
	news.SectorShipping:        true,
	news.SectorMiscCommodities: true,
}

// ParseHeadline reverses the exact template internal/news's composeHeadline
// produces: "{Event} in {Country} affects {Sector(s)} markets", with
// multiple sectors joined as "X, Y and Z" (news/events.go's joinSectors).
// Case-insensitive on the event and sector words, since the real generator
// emits title case ("Food") while other callers may pass all-caps.
func ParseHeadline(headline string) (news.EventType, []news.Sector, error) {
	m := headlineRegexp.FindStringSubmatch(strings.TrimSpace(headline))
	if m == nil {
		return "", nil, fmt.Errorf("headline %q doesn't match the expected format", headline)
	}

	event := news.EventType(canonicalize(m[1]))
	if !knownEvents[event] {
		return "", nil, fmt.Errorf("headline %q names an unrecognized event %q", headline, m[1])
	}

	rawSectors := splitSectorList(m[3])
	sectors := make([]news.Sector, 0, len(rawSectors))
	for _, raw := range rawSectors {
		sector := news.Sector(canonicalize(raw))
		if !knownSectors[sector] {
			return "", nil, fmt.Errorf("headline %q names an unrecognized sector %q", headline, raw)
		}
		sectors = append(sectors, sector)
	}
	return event, sectors, nil
}

// canonicalize reverses displayEvent/displaySector: "Oil Gas" -> "OIL_GAS".
func canonicalize(s string) string {
	return strings.ToUpper(strings.Join(strings.Fields(s), "_"))
}

// splitSectorList reverses joinSectors: "A" -> ["A"], "A and B" -> ["A",
// "B"], "A, B and C" -> ["A", "B", "C"].
func splitSectorList(s string) []string {
	rest := ""
	last := s
	if idx := strings.LastIndex(s, " and "); idx != -1 {
		rest, last = s[:idx], s[idx+len(" and "):]
	}
	var parts []string
	if rest != "" {
		parts = strings.Split(rest, ", ")
	}
	return append(parts, last)
}
