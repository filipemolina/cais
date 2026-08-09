package usageoverlay

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/docker/go-units"
	"github.com/filipemolina/stack-stitcher/src/appstyles"
	"github.com/filipemolina/stack-stitcher/src/apptypes"
	"github.com/filipemolina/stack-stitcher/src/components/chrome"
	"github.com/filipemolina/stack-stitcher/src/keys"
)

// bar renders a proportional two-part bar: filled for used, shaded for the
// remainder, in width columns. Non-zero rounds to at least one cell.
// The bar always occupies exactly `width` cells; total == 0 renders empty.
func bar(used, total int64, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}

	// Calculate filled cells with rounding.
	// Round to nearest, with ties going up.
	ratio := float64(used) / float64(total)
	filled := int(ratio*float64(width) + 0.5)

	// Non-zero values render at least one cell
	if used > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}

	shaded := width - filled

	accentStyle := lipgloss.NewStyle().Foreground(appstyles.Active.Accent)
	dimStyle := lipgloss.NewStyle().Foreground(appstyles.Active.TextDim)

	result := accentStyle.Render(strings.Repeat("█", filled)) +
		dimStyle.Render(strings.Repeat("░", shaded))
	return result
}

// formatSize formats bytes as human-readable size, matching docker's output.
func formatSize(bytes int64) string {
	return units.BytesSize(float64(bytes))
}

func (m Model) renderDiskUsage() string {
	if len(m.disk) == 0 {
		return ""
	}

	// Find the max size for scaling the bars.
	var maxSize int64
	for _, du := range m.disk {
		if du.Size > maxSize {
			maxSize = du.Size
		}
	}

	width := 30 // Bar width in characters
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render("DISK") + "                                             " +
			formatSize(m.getTotalDiskSize()) + " total",
		"",
	}

	for _, du := range m.disk {
		// Scale the bar relative to the max size.
		barStr := bar(du.Size, maxSize, width)
		active := fmt.Sprintf("%d of %d active", du.Active, du.TotalCount)
		sizeStr := formatSize(du.Size)
		reclaimableStr := formatSize(du.Reclaimable)

		line := fmt.Sprintf("  %-12s %s   %s      \n    %s                                 %s idle",
			du.Type, barStr, sizeStr, active, reclaimableStr)
		lines = append(lines, line)
	}

	// Add reclaimable summary line.
	total := m.getTotalReclaimable()
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s is reclaimable — `docker system prune -a --volumes`",
		formatSize(total)))

	return strings.Join(lines, "\n")
}

func (m Model) getTotalDiskSize() int64 {
	var total int64
	for _, du := range m.disk {
		total += du.Size
	}
	return total
}

func (m Model) getTotalReclaimable() int64 {
	var total int64
	for _, du := range m.disk {
		total += du.Reclaimable
	}
	return total
}

func (m Model) renderMemoryUsage() string {
	if len(m.containers) == 0 {
		return ""
	}

	memUsed, count := apptypes.SumContainerMemory(m.containers)

	// If we have no memTotal and no used memory, skip this section.
	if m.memTotal == 0 && memUsed == 0 {
		return ""
	}

	// Determine the denominator: prefer MemTotal, fall back to used as minimum.
	denominator := m.memTotal
	if denominator == 0 {
		denominator = memUsed
	}
	if denominator == 0 {
		return ""
	}

	width := 30 // Bar width in characters
	barStr := bar(memUsed, denominator, width)

	lines := []string{
		"",
		lipgloss.NewStyle().Bold(true).Render("MEMORY") + "                                        ",
		"",
		fmt.Sprintf("  Containers    %s   %s / %s",
			barStr, formatSize(memUsed), formatSize(denominator)),
		fmt.Sprintf("    %d running", count),
	}

	return strings.Join(lines, "\n")
}

func (m Model) View() tea.View {
	if m.loading {
		spinner := m.spinner.View()
		centered := lipgloss.NewStyle().
			Height(10).
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(spinner)

		return tea.NewView(chrome.ModalSurface(
			appstyles.Active.ModalBg,
			centered,
		))
	}

	if m.err != nil {
		// Show error message.
		// If dockerStatus is available and set, show that; otherwise show raw error.
		// For now, show the raw error as per the plan's guidance.
		errMsg := lipgloss.NewStyle().
			Foreground(appstyles.Active.TextDim).
			Render(m.err.Error())

		return tea.NewView(chrome.ModalSurface(
			appstyles.Active.ModalBg,
			strings.Join([]string{
				chrome.ModalTitle("Usage"),
				"",
				errMsg,
			}, "\n"),
		))
	}

	// Build the content sections.
	sections := []string{
		chrome.ModalTitle("Usage"),
		"",
	}

	diskSection := m.renderDiskUsage()
	if diskSection != "" {
		sections = append(sections, diskSection)
	}

	memSection := m.renderMemoryUsage()
	if memSection != "" {
		sections = append(sections, memSection)
	}

	// Add hints at the bottom.
	sections = append(sections, "")
	hint := chrome.RenderKeyHints([]chrome.KeyHint{
		{Key: "r", Desc: "refresh"},
		chrome.HintAs(keys.Overlay.Cancel, "close"),
	}, appstyles.Active.TextMuted)
	sections = append(sections, hint)

	content := strings.Join(sections, "\n")
	return tea.NewView(chrome.ModalSurface(
		appstyles.Active.ModalBg,
		content,
	))
}
