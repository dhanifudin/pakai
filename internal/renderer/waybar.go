package renderer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

type WaybarOutput struct {
	Text       string `json:"text"`
	Tooltip    string `json:"tooltip"`
	Class      string `json:"class"`
	Percentage int    `json:"percentage"`
}

func RenderWaybar(usages []*schema.Usage, separator string) string {
	if separator == "" {
		separator = " | "
	}

	standalone, subs := partitionOpencode(usages)
	ordered := orderUsages(standalone, nil)

	var textParts []string
	var tooltipParts []string
	overallClass := "ok"
	overallPct := 0

	for _, u := range ordered {
		providerClass := classForUsage(u)
		overallClass = worseClass(overallClass, providerClass)

		switch u.Status {
		case schema.StatusError:
			textParts = append(textParts, coloredText(tmuxProviderToken(u)+" ?", "error"))
			label := u.Label
			if label == "" {
				label = u.Provider
			}
			tooltipParts = append(tooltipParts, fmt.Sprintf("%s: %s", label, u.Error))

		case schema.StatusStale:
			textParts = append(textParts, coloredText(tmuxProviderToken(u)+" ?", "stale"))
			label := u.Label
			if label == "" {
				label = u.Provider
			}
			tooltipParts = append(tooltipParts, fmt.Sprintf("%s: last refreshed %s", label, u.RefreshedAt.Format(time.RFC3339)))

		default:
			pct := u.WorstPct()
			if pctInt := int(pct); pctInt > overallPct {
				overallPct = pctInt
			}
			compact := renderProviderTmuxCompact(u)
			textParts = append(textParts, coloredText(compact, providerClass))
			tooltip := renderProviderTooltip(u)
			if u.Warning != "" {
				tooltip += "\nwarning: " + u.Warning
			}
			tooltipParts = append(tooltipParts, tooltip)
		}
	}

	if len(subs) > 0 {
		pct, cls := opencodeWorst(subs)
		overallClass = worseClass(overallClass, cls)
		if pct >= 0 {
			if pctInt := int(pct); pctInt > overallPct {
				overallPct = pctInt
			}
		}

		textParts = append(textParts, coloredText(renderOpencodeToken(subs), cls))

		// Tooltip: section header + indented sub-provider detail
		lines := []string{providerIcon("opencode") + " opencode"}
		for _, u := range subs {
			tip := renderProviderTooltipAs(u, subLabel(u))
			if u.Warning != "" {
				tip += "\nwarning: " + u.Warning
			}
			for _, line := range strings.Split(tip, "\n") {
				lines = append(lines, "  "+line)
			}
		}
		tooltipParts = append(tooltipParts, strings.Join(lines, "\n"))
	}

	out := WaybarOutput{
		Text:       strings.Join(textParts, separator),
		Tooltip:    strings.Join(tooltipParts, "\n"),
		Class:      overallClass,
		Percentage: overallPct,
	}

	b, err := json.Marshal(out)
	if err != nil {
		return `{"text":"error","tooltip":"","class":"ok","percentage":0}`
	}
	return string(b)
}

func coloredText(text, class string) string {
	color := pangoColor(class)
	if color == "" {
		return text
	}
	return fmt.Sprintf("<span foreground='%s'>%s</span>", color, text)
}

func pangoColor(class string) string {
	switch class {
	case "ok":
		return "#a6e3a1"
	case "warning":
		return "#f9e2af"
	case "critical":
		return "#f38ba8"
	case "over-limit":
		return "#f38ba8"
	case "error":
		return "#fab387"
	case "stale":
		return "#6c7086"
	default:
		return ""
	}
}

func classForUsage(u *schema.Usage) string {
	if u.Status == schema.StatusError {
		return "error"
	}
	if u.Status == schema.StatusStale {
		return "stale"
	}
	pct := u.WorstPct()
	if pct < 0 {
		return "ok"
	}
	if pct > 100 {
		return "over-limit"
	}
	if pct >= 80 {
		return "critical"
	}
	if pct >= 50 {
		return "warning"
	}
	return "ok"
}

// classPrecedence maps class names to severity (higher = worse).
var classPrecedence = map[string]int{
	"ok":         0,
	"stale":      1,
	"error":      2,
	"warning":    3,
	"critical":   4,
	"over-limit": 5,
}

func worseClass(a, b string) string {
	if classPrecedence[a] >= classPrecedence[b] {
		return a
	}
	return b
}
