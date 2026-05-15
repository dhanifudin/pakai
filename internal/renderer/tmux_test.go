package renderer

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

func readGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	return strings.TrimRight(string(data), "\n")
}

func TestRenderTmux_Ok(t *testing.T) {
	pct := 72.0
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "claude",
		Status:      schema.StatusOK,
		Used:        126.0,
		Limit:       175.0,
		Unit:        "messages",
		RefreshedAt: time.Now(),
	}}
	_ = pct

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_ok.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_Error(t *testing.T) {
	usages := []*schema.Usage{{
		Provider: "claude",
		Label:    "claude",
		Status:   schema.StatusError,
		Error:    "stats cache not found",
	}}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_error.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_Stale(t *testing.T) {
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "claude",
		Status:      schema.StatusStale,
		Used:        126.0,
		Limit:       175.0,
		RefreshedAt: time.Now().Add(-10 * time.Minute),
	}}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_stale.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_OverLimit(t *testing.T) {
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "claude",
		Status:      schema.StatusOK,
		Used:        210.0,
		Limit:       200.0,
		Unit:        "messages",
		RefreshedAt: time.Now(),
	}}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_over_limit.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_Multiple(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider:    "claude",
			Label:       "claude",
			Status:      schema.StatusOK,
			Used:        126.0,
			Limit:       175.0,
			Unit:        "messages",
			RefreshedAt: time.Now(),
		},
		{
			Provider:    "opencode",
			Label:       "opencode",
			Status:      schema.StatusOK,
			Used:        9.0,
			Limit:       20.0,
			Unit:        "usd",
			RefreshedAt: time.Now(),
		},
	}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_multi.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_Mixed(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider:    "claude",
			Label:       "claude",
			Status:      schema.StatusOK,
			Used:        126.0,
			Limit:       175.0,
			Unit:        "messages",
			RefreshedAt: time.Now(),
		},
		{
			Provider: "opencode",
			Label:    "opencode",
			Status:   schema.StatusError,
			Error:    "connection failed",
		},
	}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_mixed.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_TwoProvidersOk(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider:    "claude",
			Label:       "claude",
			Status:      schema.StatusOK,
			Used:        126.0,
			Limit:       175.0,
			Unit:        "messages",
			RefreshedAt: time.Now(),
		},
		{
			Provider:    "opencode",
			Label:       "opencode",
			Status:      schema.StatusOK,
			Used:        1.24,
			Unit:        "usd",
			RefreshedAt: time.Now(),
		},
	}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_two_ok.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_OpenCodeError(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider:    "claude",
			Label:       "claude",
			Status:      schema.StatusOK,
			Used:        126.0,
			Limit:       175.0,
			Unit:        "messages",
			RefreshedAt: time.Now(),
		},
		{
			Provider: "opencode",
			Label:    "opencode",
			Status:   schema.StatusError,
			Error:    "db locked",
		},
	}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_opencode_error.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_MultiWindowProvider(t *testing.T) {
	usages := []*schema.Usage{{
		Provider: "claude",
		Label:    "Claude",
		Status:   schema.StatusOK,
		Windows: []schema.UsageWindow{
			{Key: "5h", Label: "5h", Used: 42, Limit: 100, Unit: "percent"},
			{Key: "weekly", Label: "weekly", Used: 27, Limit: 100, Unit: "percent"},
			{Key: "monthly", Label: "monthly", Used: 126, Limit: 175, Unit: "messages"},
		},
		RefreshedAt: time.Now(),
	}}

	got := RenderTmux(usages, " | ")
	want := readGolden(t, "tmux_multiwindow.golden")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderTmux_PrefersWorstAvailablePercent(t *testing.T) {
	usages := []*schema.Usage{{
		Provider: "claude",
		Label:    "claude",
		Status:   schema.StatusOK,
		Windows: []schema.UsageWindow{
			{Key: "5h", Label: "5h", Used: 11, Limit: 100, Unit: "percent"},
			{Key: "weekly", Label: "weekly", Used: 99, Limit: 100, Unit: "percent"},
			{Key: "monthly", Label: "monthly", Used: 3250, Limit: 0, Unit: "messages"},
		},
		RefreshedAt: time.Now(),
	}}

	got := RenderTmux(usages, " | ")
	if got != "󰚩 99%" {
		t.Errorf("got %q, want %q", got, "󰚩 99%")
	}
}

func TestRenderTmux_WarningMarker(t *testing.T) {
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "claude",
		Status:      schema.StatusOK,
		Used:        3250,
		Unit:        "messages",
		Warning:     "live percentage unavailable",
		RefreshedAt: time.Now(),
	}}

	got := RenderTmux(usages, " | ")
	if got != "󰚩 3.2km" {
		t.Errorf("got %q, want %q", got, "󰚩 3.2km")
	}
}
