package model

import (
	"fmt"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/utils"
	"strings"
	"testing"

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

func TestNoBackgroundBleedAcrossPages(t *testing.T) {
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
}

// A modal is composited over the page beneath it, so an unpainted cell shows
// the page through the modal rather than just the terminal color.
func TestNoBackgroundBleedInModals(t *testing.T) {
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
}

func TestNoBackgroundBleedWhenErrorBannerIsShown(t *testing.T) {
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
