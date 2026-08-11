package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/config"
)

// URLSource is how ResolveURL worked out a service's URL, shown dimmed
// beside it so a wrong guess is diagnosable instead of merely wrong.
type URLSource int

const (
	SourceLabel URLSource = iota
	SourceAppProtocol
	SourceKnownPort
	SourcePublishedPort
)

// ServiceURL is the address a service is reachable at, plus how the app
// worked it out and, when the guess is shaky, why.
type ServiceURL struct {
	URL    string
	Source URLSource
	Note   string
}

// urlLabel is the escape hatch: the user's own answer, used
// verbatim and never second-guessed. Namespaced under "cais." so the app
// does not squat on an unprefixed label key in the user's compose file.
// caisURLLabel is the canonical key; stitcherURLLabel is the rename-era
// predecessor, still honored so existing compose files keep working.
const (
	caisURLLabel     = "cais.url"
	stitcherURLLabel = "stitcher.url"
)

// httpsTargetPorts is the tiny, fixed set of container ports that imply
// https when nothing else says so - D2's rung 4. Not the tie-break table
// below; this one only ever picks a scheme.
var httpsTargetPorts = map[uint32]bool{443: true, 8443: true, 9443: true}

// knownWebTargetPorts breaks ties between several published ports - D3.
// It exists to promote a recognisable service port above an unrecognised
// one, nothing more: do not grow it into a catalog of every self-hosted
// app. Keyed on the container's own (target) port, which is the stable
// fact - Jellyfin listens on 8096 inside no matter what the host maps it
// to.
var knownWebTargetPorts = map[uint32]bool{
	80: true, 443: true, 8080: true, 8443: true,
	3000: true, 5000: true, 8000: true, 9000: true,
	8989: true, 32400: true,
	7878: true, 4533: true,
	9696: true, 5055: true,
	8686: true, 13378: true,
	8787: true, 9091: true,
	6767: true, 8081: true,
	8112: true,
	8096: true, 8920: true,
}

// ResolveURL works out the service's main URL. ok is false when the
// service publishes nothing and declares nothing - the caller omits the
// row entirely rather than rendering it empty. host is passed in, never
// read from the environment here, so this whole function is a table test -
// see URLHost for where SSH_CONNECTION is actually read.
func ResolveURL(svc types.ServiceConfig, host string) (ServiceURL, bool) {
	// cais.url is canonical; stitcher.url is honored as the rename-era
	// predecessor so compose files written before the rename keep working.
	if label, ok := svc.Labels[caisURLLabel]; ok {
		if label == "" {
			// Explicit suppression: the only way for a user to say "no,
			// there isn't one" rather than live with a wrong guess forever.
			return ServiceURL{}, false
		}
		return ServiceURL{URL: label, Source: SourceLabel}, true
	}
	if label, ok := svc.Labels[stitcherURLLabel]; ok {
		if label == "" {
			return ServiceURL{}, false
		}
		return ServiceURL{URL: label, Source: SourceLabel}, true
	}

	target, hostPort, note, ok := choosePort(svc)
	if !ok {
		return ServiceURL{}, false
	}

	scheme, source := schemeFor(target, appProtocolFor(svc, target))

	hostPart := host
	if strings.Contains(host, ":") {
		hostPart = "[" + host + "]"
	}

	return ServiceURL{
		URL:    fmt.Sprintf("%s://%s:%d", scheme, hostPart, hostPort),
		Source: source,
		Note:   note,
	}, true
}

// candidatePort is one port ResolveURL could choose, before the tie-break.
type candidatePort struct {
	target   uint32
	hostPort uint32
	loopback bool
}

// choosePort implements D3 (which published port is "the main one") and D7
// (network_mode: host publishes nothing but is reachable on the
// container's own port). Returns the target port (for scheme lookup), the
// host-reachable port number, and an optional note - "bound to
// 127.0.0.1..." or "host network".
func choosePort(svc types.ServiceConfig) (target, hostPort uint32, note string, ok bool) {
	if svc.NetworkMode == "host" {
		return chooseHostNetworkPort(svc)
	}

	var candidates []candidatePort
	for _, p := range svc.Ports {
		if p.Published == "" {
			continue // exposed, not published - never reachable off-container
		}
		if p.Protocol != "" && p.Protocol != "tcp" {
			continue // a UDP port is not a web address
		}
		hp, err := firstPortNumber(p.Published)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidatePort{
			target:   p.Target,
			hostPort: hp,
			loopback: p.HostIP == "127.0.0.1" || p.HostIP == "::1" || p.HostIP == "localhost",
		})
	}

	if len(candidates) == 0 {
		return 0, 0, "", false
	}

	// Discard loopback-bound candidates for the purpose of choosing, unless
	// every candidate is loopback-bound - then take the first anyway and
	// say so, because silently offering an unreachable URL is worse than
	// no URL.
	reachable := make([]candidatePort, 0, len(candidates))
	for _, c := range candidates {
		if !c.loopback {
			reachable = append(reachable, c)
		}
	}
	if len(reachable) == 0 {
		chosen := candidates[0]
		return chosen.target, chosen.hostPort, "bound to 127.0.0.1 - reachable only on this host", true
	}

	chosen := pickKnownOrFirst(reachable)
	return chosen.target, chosen.hostPort, "", true
}

