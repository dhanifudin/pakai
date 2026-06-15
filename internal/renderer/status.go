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
		return fmt.Sprintf("%s%-20s error: %s\n", indent, label, u.Error)
	}

	// Collect windows with non-zero usage and a configured limit.
	windows := u.WindowsOrDefault()
	var parts []string
	for _, w := range windows {
		if pct := w.Pct(); pct > 0 {
			parts = append(parts, fmt.Sprintf("%s: %.0f%%", shortWindowLabel(w), pct))
		}
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s%-20s no usage\n", indent, label)
	}
	return fmt.Sprintf("%s%-14s %s\n", indent, label, strings.Join(parts, " | "))
}
