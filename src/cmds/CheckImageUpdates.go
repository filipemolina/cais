package cmds

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"

	"github.com/filipemolina/cais/src/utils"
)

// ImageUpdatesMsg carries the whole check's answer, keyed by service name.
// There is no error field: the check swallows everything (D3) — offline,
// rate-limited, private registry — and those services simply carry
// ImageUnknown, which renders as nothing at all.
type ImageUpdatesMsg struct {
	Updates map[string]utils.ImageUpdate
}

// remoteRequestTimeout is the per-request budget (D6). The check measured
// ~0.5 s per image against the live registry; 5 s is generous and bounds a
// 20-service stack at ~25 s worst case, ~3 s typical.
const remoteRequestTimeout = 5 * time.Second

// CheckImageUpdates runs the whole image update check for one load: the
// batched local inspect, then the anonymous registry flow, four workers
// deep, every failure per-ref and silent.
//
// services is the merged list — Services and DisabledServices both — because
// a profiled service is not in Services (02-dependency-guard.md §R2) and a
// profiled service's image can go stale exactly like an unprofiled one.
//
// A service with no image: is skipped outright — not Unknown, not listed
// anywhere; there is no reference to check. A pure-digest reference
// (repo@sha256:…, no tag) is checked for nothing: a digest is immutable,
// there is nothing to compare, so it files as Unknown without spending the
// network call (D4).
func CheckImageUpdates(services []types.ServiceConfig, projectLoaded bool) tea.Cmd {
	if !projectLoaded {
		return nil
	}

	// Grouped by the raw reference, because refs two services share are one
	// pair of halves, not two: a 20-service stack on five images is five
	// inspects and five registry reads, and the limiter's budget is per IP.
	type refCheck struct {
		names   []string
		parsed  utils.ImageRef
		checked bool // the ref parsed; its halves can run
		pinned  bool // pure digest: no halves run at all (D4)
	}
	plans := make(map[string]*refCheck)
	updates := make(map[string]utils.ImageUpdate)

	for _, svc := range services {
		if svc.Image == "" {
			continue
		}

		// Every service with a reference starts Unknown — the value the
		// halves can only improve on, never regress.
		updates[svc.Name] = utils.ImageUpdate{Service: svc.Name, Image: svc.Image, State: utils.ImageUnknown}

		plan, ok := plans[svc.Image]
		if !ok {
			plan = &refCheck{}
			if parsed, err := utils.ParseImageRef(svc.Image); err == nil {
				plan.parsed = parsed
				plan.checked = true
			}
			plan.pinned = utils.IsPinnedToDigest(svc.Image)
			plans[svc.Image] = plan
		}
		plan.names = append(plan.names, svc.Name)
	}

	return func() tea.Msg {
		refs := make([]string, 0, len(plans))
		var parsed []utils.ImageRef
		for ref, plan := range plans {
			if !plan.checked || plan.pinned {
				continue
			}
			refs = append(refs, ref)
			parsed = append(parsed, plan.parsed)
		}

		locals, err := utils.LocalImageDigests(refs)
		if err != nil {
			// A daemon that is down fails every ref the same way. The
			// answer is the all-Unknown map just built — which also
			// clears any stale glyphs a previous check left behind —
			// silently; DockerPreflight's existing surface says why,
			// elsewhere.
			return ImageUpdatesMsg{Updates: updates}
		}

		remotes := utils.RemoteImageDigests(parsed, remoteRequestTimeout)

		for ref, plan := range plans {
			if !plan.checked || plan.pinned {
				continue
			}

			remote := ""
			if answer, ok := remotes[utils.ImageRefKey(plan.parsed)]; ok {
				remote = answer.RemoteDigest
			}

			for _, name := range plan.names {
				u := updates[name]
				u.Repo = plan.parsed.Repo
				u.LocalDigest = locals[ref]
				u.RemoteDigest = remote
				u.State = utils.ClassifyImageUpdate(u.LocalDigest, u.RemoteDigest)
				updates[name] = u
			}
		}

		return ImageUpdatesMsg{Updates: updates}
	}
}
