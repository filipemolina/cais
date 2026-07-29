package apptypes

import (
	"strings"
	"testing"
)

func TestFormatMemUsage(t *testing.T) {
	tests := []struct {
		name     string
		memUsage string
		memPerc  string
		want     string
	}{
		{
			name:     "usage and percent",
			memUsage: "21.71MiB / 31.02GiB",
			memPerc:  "0.07%",
			want:     "21.71MiB (0.07%)",
		},
		{
			name:     "no percent keeps just the used side",
			memUsage: "21.71MiB / 31.02GiB",
			want:     "21.71MiB",
		},
		{
			name:     "no limit to split on",
			memUsage: "21.71MiB",
			memPerc:  "0.07%",
			want:     "21.71MiB (0.07%)",
		},
		{
			name: "empty in, empty out",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatMemUsage(tc.memUsage, tc.memPerc); got != tc.want {
				t.Errorf("FormatMemUsage(%q, %q) = %q, want %q",
					tc.memUsage, tc.memPerc, got, tc.want)
			}
		})
	}
}

// FormatMemUsage is not idempotent: the second pass finds no "/" to split on
// and appends the percent a second time. That is why ServiceListItem holds
// docker's raw strings and formats at render time. This test is here so the
// day someone formats on the way in, it fails loudly rather than shipping
// "21.71MiB (0.07%) (0.07%)" into the list.
func TestFormatMemUsageIsNotIdempotent(t *testing.T) {
	once := FormatMemUsage("21.71MiB / 31.02GiB", "0.07%")
	twice := FormatMemUsage(once, "0.07%")

	if twice == once {
		t.Fatal("FormatMemUsage became idempotent; the raw-value rule can be relaxed")
	}

	if twice != "21.71MiB (0.07%) (0.07%)" {
		t.Errorf("double-format = %q, want the doubled suffix", twice)
	}
}

// The list row renders through FormatMemUsage, so the item has to be holding
// raw docker values for the percent to land exactly once.
func TestServiceListItemDescriptionFormatsOnce(t *testing.T) {
	item := ServiceListItem{
		Status:   "running",
		MemUsage: "21.71MiB / 31.02GiB",
		MemPerc:  "0.07%",
	}

	// The rendered row carries ANSI styling around the value, so match on the
	// substring rather than the whole string.
	got := item.Description(false)

	if !strings.Contains(got, "21.71MiB (0.07%)") {
		t.Errorf("description = %q, want it to contain 21.71MiB (0.07%%)", got)
	}

	if strings.Contains(got, "(0.07%) (0.07%)") {
		t.Errorf("description = %q, want the percent applied once", got)
	}
}
