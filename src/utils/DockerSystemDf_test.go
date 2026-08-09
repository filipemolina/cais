package utils

import (
	"testing"
)

func TestParseSystemDf(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
		checks  func(*testing.T, []DiskUsage)
	}{
		{
			name: "real docker system df output",
			input: `{"Type":"Images","TotalCount":"76","Active":"22","Size":"60.11GB","Reclaimable":"42.28GB (70%)"}
{"Type":"Containers","TotalCount":"38","Active":"8","Size":"384.8MB","Reclaimable":"342.7MB (89%)"}
{"Type":"Local Volumes","TotalCount":"34","Active":"12","Size":"1.097GB","Reclaimable":"1.076GB (98%)"}
{"Type":"Build Cache","TotalCount":"10","Active":"0","Size":"1.311GB","Reclaimable":"1.311GB"}`,
			want: 4,
			checks: func(t *testing.T, du []DiskUsage) {
				if du[0].Type != "Images" {
					t.Errorf("got Type %q, want Images", du[0].Type)
				}
				if du[0].TotalCount != 76 {
					t.Errorf("got TotalCount %d, want 76", du[0].TotalCount)
				}
				if du[0].Active != 22 {
					t.Errorf("got Active %d, want 22", du[0].Active)
				}
				// 60.11GB in bytes (60.11 * 1000^3)
				if du[0].Size < 60_000_000_000 || du[0].Size > 61_000_000_000 {
					t.Errorf("got Size %d, want ~60.11GB", du[0].Size)
				}
				if du[0].Reclaimable < 42_000_000_000 || du[0].Reclaimable > 43_000_000_000 {
					t.Errorf("got Reclaimable %d, want ~42.28GB", du[0].Reclaimable)
				}
			},
		},
		{
			name:  "reclaimable without percentage",
			input: `{"Type":"Build Cache","TotalCount":"10","Active":"0","Size":"1.311GB","Reclaimable":"1.311GB"}`,
			want:  1,
			checks: func(t *testing.T, du []DiskUsage) {
				if du[0].Reclaimable < 1_300_000_000 || du[0].Reclaimable > 1_320_000_000 {
					t.Errorf("got Reclaimable %d, want ~1.311GB", du[0].Reclaimable)
				}
			},
		},
		{
			name:  "zero size",
			input: `{"Type":"Images","TotalCount":"0","Active":"0","Size":"0B","Reclaimable":"0B"}`,
			want:  1,
			checks: func(t *testing.T, du []DiskUsage) {
				if du[0].Size != 0 {
					t.Errorf("got Size %d, want 0", du[0].Size)
				}
			},
		},
		{
			name:    "empty input",
			input:   "",
			want:    0,
			wantErr: false,
		},
		{
			name:  "json array instead of ndjson",
			input: `[{"Type":"Images","TotalCount":"76","Active":"22","Size":"60.11GB","Reclaimable":"42.28GB (70%)"}]`,
			want:  1,
			checks: func(t *testing.T, du []DiskUsage) {
				if du[0].Type != "Images" {
					t.Errorf("got Type %q, want Images", du[0].Type)
				}
			},
		},
		{
			name:    "garbage input",
			input:   "not json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSystemDf(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSystemDf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.want {
				t.Errorf("ParseSystemDf() got %d rows, want %d", len(got), tt.want)
				return
			}
			if tt.checks != nil {
				tt.checks(t, got)
			}
		})
	}
}
