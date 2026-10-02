package util

import (
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var (
	chooserAccent = lipgloss.AdaptiveColor{Light: "#6D28D9", Dark: "#C4B5FD"}
	chooserCyan   = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#67E8F9"}
	chooserText   = lipgloss.AdaptiveColor{Light: "#1E293B", Dark: "#E2E8F0"}
	chooserMuted  = lipgloss.AdaptiveColor{Light: "#64748B", Dark: "#94A3B8"}
	chooserBorder = lipgloss.AdaptiveColor{Light: "#CBD5E1", Dark: "#475569"}
	chooserTint   = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#292342"}
	chooserWarm   = lipgloss.AdaptiveColor{Light: "#92400E", Dark: "#FBBF24"}
)

type chooserTheme struct {
	base, muted, accent, cyan, warm lipgloss.Style
	brand, selected, key, panel     lipgloss.Style
}

func newChooserTheme(output io.Writer) chooserTheme {
	// Detect colour support on the UI's output, including when BibTeX stdout is
	// redirected. Keep the renderer local rather than changing global styling.
	return chooserThemeFromRenderer(lipgloss.NewRenderer(output))
}

func chooserThemeFromRenderer(renderer *lipgloss.Renderer) chooserTheme {
	style := renderer.NewStyle
	return chooserTheme{
		base:     style().Foreground(chooserText),
		muted:    style().Foreground(chooserMuted),
		accent:   style().Foreground(chooserAccent).Bold(true),
		cyan:     style().Foreground(chooserCyan),
		warm:     style().Foreground(chooserWarm),
		brand:    style().Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#7C3AED")).Bold(true),
		selected: style().Foreground(chooserAccent).Background(chooserTint).Bold(true),
		key:      style().Foreground(chooserCyan).Background(chooserTint).Bold(true),
		panel:    style().Border(lipgloss.RoundedBorder()).BorderForeground(chooserBorder).Padding(0, 1),
	}
}

func (theme chooserTheme) box(title, content string, width int, focused bool) string {
	style, heading := theme.panel, theme.muted
	if focused {
		style = style.BorderForeground(chooserAccent)
		heading = theme.accent
	}
	title = ansi.Truncate(title, max(1, width-4), "…")
	return style.Width(max(1, width-2)).Render(heading.Render(title) + "\n" + content)
}

func (theme chooserTheme) details(text string, width int) string {
	paragraphs := strings.Split(displayText(text), "\n\n")
	for i, paragraph := range paragraphs {
		label, value, ok := strings.Cut(paragraph, ": ")
		if !ok || strings.Contains(label, "\n") {
			paragraphs[i] = theme.base.Render(ansi.Wrap(paragraph, width, ""))
			continue
		}
		valueStyle := theme.base
		switch label {
		case "Title":
			valueStyle = valueStyle.Bold(true)
		case "DOI", "mr", "zb", "zbl", "arxiv", "mgp":
			valueStyle = theme.cyan
		case "Edition", "Notes":
			valueStyle = theme.warm
		}
		paragraphs[i] = ansi.Wrap(theme.cyan.Bold(true).Render(label+": ")+valueStyle.Render(value), width, "")
	}
	return strings.Join(paragraphs, "\n")
}

func (theme chooserTheme) shortcut(key, description string) string {
	return theme.key.Render(" "+key+" ") + " " + theme.muted.Render(description)
}
