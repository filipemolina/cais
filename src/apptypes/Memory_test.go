package apptypes

import (
	"testing"
)

func TestSumContainerMemory(t *testing.T) {
	tests := []struct {
		name     string
		input    []DockerContainer
		wantSum  int64
		wantCnt  int
	}{
		{
			name: "single container",
			input: []DockerContainer{
				{Service: "test", MemUsage: "128MiB / 512MiB"},
			},
			wantSum: 128 * 1024 * 1024,
			wantCnt: 1,
		},
		{
			name: "multiple containers",
			input: []DockerContainer{
				{Service: "test1", MemUsage: "128MiB / 512MiB"},
				{Service: "test2", MemUsage: "256MiB / 1GiB"},
			},
			wantSum: (128 + 256) * 1024 * 1024,
			wantCnt: 2,
		},
		{
			name: "container with empty MemUsage",
			input: []DockerContainer{
				{Service: "test1", MemUsage: "128MiB / 512MiB"},
				{Service: "test2", MemUsage: ""},
			},
			wantSum: 128 * 1024 * 1024,
			wantCnt: 1,
		},
		{
			name: "container with GiB",
			input: []DockerContainer{
				{Service: "test", MemUsage: "1.5GiB / 4GiB"},
			},
			wantSum: 1536 * 1024 * 1024,
			wantCnt: 1,
		},
		{
			name: "no containers",
			input: []DockerContainer{},
			wantSum: 0,
			wantCnt: 0,
		},
		{
			name: "malformed MemUsage",
			input: []DockerContainer{
				{Service: "test", MemUsage: "invalid"},
			},
			wantSum: 0,
			wantCnt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sum, cnt := SumContainerMemory(tt.input)
			if sum != tt.wantSum {
				t.Errorf("SumContainerMemory() sum = %d, want %d", sum, tt.wantSum)
			}
			if cnt != tt.wantCnt {
				t.Errorf("SumContainerMemory() count = %d, want %d", cnt, tt.wantCnt)
			}
		})
	}
}
