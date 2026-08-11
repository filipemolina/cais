package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/filipemolina/cais/src/appstyles"
	"github.com/filipemolina/cais/src/cmds"
	"github.com/filipemolina/cais/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// The three-tier background system only reads as solid surfaces if every cell
// the app draws carries a background. A run of spaces that follows an ANSI
// reset renders on the terminal's own color instead, which shows up as a notch
// after a title pill, a dark band behind the action buttons, or a block beside
// an empty-state card.
//
// appstyles.HasBackgroundBleed encodes that rule, and these tests assert it
// over fully rendered frames, so a component that joins blocks without sealing
// the result fails here rather than in a screenshot.

// project builds a compose project with a couple of services in one group, so
// the panels render their populated states (member tables, action buttons)
// rather than only their empty states.
func project() *types.Project {
	return &types.Project{
		Services: types.Services{
			"web": types.ServiceConfig{
				Name:     "web",
				Image:    "nginx:latest",
				Profiles: []string{"frontend"},
			},
			"db": types.ServiceConfig{
				Name:     "db",
				Image:    "postgres:16",
				Profiles: []string{"frontend"},
			},
		},
	}
}

// ungroupedProject has services but none carrying a profiles: tag, so
// knownGroups() is empty and the Home details panel shows the service
// overview card rather than the populated group view.
func ungroupedProject() *types.Project {
	return &types.Project{
		Services: types.Services{
			"radarr": types.ServiceConfig{Name: "radarr", Image: "linuxserver/radarr:latest"},
			"sonarr": types.ServiceConfig{Name: "sonarr", Image: "linuxserver/sonarr:latest"},
		},
	}
}

// forEachTheme runs fn once per registered theme, with appstyles.Active set
// to that theme for the duration, in sorted order for stable test output.
// This is the "free property" the roadmap asks for: a theme that leaves a
// tier unsealed fails the exact same assertions every existing state already
// runs, just under a different palette - see docs/ROADMAP.md's Theme phase.
func forEachTheme(t *testing.T, fn func(t *testing.T)) {
	t.Helper()

	original := appstyles.Active
	t.Cleanup(func() { appstyles.Active = original })

	for _, name := range slices.Sorted(maps.Keys(appstyles.Themes)) {
		t.Run(name, func(t *testing.T) {
			appstyles.Active = appstyles.Themes[name]
			fn(t)
		})
	}
}

func TestNoBackgroundBleedAcrossPages(t *testing.T) {
	forEachTheme(t, func(t *testing.T) {
		cases := []struct {
			name  string
			msgs  []tea.Msg
			width int
		}{
			{name: "home empty", width: 120},
			{
				name: "home with groups",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
				},
				width: 120,
			},
			{
				name: "home with a group selected",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.SetSelectedGroupMsg("frontend"),
				},
				width: 120,
			},
			{
				// Services exist but none carry a profiles: tag - the service
				// overview card (docs/plans/service-aware-empty-state.md) rather
				// than the plain onboarding card. Its member table reuses the
				// same row rendering as "home with a group selected" above, but
				// the header line and hint above it are new surfaces of their
				// own.
				name: "home service overview, no groups",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: ungroupedProject()},
				},
				width: 120,
			},
			{
				name: "dashboard empty",
				msgs: []tea.Msg{
					cmds.SetActivePageMsg("Services"),
				},
				width: 120,
			},
			{
				name: "dashboard with a service selected",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.SetActivePageMsg("Services"),
					cmds.SetSelectedServiceMsg(types.ServiceConfig{
						Name:  "web",
						Image: "nginx:latest",
					}),
				},
				width: 120,
			},
			{
				// The Files page's viewport holds raw, unstyled YAML. Lines
				// shorter than the panel width are exactly where an unpainted
				// cell would show through.
				name: "files page with contents",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.SetActivePageMsg("Compose Files"),
					cmds.SetComposeFileMsg{Name: "compose.yaml"},
					cmds.ComposeFileContentsMsg{
						Name:     "compose.yaml",
						Contents: "services:\n  web:\n    image: nginx:alpine\n",
					},
				},
				width: 120,
			},
			{
				// Narrow enough that panels hit their minimum width and content
				// has to wrap, which is where padding maths tends to slip.
				name:  "narrow terminal",
				width: 64,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				m := applyLayout(drive(startup(tc.width, 40), tc.msgs...))

				frame := m.View().Content

				if appstyles.HasBackgroundBleed(frame) {
					t.Errorf("rendered frame has unpainted spaces:\n%s", describeBleed(frame))
				}
			})
		}
	})
}

// A modal is composited over the page beneath it, so an unpainted cell shows
// the page through the modal rather than just the terminal color.
func TestNoBackgroundBleedInModals(t *testing.T) {
	forEachTheme(t, func(t *testing.T) {
		cases := []struct {
			name string
			msgs []tea.Msg
		}{
			{
				name: "create group name prompt",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.OpenCreateGroupModalMsg{},
				},
			},
			{
				name: "rename group prompt",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.OpenRenameGroupModalMsg{GroupName: "frontend"},
				},
			},
			{
				name: "delete group confirmation",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.SetSelectedGroupMsg("frontend"),
					cmds.OpenDeleteGroupModalMsg("frontend"),
				},
			},
			{
				// No compose file in the working directory opens the bootstrap
				// modal off the back of the error.
				name: "bootstrap compose file",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{Err: utils.ErrNoComposeFile},
				},
			},
			{
				name: "help overlay",
				msgs: []tea.Msg{
					cmds.GetConfigMsg{FileName: "compose.yaml", Project: project()},
					cmds.OpenHelpModalMsg{},
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				m := applyLayout(drive(startup(120, 40), tc.msgs...))

				if m.activeModal == nil {
					t.Fatalf("precondition: expected a modal to be open")
				}

				frame := m.View().Content

				if appstyles.HasBackgroundBleed(frame) {
					t.Errorf("frame with %s has unpainted spaces:\n%s", tc.name, describeBleed(frame))
				}
			})
		}
	})
}

func TestNoBackgroundBleedWhenErrorBannerIsShown(t *testing.T) {
	forEachTheme(t, func(t *testing.T) {
		m := applyLayout(drive(startup(120, 40),
			cmds.DockerActionMsg{Action: "start", Target: "frontend", IsGroup: true, Err: errBoom{}},
		))

		frame := m.View().Content

		if !strings.Contains(frame, "boom") {
			t.Fatalf("precondition: expected the error banner to be rendered")
		}

		if appstyles.HasBackgroundBleed(frame) {
			t.Errorf("frame with an error banner has unpainted spaces:\n%s", describeBleed(frame))
		}
	})
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }

// describeBleed reports the offending lines by index, with the escapes made
// visible, so a failure points at a row rather than dumping the whole frame.
func describeBleed(frame string) string {
	var b strings.Builder

	for i, line := range strings.Split(frame, "\n") {
		if appstyles.HasBackgroundBleed(line) {
			fmt.Fprintf(&b, "  line %d: %q\n", i, line)
		}
	}

	return b.String()
}
