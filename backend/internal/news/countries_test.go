package news

import "testing"

func TestCountries_NoDuplicatesAndReasonableSize(t *testing.T) {
	if len(Countries) < 15 || len(Countries) > 20 {
		t.Fatalf("len(Countries) = %d, want within §9.1's ~15-20 target", len(Countries))
	}
	seen := make(map[string]bool, len(Countries))
	for _, c := range Countries {
		if c.Name == "" {
			t.Fatal("found a Country with an empty Name")
		}
		if c.RealWorldAnalog == "" {
			t.Fatalf("Country %q has an empty RealWorldAnalog", c.Name)
		}
		if seen[c.Name] {
			t.Fatalf("duplicate country name: %q", c.Name)
		}
		seen[c.Name] = true
	}
}