// chooseHostNetworkPort implements D7: network_mode: host ignores ports:'s
// host-side mapping entirely, and the service is reachable on its own
// container port, on the host. ports: and expose: both declare that port;
// ports: is preferred when both are present since it is the more specific
// statement.
func chooseHostNetworkPort(svc types.ServiceConfig) (target, hostPort uint32, note string, ok bool) {
	var candidates []candidatePort

	for _, p := range svc.Ports {
		if p.Protocol != "" && p.Protocol != "tcp" {
			continue
		}
		if p.Target == 0 {
			continue
		}
		candidates = append(candidates, candidatePort{target: p.Target, hostPort: p.Target})
	}

	if len(candidates) == 0 {
		for _, e := range svc.Expose {
			n, err := firstPortNumber(e)
			if err != nil {
				continue
			}
			candidates = append(candidates, candidatePort{target: n, hostPort: n})
		}
	}

	if len(candidates) == 0 {
		return 0, 0, "", false
	}

	chosen := pickKnownOrFirst(candidates)
	return chosen.target, chosen.hostPort, "host network", true
}

// pickKnownOrFirst is D3's tie-break: a candidate whose target port is
// recognised wins over one that is not, in file order; among candidates
// that are equally recognised (or equally unrecognised), file order alone
// decides - the user's own ordering is more meaningful than the port
// number. A port that implies https (httpsTargetPorts) is a web port by
// definition, so it counts as recognised here too even where it is not
// separately listed in knownWebTargetPorts.
func pickKnownOrFirst(candidates []candidatePort) candidatePort {
	for _, c := range candidates {
		if knownWebTargetPorts[c.target] || httpsTargetPorts[c.target] {
			return c
		}
	}
	return candidates[0]
}

// firstPortNumber parses a published/exposed port value, which may be a
// single number or a range ("8000-8010") - the first number in either case.
func firstPortNumber(s string) (uint32, error) {
	s, _, _ = strings.Cut(s, "-")
	s, _, _ = strings.Cut(s, "/")
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(n), nil
}

// appProtocolFor returns the app_protocol declared on the port ResolveURL
// chose, matched by target port - the same fact choosePort already found,
// looked up again because ports: entries do not carry their own identity
// through choosePort's return values.
func appProtocolFor(svc types.ServiceConfig, target uint32) string {
	for _, p := range svc.Ports {
		if p.Target == target && p.AppProtocol != "" {
			return p.AppProtocol
		}
	}
	return ""
}

// schemeFor is D2's rungs 3-4: app_protocol when the compose file declares
// one (the spec's own scheme field - there is no need to invent a
// stitcher.scheme label), otherwise https for the tiny fixed set of
// container ports that conventionally mean it, otherwise http.
func schemeFor(target uint32, appProtocol string) (scheme string, source URLSource) {
	if appProtocol != "" {
		return strings.ToLower(appProtocol), SourceAppProtocol
	}
	if httpsTargetPorts[target] {
		return "https", SourceKnownPort
	}
	return "http", SourcePublishedPort
}

// URLHost is the host part of every service URL the app builds, in order:
//
//  1. config.URLHost   - the user's explicit answer, wins always
//  2. SSH_CONNECTION[2] - the address this SSH client used to get here,
//     measured rather than guessed
//  3. "localhost"       - running locally, which is then correct
//
// env is passed in (rather than reading os.Getenv directly) so this stays
// a table test; AppModel is the one real caller and reads the environment
// once, at startup, since the answer cannot change during a run.
func URLHost(cfg config.Config, env func(string) string) string {
	if cfg.URLHost != "" {
		return cfg.URLHost
	}

	if conn := env("SSH_CONNECTION"); conn != "" {
		fields := strings.Fields(conn)
		if len(fields) == 4 {
			return fields[2]
		}
	}

	return "localhost"
}
