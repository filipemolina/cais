package chrome

import (
	"fmt"

	"github.com/compose-spec/compose-go/v2/types"
)

// PortLabel is one published port in full: the form the service details
// panel shows, where there is room for it.
//
//	14533->4533/tcp
//	127.0.0.1:14533->4533/tcp   (bound to loopback: not reachable off-host)
//	4533/tcp                    (exposed, not published)
func PortLabel(p types.ServicePortConfig) string {
	protocol := p.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	label := fmt.Sprintf("%d/%s", p.Target, protocol)
	if p.Published == "" {
		return label
	}

	published := p.Published
	if p.HostIP != "" && p.HostIP != "0.0.0.0" {
		published = p.HostIP + ":" + published
	}

	return published + "->" + label
}

// portEntry tracks the protocols seen for one published value, in the order
// they were first seen, so PublishedPorts can tell "every entry is non-TCP"
// from "TCP is one of several" without depending on map iteration order.
type portEntry struct {
	published string
	protocols []string
}

// PublishedPorts returns the host-side ports a service publishes, in file
// order, deduplicated, for a column too narrow for the full mapping. A
// service that publishes nothing returns an empty slice.
//
//	[]{14533:4533/tcp}                        -> ["14533"]
//	[]{8080:8080/tcp, 6881:6881/tcp+udp}      -> ["8080", "6881"]
//	[]{4533/tcp}                              -> []          (not published)
//	[]{53:53/udp}                             -> ["53/udp"]
func PublishedPorts(ports []types.ServicePortConfig) []string {
	var entries []portEntry
	index := map[string]int{}

	for _, p := range ports {
		if p.Published == "" {
			continue
		}

		protocol := p.Protocol
		if protocol == "" {
			protocol = "tcp"
		}

		i, ok := index[p.Published]
		if !ok {
			i = len(entries)
			index[p.Published] = i
			entries = append(entries, portEntry{published: p.Published})
		}

		seen := false
		for _, existing := range entries[i].protocols {
			if existing == protocol {
				seen = true
				break
			}
		}
		if !seen {
			entries[i].protocols = append(entries[i].protocols, protocol)
		}
	}

	result := make([]string, 0, len(entries))
	for _, e := range entries {
		allNonTCP := true
		for _, protocol := range e.protocols {
			if protocol == "tcp" {
				allNonTCP = false
				break
			}
		}

		if allNonTCP && len(e.protocols) == 1 {
			result = append(result, e.published+"/"+e.protocols[0])
		} else {
			result = append(result, e.published)
		}
	}

	return result
}
