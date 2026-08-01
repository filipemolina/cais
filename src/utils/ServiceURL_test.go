package utils

import (
	"testing"

	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/stack-stitcher/src/config"
)

func port(published string, target uint32) types.ServicePortConfig {
	return types.ServicePortConfig{Published: published, Target: target}
}

// The table is the specification - see docs/plans/service-urls.md's
// Research and D2/D3/D7. Every row was checked against the plan's own
// worked examples.
func TestResolveURL(t *testing.T) {
	cases := []struct {
		name string
		svc  types.ServiceConfig
		host string
		want string // "" means ok should be false
		note bool   // want a non-empty Note
	}{
		{
			name: "single published port",
			svc:  types.ServiceConfig{Ports: []types.ServicePortConfig{port("14533", 4533)}},
			host: "10.0.0.5",
			want: "http://10.0.0.5:14533",
		},
		{
			// Both 8096 and 8920 are in the known-port table (both Jellyfin),
			// so neither is promoted over the other - file order breaks the
			// tie, and 18096 was declared first.
			name: "two known ports: file order breaks the tie, not the table",
			svc: types.ServiceConfig{Ports: []types.ServicePortConfig{
				port("18096", 8096),
				port("18920", 8920),
			}},
			host: "10.0.0.5",
			want: "http://10.0.0.5:18096",
		},
		{
			name: "9443 target implies https",
			svc: types.ServiceConfig{Ports: []types.ServicePortConfig{
				port("19443", 9443),
				port("18000", 8000),
			}},
			host: "10.0.0.5",
			want: "https://10.0.0.5:19443",
		},
		{
			name: "udp port skipped",
			svc: types.ServiceConfig{Ports: []types.ServicePortConfig{
				{Published: "16881", Target: 6881, Protocol: "udp"},
				port("18080", 8080),
			}},
			host: "10.0.0.5",
			want: "http://10.0.0.5:18080",
		},
		{
			name: "loopback-bound still shown, with a note",
			svc: types.ServiceConfig{Ports: []types.ServicePortConfig{
				{Published: "14533", Target: 4533, HostIP: "127.0.0.1"},
			}},
			host: "10.0.0.5",
			want: "http://10.0.0.5:14533",
			note: true,
		},
		{
			name: "app_protocol wins the scheme",
			svc: types.ServiceConfig{Ports: []types.ServicePortConfig{
				{Published: "14533", Target: 4533, AppProtocol: "https"},
			}},
			host: "10.0.0.5",
			want: "https://10.0.0.5:14533",
		},
		{
			name: "network_mode host uses the target port directly",
			svc: types.ServiceConfig{
				NetworkMode: "host",
				Ports:       []types.ServicePortConfig{{Target: 8096}},
			},
			host: "10.0.0.5",
			want: "http://10.0.0.5:8096",
			note: true,
		},
		{
			name: "stitcher.url label overrides everything",
			svc: types.ServiceConfig{
				Labels: types.Labels{"stitcher.url": "https://x.ts.net"},
				Ports:  []types.ServicePortConfig{port("14533", 4533)},
			},
			host: "10.0.0.5",
			want: "https://x.ts.net",
		},
		{
			name: "stitcher.url empty suppresses the row",
			svc: types.ServiceConfig{
				Labels: types.Labels{"stitcher.url": ""},
				Ports:  []types.ServicePortConfig{port("14533", 4533)},
			},
			host: "10.0.0.5",
			want: "",
		},
		{
			name: "no ports, no labels",
			svc:  types.ServiceConfig{},
			host: "10.0.0.5",
			want: "",
		},
		{
			name: "ipv6 host is bracketed",
			svc:  types.ServiceConfig{Ports: []types.ServicePortConfig{port("14533", 4533)}},
			host: "fe80::1",
			want: "http://[fe80::1]:14533",
		},
		{
			name: "a port range resolves to its first number",
			svc:  types.ServiceConfig{Ports: []types.ServicePortConfig{port("8000-8010", 8000)}},
			host: "10.0.0.5",
			want: "http://10.0.0.5:8000",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ResolveURL(tc.svc, tc.host)

			if tc.want == "" {
				if ok {
					t.Fatalf("got ok=true URL=%q, want ok=false", got.URL)
				}
				return
			}

			if !ok {
				t.Fatalf("got ok=false, want URL=%q", tc.want)
			}
			if got.URL != tc.want {
				t.Errorf("URL = %q, want %q", got.URL, tc.want)
			}
			if tc.note && got.Note == "" {
				t.Errorf("expected a non-empty Note, got none")
			}
			if !tc.note && got.Note != "" {
				t.Errorf("unexpected Note %q", got.Note)
			}
		})
	}
}

// Exposed-but-not-published ports never win a bridge-network service a URL:
// they are not reachable off the container.
func TestResolveURLIgnoresExposeWithoutNetworkModeHost(t *testing.T) {
	svc := types.ServiceConfig{
		Expose: types.StringOrNumberList{"8096"},
	}

	if _, ok := ResolveURL(svc, "10.0.0.5"); ok {
		t.Fatal("expose: alone should not produce a URL without network_mode: host")
	}
}

// network_mode: host falls back to expose: when ports: is empty.
func TestResolveURLHostNetworkFallsBackToExpose(t *testing.T) {
	svc := types.ServiceConfig{
		NetworkMode: "host",
		Expose:      types.StringOrNumberList{"8096"},
	}

	got, ok := ResolveURL(svc, "10.0.0.5")
	if !ok {
		t.Fatal("expected a URL from expose: under network_mode: host")
	}
	if got.URL != "http://10.0.0.5:8096" {
		t.Errorf("URL = %q, want http://10.0.0.5:8096", got.URL)
	}
}

func TestURLHost(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}

	cases := []struct {
		name string
		cfg  config.Config
		env  map[string]string
		want string
	}{
		{
			name: "config wins over everything",
			cfg:  config.Config{URLHost: "homelab.lan"},
			env:  map[string]string{"SSH_CONNECTION": "192.168.1.9 54321 192.168.1.10 22"},
			want: "homelab.lan",
		},
		{
			name: "SSH_CONNECTION's server field",
			env:  map[string]string{"SSH_CONNECTION": "192.168.1.9 54321 192.168.1.10 22"},
			want: "192.168.1.10",
		},
		{
			name: "malformed SSH_CONNECTION falls back",
			env:  map[string]string{"SSH_CONNECTION": "not enough fields"},
			want: "localhost",
		},
		{
			name: "unset falls back",
			want: "localhost",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := URLHost(tc.cfg, env(tc.env)); got != tc.want {
				t.Errorf("URLHost = %q, want %q", got, tc.want)
			}
		})
	}
}
