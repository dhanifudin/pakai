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

	ordered := orderUsages(usages, nil)

	var textParts []string
	var tooltipParts []string
	overallClass := "ok"
	overallPct := 0

	for _, u := range ordered {
		providerClass := classForUsage(u)
		overallClass = worseClass(overallClass, providerClass)

		switch u.Status {
		case schema.StatusError:
			label := u.Label
			if label == "" {
				label = u.Provider
			}
			textParts = append(textParts, fmt.Sprintf("%s:??", label))
			tooltipParts = append(tooltipParts, fmt.Sprintf("%s: %s", label, u.Error))

		case schema.StatusStale:
			label := u.Label
			if label == "" {
				label = u.Provider
			}
			textParts = append(textParts, fmt.Sprintf("%s:??", label))
			tooltipParts = append(tooltipParts, fmt.Sprintf("%s: last refreshed %s", label, u.RefreshedAt.Format(time.RFC3339)))

		default:
			pct := u.WorstPct()
			if pctInt := int(pct); pctInt > overallPct {
				overallPct = pctInt
			}
			textParts = append(textParts, renderProviderWaybarCompact(u))
			tooltip := renderProviderTooltip(u)
			if u.Warning != "" {
				tooltip += "\nwarning: " + u.Warning
			}
			tooltipParts = append(tooltipParts, tooltip)
		}
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
