package chrome

import (
	"slices"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
)

func TestPublishedPorts(t *testing.T) {
	cases := []struct {
		name  string
		ports []types.ServicePortConfig
		want  []string
	}{
		{
			name:  "single published port",
			ports: []types.ServicePortConfig{{Published: "14533", Target: 4533, Protocol: "tcp"}},
			want:  []string{"14533"},
		},
		{
			name: "two published values, one with two protocols on the same port",
			ports: []types.ServicePortConfig{
				{Published: "8080", Target: 8080, Protocol: "tcp"},
				{Published: "6881", Target: 6881, Protocol: "tcp"},
				{Published: "6881", Target: 6881, Protocol: "udp"},
			},
			want: []string{"8080", "6881"},
		},
		{
			name:  "unpublished (expose-style) is skipped",
			ports: []types.ServicePortConfig{{Target: 4533, Protocol: "tcp"}},
			want:  []string{},
		},
		{
			name:  "lone udp port keeps its protocol suffix",
			ports: []types.ServicePortConfig{{Published: "53", Target: 53, Protocol: "udp"}},
			want:  []string{"53/udp"},
		},
		{
			name:  "a compose port range passes through as written",
			ports: []types.ServicePortConfig{{Published: "8000-8010", Target: 8000, Protocol: "tcp"}},
			want:  []string{"8000-8010"},
		},
		{
			name:  "host IP is not shown",
			ports: []types.ServicePortConfig{{HostIP: "127.0.0.1", Published: "14533", Target: 4533, Protocol: "tcp"}},
			want:  []string{"14533"},
		},
		{
			name:  "no ports at all",
			ports: nil,
			want:  []string{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PublishedPorts(c.ports)
			if !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestPortLabel(t *testing.T) {
	cases := []struct {
		name string
		port types.ServicePortConfig
		want string
	}{
		{
			name: "published, default protocol",
			port: types.ServicePortConfig{Published: "14533", Target: 4533},
			want: "14533->4533/tcp",
		},
		{
			name: "published with an explicit protocol",
			port: types.ServicePortConfig{Published: "53", Target: 53, Protocol: "udp"},
			want: "53->53/udp",
		},
		{
			name: "not published, exposed only",
			port: types.ServicePortConfig{Target: 4533, Protocol: "tcp"},
			want: "4533/tcp",
		},
		{
			name: "loopback host IP is shown",
			port: types.ServicePortConfig{HostIP: "127.0.0.1", Published: "14533", Target: 4533, Protocol: "tcp"},
			want: "127.0.0.1:14533->4533/tcp",
		},
		{
			name: "wildcard host IP is not shown",
			port: types.ServicePortConfig{HostIP: "0.0.0.0", Published: "14533", Target: 4533, Protocol: "tcp"},
			want: "14533->4533/tcp",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := PortLabel(c.port); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
