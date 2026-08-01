package utils

import "testing"

const nginxSearchFixture = `{"Description":"Official build of Nginx.","IsOfficial":"true","Name":"nginx","StarCount":"21347"}
{"Description":"NGINX and  NGINX Plus Ingress Controllers for Kubernetes","IsOfficial":"false","Name":"nginx/nginx-ingress","StarCount":"122"}
{"Description":"","IsOfficial":"false","Name":"nginx/nginxaas-loadbalancer-kubernetes","StarCount":"1"}
{"Description":"Nginx, a high-performance reverse proxy & web server. Long-term tracks maintained by Canonical.","IsOfficial":"false","Name":"ubuntu/nginx","StarCount":"141"}
`

func TestParseSearchOutputDecodesRealDockerOutput(t *testing.T) {
	results, err := parseSearchOutput([]byte(nginxSearchFixture))
	if err != nil {
		t.Fatalf("parseSearchOutput: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("got %d results, want 4", len(results))
	}
	if results[0] != (ImageResult{Name: "nginx", Description: "Official build of Nginx.", Stars: 21347, Official: true}) {
		t.Errorf("official image decoded wrong: %+v", results[0])
	}
	if results[2].Description != "" {
		t.Errorf("empty description should decode as empty string, got %q", results[2].Description)
	}
	// Unicode escape in the source JSON must decode to the real character,
	// not stay as the literal &.
	if want := "Nginx, a high-performance reverse proxy & web server. Long-term tracks maintained by Canonical."; results[3].Description != want {
		t.Errorf("got %q, want %q", results[3].Description, want)
	}
}

func TestParseSearchOutputHandlesEmptyOutput(t *testing.T) {
	// Real docker search on zero matches: empty stdout, exit 0 (verified
	// 2026-08-01 with a nonsense query term). This must decode to a nil/
	// empty slice, not an error.
	results, err := parseSearchOutput([]byte(""))
	if err != nil {
		t.Fatalf("parseSearchOutput on empty output: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results from empty output, want 0", len(results))
	}
}

func TestParseSearchOutputDegradesAMalformedStarCount(t *testing.T) {
	results, err := parseSearchOutput([]byte(`{"Description":"x","IsOfficial":"false","Name":"foo/bar","StarCount":"not-a-number"}`))
	if err != nil {
		t.Fatalf("parseSearchOutput: %v", err)
	}
	if len(results) != 1 || results[0].Stars != 0 {
		t.Errorf("got %+v, want Stars: 0 (degraded, not an error)", results)
	}
}
