package dockerstatusmodal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/cmds"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
	"github.com/filipemolina/stack-stitcher/src/utils"
)

// Model is the docker diagnosis overlay: which of the five states the
// preflight found, what it means, and the exact command that fixes it on
// this machine - copyable, never run (D2 in docs/plans/docker-preflight.md).
// It is a read-only surface like aboutmodal, closing on esc/q.
type Model struct {
	status utils.DockerStatus
	remedy utils.Remedy
	width  int
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.Overlay.Cancel), key.Matches(keyMsg, keys.Global.Quit):
			return m, cmds.CloseModal(nil)
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	pillBg := appstyles.Active.StatusError
	if m.status.State == utils.DockerPermissionDenied {
		// A configuration problem, not an outage - see the plan.
		pillBg = appstyles.Active.StatusStarting
	}
	pill := lipgloss.NewStyle().
		Background(pillBg).
		Foreground(appstyles.InkOn(pillBg)).
		Bold(true).
		Padding(0, 1).
		Render(strings.ToUpper(stateLabel(m.status.State)))

	dim := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)
	primary := lipgloss.NewStyle().Foreground(appstyles.Active.TextPrimary)
	code := lipgloss.NewStyle().
		Foreground(appstyles.Active.TextPrimary).
		Background(appstyles.Active.BackgroundElevated).
		Padding(0, 1)

	sections := []string{
		chrome.ModalTitle("Docker"),
		pill,
		"",
		primary.Render(wrap(m.remedy.Summary, m.width)),
	}

	if m.status.Endpoint != "" {
		sections = append(sections, dim.Render(wrap("endpoint: "+m.status.Endpoint, m.width)))
	}

	if len(m.remedy.Steps) > 0 {
		lines := make([]string, len(m.remedy.Steps))
		for i, step := range m.remedy.Steps {
			lines[i] = code.Render(step)
		}
		sections = append(sections, "", strings.Join(lines, "\n"))
	}

	if m.remedy.Note != "" {
		sections = append(sections, "", dim.Render(wrap(m.remedy.Note, m.width)))
	}

	if m.remedy.DocsURL != "" {
		sections = append(sections, dim.Render(m.remedy.DocsURL))
	}

	sections = append(sections, "", chrome.ModalHints(chrome.HintAs(keys.Overlay.Cancel, "close")))

	return tea.NewView(chrome.ModalSurface(appstyles.Active.ModalBg, strings.Join(sections, "\n")))
}

// New builds the docker diagnosis overlay for status, resolving the current
// machine's remedy from it. Prose wraps to half the terminal width or 60,
// whichever is smaller - the same rule errormodal uses.
func New(status utils.DockerStatus, terminalWidth int) tea.Model {
	width := min(60, terminalWidth/2)
	if width < 20 {
		width = 20
	}

	return Model{
		status: status,
		remedy: utils.RemedyFor(status, utils.DetectHost()),
		width:  width,
	}
}

func stateLabel(state utils.DockerState) string {
	switch state {
	case utils.DockerNotInstalled:
		return "docker not installed"
	case utils.DockerComposeMissing:
		return "compose plugin missing"
	case utils.DockerDaemonUnreachable:
		return "daemon unreachable"
	case utils.DockerPermissionDenied:
		return "permission denied"
	default:
		return "docker"
	}
}

// wrap breaks s into lines of at most width runes, breaking on spaces. It is
// a plain word-wrap, not lipgloss.Width-aware - the note text is plain ASCII.
func wrap(s string, width int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		line += " " + word
	}
	lines = append(lines, line)

	return strings.Join(lines, "\n")
}
