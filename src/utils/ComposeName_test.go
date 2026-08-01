package utils

import "testing"

func TestIsValidServiceName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"", false},
		{"web", true},
		{"web-1", true},
		{"web_1", true},
		{"Web1", true},
		{"web service", false},
		{"web.service", false},
		{"web:1", false},
		{"web/1", false},
		{"-web", true},
	}

	for _, c := range cases {
		if got := IsValidServiceName(c.name); got != c.want {
			t.Errorf("IsValidServiceName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
