package renderer

import (
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

func TestRenderWaybar_Ok(t *testing.T) {
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "Claude",
		Status:      schema.StatusOK,
		Used:        126.0,
		Limit:       175.0,
		Unit:        "messages",
		RefreshedAt: time.Now(),
	}}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_ok.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_Error(t *testing.T) {
	usages := []*schema.Usage{{
		Provider: "claude",
		Label:    "Claude",
		Status:   schema.StatusError,
		Error:    "stats cache not found: /home/user/.claude/stats-cache.json",
	}}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_error.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_Stale(t *testing.T) {
	staleTime, _ := time.Parse(time.RFC3339, "2026-05-01T12:00:00Z")
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "Claude",
		Status:      schema.StatusStale,
		Used:        126.0,
		Limit:       175.0,
		RefreshedAt: staleTime,
	}}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_stale.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_OverLimit(t *testing.T) {
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "Claude",
		Status:      schema.StatusOK,
		Used:        210.0,
		Limit:       200.0,
		Unit:        "messages",
		RefreshedAt: time.Now(),
	}}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_over_limit.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_TwoOk(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider:    "Claude",
			Label:       "Claude",
			Status:      schema.StatusOK,
			Used:        126.0,
			Limit:       175.0,
			Unit:        "messages",
			RefreshedAt: time.Now(),
		},
		{
			Provider:    "OpenCode",
			Label:       "OpenCode",
			Status:      schema.StatusOK,
			Used:        1.24,
			Unit:        "usd",
			RefreshedAt: time.Now(),
		},
	}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_two_ok.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_WorstState(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider:    "Claude",
			Label:       "Claude",
			Status:      schema.StatusOK,
			Used:        210.0,
			Limit:       200.0,
			Unit:        "messages",
			RefreshedAt: time.Now(),
		},
		{
			Provider:    "OpenCode",
			Label:       "OpenCode",
			Status:      schema.StatusOK,
			Used:        5.00,
			Unit:        "usd",
			RefreshedAt: time.Now(),
		},
	}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_worst_state.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_MultiWindowProvider(t *testing.T) {
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

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_multiwindow.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}

func TestRenderWaybar_WarningMarker(t *testing.T) {
	usages := []*schema.Usage{{
		Provider:    "claude",
		Label:       "claude",
		Status:      schema.StatusOK,
		Used:        3250,
		Unit:        "messages",
		Warning:     "live percentage unavailable",
		RefreshedAt: time.Now(),
	}}

	got := RenderWaybar(usages, " | ")
	want := readGolden(t, "waybar_warning.golden")
	if got != want {
		t.Errorf("got %s\nwant %s", got, want)
	}
}
