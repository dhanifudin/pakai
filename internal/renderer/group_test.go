package renderer

import (
	"testing"

	"github.com/dhanifudin/pakai/internal/schema"
)

func TestPartitionOpencode(t *testing.T) {
	usages := []*schema.Usage{
		{Provider: "claude", Label: "claude"},
		{Provider: "openai", Label: "codex"},
		{Provider: "opencode/openai", Label: "opencode/openai"},
		{Provider: "opencode/opencode-go", Label: "opencode/opencode-go"},
		{Provider: "opencode", Label: "opencode", Status: schema.StatusError},
	}

	standalone, subs := partitionOpencode(usages)

	if len(standalone) != 3 {
		t.Errorf("standalone: got %d, want 3", len(standalone))
	}
	if len(subs) != 2 {
		t.Errorf("subs: got %d, want 2", len(subs))
	}

	standaloneProviders := make(map[string]bool)
	for _, u := range standalone {
		standaloneProviders[u.Provider] = true
	}
	for _, id := range []string{"claude", "openai", "opencode"} {
		if !standaloneProviders[id] {
			t.Errorf("expected %q in standalone", id)
		}
	}

	subProviders := make(map[string]bool)
	for _, u := range subs {
		subProviders[u.Provider] = true
	}
	for _, id := range []string{"opencode/openai", "opencode/opencode-go"} {
		if !subProviders[id] {
			t.Errorf("expected %q in subs", id)
		}
	}
}

func TestSubLabel(t *testing.T) {
	tests := []struct {
		provider string
		label    string
		want     string
	}{
		{"opencode/openai", "opencode/openai", "openai"},
		{"opencode/opencode-go", "opencode/opencode-go", "opencode-go"},
		{"opencode/openai", "", "openai"},
		{"opencode/openai", "Custom Label", "Custom Label"},
		{"opencode/openai", "openai", "openai"},
	}

	for _, tc := range tests {
		u := &schema.Usage{Provider: tc.provider, Label: tc.label}
		got := subLabel(u)
		if got != tc.want {
			t.Errorf("subLabel({Provider:%q, Label:%q}) = %q, want %q", tc.provider, tc.label, got, tc.want)
		}
	}
}

func TestOpencodeWorst(t *testing.T) {
	t.Run("no limits", func(t *testing.T) {
		subs := []*schema.Usage{
			{Provider: "opencode/openai", Label: "openai", Used: 5.0, Unit: "usd", Status: schema.StatusOK},
			{Provider: "opencode/opencode-go", Label: "opencode-go", Used: 10.0, Unit: "usd", Status: schema.StatusOK},
		}
		pct, class := opencodeWorst(subs)
		if pct != -1 {
			t.Errorf("pct = %.1f, want -1 (no limits)", pct)
		}
		if class != "ok" {
			t.Errorf("class = %q, want %q", class, "ok")
		}
	})

	t.Run("one with limit", func(t *testing.T) {
		subs := []*schema.Usage{
			{Provider: "opencode/openai", Label: "openai", Used: 5.0, Limit: 20.0, Unit: "usd", Status: schema.StatusOK},
			{Provider: "opencode/opencode-go", Label: "opencode-go", Used: 18.0, Limit: 30.0, Unit: "usd", Status: schema.StatusOK},
		}
		pct, class := opencodeWorst(subs)
		// opencode-go: 18/30 = 60% (worst)
		if pct < 59 || pct > 61 {
			t.Errorf("pct = %.1f, want ~60", pct)
		}
		if class != "warning" {
			t.Errorf("class = %q, want %q", class, "warning")
		}
	})

	t.Run("error sub", func(t *testing.T) {
		subs := []*schema.Usage{
			{Provider: "opencode/openai", Status: schema.StatusError, Error: "db locked"},
		}
		_, class := opencodeWorst(subs)
		if class != "error" {
			t.Errorf("class = %q, want %q", class, "error")
		}
	})
}

func TestRenderTmux_OpencodeSubProviders(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider: "claude",
			Label:    "claude",
			Status:   schema.StatusOK,
			Used:     30,
			Limit:    100,
			Unit:     "messages",
		},
		{
			Provider: "openai",
			Label:    "codex",
			Status:   schema.StatusOK,
			Used:     1,
			Limit:    100,
			Unit:     "percent",
		},
		{
			Provider: "opencode/openai",
			Label:    "opencode/openai",
			Status:   schema.StatusOK,
			Used:     10,
			Limit:    30,
			Unit:     "usd",
		},
		{
			Provider: "opencode/opencode-go",
			Label:    "opencode/opencode-go",
			Status:   schema.StatusOK,
			Used:     13.2,
			Limit:    30,
			Unit:     "usd",
		},
	}

	got := RenderTmux(usages, " | ")
	// opencode/openai: 10/30 = 33%, opencode/opencode-go: 13.2/30 = 44% (worst)
	// Expected: claude 30% | openai 1% | 󰘦 44%
	want := "󰚩 30% | 󱢆 1% | 󰘦 44%"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
