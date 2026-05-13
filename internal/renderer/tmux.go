package renderer

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/dhanifudin/pakai/internal/schema"
)

func RenderTmux(usages []*schema.Usage, separator string) string {
	if separator == "" {
		separator = " | "
	}

	ordered := orderUsages(usages, nil)

	var parts []string
	for _, u := range ordered {
		switch u.Status {
		case schema.StatusError, schema.StatusStale:
			parts = append(parts, tmuxProviderToken(u)+" ?")
		default:
			parts = append(parts, renderProviderTmuxCompact(u))
		}
	}

	return strings.Join(parts, separator)
}

func renderProviderTmuxCompact(u *schema.Usage) string {
	token := tmuxProviderToken(u)

	bestPct := -1.0
	for _, w := range u.WindowsOrDefault() {
		pct := w.Pct()
		if pct > bestPct {
			bestPct = pct
		}
	}

	if bestPct >= 0 {
		return maybeWarn(token, fmt.Sprintf("%.0f%%", bestPct), u.Warning)
	}

	return maybeWarn(token, tmuxCompactValue(u), u.Warning)
}

func maybeWarn(token, value, warning string) string {
	if warning != "" {
		return token + " !" + value
	}
	return token + " " + value
}

func tmuxProviderToken(u *schema.Usage) string {
	switch strings.ToLower(u.Provider) {
	case "claude":
		return "󰚩"
	case "openai":
		return "󱢆"
	case "opencode":
		return "󰘦"
	case "opencode-go":
		return "󰘦"
	}

	label := u.Label
	if label == "" {
		label = u.Provider
	}
	label = strings.ToLower(strings.TrimSpace(label))
	if len(label) <= 2 {
		return label
	}
	return label[:2]
}

func tmuxCompactValue(u *schema.Usage) string {
	return compactUsed(u.Used, u.Unit)
}

func compactUsed(used float64, unit string) string {
	switch unit {
	case "messages":
		return compactNumber(used) + "m"
	case "usd":
		return "$" + trimCompactFloat(used)
	case "tokens":
		return compactNumber(used) + "t"
	default:
		return trimCompactFloat(used)
	}
}

func compactNumber(v float64) string {
	abs := math.Abs(v)
	sign := ""
	if v < 0 {
		sign = "-"
	}
	switch {
	case abs >= 1_000_000:
		return sign + trimCompactFloat(abs/1_000_000) + "M"
	case abs >= 1_000:
		return sign + trimCompactFloat(abs/1_000) + "k"
	default:
		return sign + trimCompactFloat(abs)
	}
}

func trimCompactFloat(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.05 {
		return fmt.Sprintf("%.0f", math.Round(v))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), ".")
}

func orderUsages(usages []*schema.Usage, order []string) []*schema.Usage {
	if len(order) == 0 {
		ordered := make([]*schema.Usage, len(usages))
		copy(ordered, usages)
		sort.Slice(ordered, func(i, j int) bool {
			return ordered[i].Provider < ordered[j].Provider
		})
		return ordered
	}

	result := make([]*schema.Usage, 0, len(usages))
	seen := make(map[string]bool)
	for _, id := range order {
		for _, u := range usages {
			if u.Provider == id && !seen[id] {
				result = append(result, u)
				seen[id] = true
			}
		}
	}
	for _, u := range usages {
		if !seen[u.Provider] {
			result = append(result, u)
			seen[u.Provider] = true
		}
	}
	return result
}
