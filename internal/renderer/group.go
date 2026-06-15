package renderer

import (
	"strings"

	"github.com/dhanifudin/pakai/internal/schema"
)

const opencodePrefix = "opencode/"

// partitionOpencode splits usages into two slices: standalone providers (those
// whose Provider field does NOT start with "opencode/") and opencode
// sub-providers (those whose Provider field starts with "opencode/"). A
// whole-provider error from opencode arrives as Provider:"opencode" (no slash)
// and stays in standalone.
func partitionOpencode(usages []*schema.Usage) (standalone, subs []*schema.Usage) {
	for _, u := range usages {
		if strings.HasPrefix(u.Provider, opencodePrefix) {
			subs = append(subs, u)
		} else {
			standalone = append(standalone, u)
		}
	}
	return
}

// subLabel returns the display label for an opencode sub-provider, stripping
// the "opencode/" prefix from both the provider ID and the label.
func subLabel(u *schema.Usage) string {
	label := u.Label
	if label == "" || label == u.Provider {
		if after, ok := strings.CutPrefix(u.Provider, opencodePrefix); ok {
			return after
		}
		return u.Provider
	}
	if after, ok := strings.CutPrefix(label, opencodePrefix); ok {
		return after
	}
	return label
}

// opencodeWorst returns the worst percentage and CSS class across all opencode
// sub-providers. pct is -1 when no sub-provider has a limit configured.
func opencodeWorst(subs []*schema.Usage) (pct float64, class string) {
	pct = -1
	class = "ok"
	for _, u := range subs {
		p := u.WorstPct()
		if p > pct {
			pct = p
		}
		class = worseClass(class, classForUsage(u))
	}
	return
}
