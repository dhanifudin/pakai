package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

// RenderStatus renders human-readable multi-provider status output.
func RenderStatus(usages []*schema.Usage, refreshedAt time.Time) string {
	if len(usages) == 0 {
		return "No providers configured.\n"
	}

	standalone, subs := partitionOpencode(usages)

	var sb strings.Builder

	for _, u := range standalone {
		sb.WriteString(renderUsageLine(u))
	}
	if len(subs) > 0 {
		sb.WriteString(renderOpencodeSection(subs))
	}

	sb.WriteString("\n")

	// Count opencode sub-providers as a single group in the footer.
	providerCount := len(standalone)
	if len(subs) > 0 {
		providerCount++
	}
	plural := "provider"
	if providerCount != 1 {
		plural = "providers"
	}

	if !refreshedAt.IsZero() {
		ago := time.Since(refreshedAt).Round(time.Second)
		sb.WriteString(fmt.Sprintf("  %d %s · refreshed %s ago\n", providerCount, plural, ago))
	} else {
		sb.WriteString(fmt.Sprintf("  %d %s\n", providerCount, plural))
	}

	return sb.String()
}

// renderOpencodeSection renders an "opencode" group header followed by one
// indented row per sub-provider.
func renderOpencodeSection(subs []*schema.Usage) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %s opencode\n", providerIcon("opencode")))
	for _, u := range subs {
		sb.WriteString(renderUsageLineAs(u, subLabel(u), "    "))
	}
	return sb.String()
}

func renderUsageLine(u *schema.Usage) string {
	label := u.Label
	if label == "" {
		label = u.Provider
	}
	return renderUsageLineAs(u, label, "  ")
}

func renderUsageLineAs(u *schema.Usage, label, indent string) string {
	if u.Status == schema.StatusError {
		return fmt.Sprintf("%s%s\n%s  error: %s\n", indent, label, indent, u.Error)
	}

	windows := u.WindowsOrDefault()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s%s\n", indent, label))

	if u.Warning != "" {
		sb.WriteString(fmt.Sprintf("%s  note: %s\n", indent, u.Warning))
	}
	if u.Status == schema.StatusMock {
		sb.WriteString(fmt.Sprintf("%s  mock data\n", indent))
	}

	rendered := false
	for _, w := range windows {
		if w.Pct() <= 0 && w.Used == 0 {
			continue
		}
		sb.WriteString(renderWindowStatusLine(w, indent+"  "))
		rendered = true
	}

	if !rendered {
		sb.WriteString(fmt.Sprintf("%s  no usage\n", indent))
	}

	return sb.String()
}

// renderWindowStatusLine renders a single window line for the status CLI output.
func renderWindowStatusLine(w schema.UsageWindow, indent string) string {
	wname := shortWindowLabel(w)
	pct := w.Pct()
	leftPct := w.PctLeft()

	if pct < 0 {
		line := fmt.Sprintf("%s%3s  %s", indent, wname, w.FormatUsed())
		if reset := formatResetAt(w.ResetAt); reset != "" {
			line += fmt.Sprintf("  ·  %s", reset)
		}
		return line + "\n"
	}

	parts := []string{fmt.Sprintf("%.0f%% left", leftPct)}
	if reserve := renderReserve(w); reserve != "" {
		parts = append(parts, reserve)
	}
	if reset := formatResetAt(w.ResetAt); reset != "" {
		parts = append(parts, reset)
	}

	return fmt.Sprintf("%s%3s  %s\n", indent, wname, strings.Join(parts, "  ·  "))
}
