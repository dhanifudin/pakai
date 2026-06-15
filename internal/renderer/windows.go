package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

func renderProviderCompact(u *schema.Usage) string {
	label := u.Label
	if label == "" {
		label = u.Provider
	}

	windows := u.WindowsOrDefault()
	if len(windows) == 1 {
		w := windows[0]
		pct := w.Pct()
		if pct >= 0 {
			return fmt.Sprintf("%s:%.0f%%", label, pct)
		}
		return fmt.Sprintf("%s:%s", label, w.FormatUsed())
	}

	parts := make([]string, 0, len(windows))
	for _, w := range windows {
		parts = append(parts, renderWindowCompact(w))
	}
	return fmt.Sprintf("%s %s", label, strings.Join(parts, " "))
}

func renderProviderWaybarCompact(u *schema.Usage) string {
	return renderProviderTmuxCompact(u)
}

func renderWindowCompact(w schema.UsageWindow) string {
	label := shortWindowLabel(w)
	pct := w.Pct()
	if pct >= 0 {
		return fmt.Sprintf("%s:%.0f%%", label, pct)
	}
	return fmt.Sprintf("%s:%s", label, w.FormatUsed())
}

func renderWindowWaybarCompact(w schema.UsageWindow) string {
	label := shortWindowLabel(w)
	pct := w.Pct()
	if pct >= 0 {
		return fmt.Sprintf("%s %.0f%%", label, pct)
	}
	return fmt.Sprintf("%s %s", label, w.FormatUsed())
}

func renderProviderTooltip(u *schema.Usage) string {
	label := u.Label
	if label == "" {
		label = u.Provider
	}
	return renderProviderTooltipAs(u, label)
}

// renderProviderTooltipAs is like renderProviderTooltip but uses the given
// label instead of deriving it from the usage struct. Useful for sub-providers
// where the label needs to be stripped of its namespace prefix.
func renderProviderTooltipAs(u *schema.Usage, label string) string {
	header := label
	if icon := providerIcon(u.Provider); icon != "" {
		header = fmt.Sprintf("%s %s", icon, label)
	}

	windows := u.WindowsOrDefault()
	if len(windows) == 1 {
		w := windows[0]
		pct := w.Pct()
		if pct >= 0 {
			return fmt.Sprintf("%s: %.0f%% (%.2f of %.2f %s)", header, pct, w.Used, w.Limit, w.Unit)
		}
		return fmt.Sprintf("%s: %s this month", header, w.FormatUsed())
	}

	lines := []string{header}
	for _, w := range windows {
		for _, line := range renderWindowTooltipLines(w) {
			lines = append(lines, "  "+line)
		}
	}
	return strings.Join(lines, "\n")
}

func renderWindowTooltipLines(w schema.UsageWindow) []string {
	label := shortWindowLabel(w)
	pct := w.Pct()
	reset := formatResetAt(w.ResetAt)

	if w.Unit == "percent" && w.Limit == 100 {
		detail := fmt.Sprintf("%s %s %.0f%%", padWindowLabel(label), progressBar(pct, 10), pct)
		if reset != "" {
			detail += fmt.Sprintf("  %s", reset)
		}
		return []string{detail}
	}

	if pct >= 0 {
		detail := fmt.Sprintf("%s %s %.0f%%", padWindowLabel(label), progressBar(pct, 10), pct)
		meta := fmt.Sprintf("   %s / %s", w.FormatUsed(), formatWindowLimit(w))
		if reset != "" {
			meta += fmt.Sprintf("  %s", reset)
		}
		return []string{detail, meta}
	}

	detail := fmt.Sprintf("%s %s", label, w.FormatUsed())
	if reset != "" {
		detail += fmt.Sprintf("  %s", reset)
	}
	return []string{detail}
}

func padWindowLabel(label string) string {
	if len(label) >= 2 {
		return label
	}
	return label + " "
}

func renderWindowDetail(w schema.UsageWindow) string {
	label := shortWindowLabel(w)
	pct := w.Pct()
	reset := formatResetAt(w.ResetAt)

	if w.Unit == "percent" && w.Limit == 100 {
		if reset != "" {
			return fmt.Sprintf("%s: %.0f%% (%s)", label, pct, reset)
		}
		return fmt.Sprintf("%s: %.0f%%", label, pct)
	}

	if pct >= 0 {
		detail := fmt.Sprintf("%s: %.0f%% (%s of %s)", label, pct, w.FormatUsed(), formatWindowLimit(w))
		if reset != "" {
			detail += fmt.Sprintf(" (%s)", reset)
		}
		return detail
	}

	detail := fmt.Sprintf("%s: %s", label, w.FormatUsed())
	if reset != "" {
		detail += fmt.Sprintf(" (%s)", reset)
	}
	return detail
}

func shortWindowLabel(w schema.UsageWindow) string {
	key := strings.ToLower(w.Key)
	if key == "" {
		key = strings.ToLower(w.Label)
	}

	switch key {
	case "5h", "five_hour", "session":
		return "5h"
	case "weekly", "week", "7d", "seven_day":
		return "w"
	case "monthly", "month", "30d":
		return "m"
	default:
		if w.Label != "" {
			return w.Label
		}
		if w.Key != "" {
			return w.Key
		}
		return "?"
	}
}

func formatWindowLimit(w schema.UsageWindow) string {
	switch w.Unit {
	case "messages":
		return fmt.Sprintf("%.0f msg", w.Limit)
	case "usd":
		return fmt.Sprintf("$%.2f", w.Limit)
	case "tokens":
		if w.Limit >= 1_000_000 {
			return fmt.Sprintf("%.1fM tok", w.Limit/1_000_000)
		}
		if w.Limit >= 1_000 {
			return fmt.Sprintf("%.0fK tok", w.Limit/1_000)
		}
		return fmt.Sprintf("%.0f tok", w.Limit)
	case "percent":
		return fmt.Sprintf("%.0f%%", w.Limit)
	default:
		return fmt.Sprintf("%.2f %s", w.Limit, w.Unit)
	}
}

func formatResetAt(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Until(t).Round(time.Minute)
	if d <= 0 {
		return "resets now"
	}
	if d >= 24*time.Hour {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		if hours == 0 {
			return fmt.Sprintf("resets in %dd", days)
		}
		return fmt.Sprintf("resets in %dd %dh", days, hours)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if hours == 0 {
		return fmt.Sprintf("resets in %dm", minutes)
	}
	return fmt.Sprintf("resets in %dh %02dm", hours, minutes)
}

func providerIcon(provider string) string {
	p := strings.ToLower(provider)
	// Strip "opencode/" namespace prefix to reuse base-provider icons.
	if after, ok := strings.CutPrefix(p, "opencode/"); ok {
		p = after
	}
	switch p {
	case "claude":
		return "󰚩"
	case "openai":
		return "󱢆"
	case "opencode", "opencode-go":
		return "󰘦"
	default:
		return ""
	}
}
