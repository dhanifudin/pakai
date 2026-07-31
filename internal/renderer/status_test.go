package renderer

import (
	"strings"
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

func TestRenderStatus_PercentOnly(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider: "claude",
			Label:    "claude",
			Status:   schema.StatusOK,
			Windows: []schema.UsageWindow{
				{Key: "5h", Label: "5h", Used: 51, Limit: 100, Unit: "percent"},
				{Key: "weekly", Label: "weekly", Used: 43, Limit: 100, Unit: "percent"},
				// Monthly has no limit — should be hidden.
				{Key: "monthly", Label: "monthly", Used: 0, Limit: 0, Unit: "messages"},
			},
			RefreshedAt: time.Now(),
		},
	}

	got := RenderStatus(usages, time.Time{})

	if !strings.Contains(got, "49% left") {
		t.Errorf("expected 49%% left for 5h window, got: %q", got)
	}
	if !strings.Contains(got, "57% left") {
		t.Errorf("expected 57%% left for weekly window, got: %q", got)
	}
	// Should not contain dollar signs or raw message counts.
	if strings.Contains(got, "$") || strings.Contains(got, "msg") {
		t.Errorf("should not contain dollar signs or msg units, got: %q", got)
	}
}

func TestRenderStatus_HidesZeroUsageWindows(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider: "codex",
			Label:    "codex",
			Status:   schema.StatusOK,
			Windows: []schema.UsageWindow{
				// 5h has 0% — should be hidden.
				{Key: "5h", Label: "5h", Used: 0, Limit: 100, Unit: "percent"},
				{Key: "weekly", Label: "weekly", Used: 15, Limit: 100, Unit: "percent"},
			},
			RefreshedAt: time.Now(),
		},
	}

	got := RenderStatus(usages, time.Time{})

	if strings.Contains(got, "5h") {
		t.Errorf("zero-usage 5h window should be hidden, got: %q", got)
	}
	if !strings.Contains(got, "85% left") {
		t.Errorf("non-zero weekly window should appear, got: %q", got)
	}
}

func TestRenderStatus_NoUsageFallback(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider: "codex",
			Label:    "codex",
			Status:   schema.StatusOK,
			Windows: []schema.UsageWindow{
				{Key: "5h", Label: "5h", Used: 0, Limit: 100, Unit: "percent"},
				{Key: "weekly", Label: "weekly", Used: 0, Limit: 100, Unit: "percent"},
			},
			RefreshedAt: time.Now(),
		},
	}

	got := RenderStatus(usages, time.Time{})

	if !strings.Contains(got, "no usage") {
		t.Errorf("expected 'no usage' fallback when all windows are zero, got: %q", got)
	}
}

func TestRenderStatus_ErrorLine(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider: "claude",
			Label:    "claude",
			Status:   schema.StatusError,
			Error:    "stats-cache.json not found",
		},
	}

	got := RenderStatus(usages, time.Time{})

	if !strings.Contains(got, "error: stats-cache.json not found") {
		t.Errorf("expected error message in output, got: %q", got)
	}
}

func TestRenderStatus_OpencodeSection_Percentages(t *testing.T) {
	usages := []*schema.Usage{
		{
			Provider: "claude",
			Label:    "claude",
			Status:   schema.StatusOK,
			Windows: []schema.UsageWindow{
				{Key: "5h", Used: 51, Limit: 100, Unit: "percent"},
			},
		},
		{
			Provider: "opencode/openai",
			Label:    "opencode/openai",
			Status:   schema.StatusOK,
			Windows: []schema.UsageWindow{
				{Key: "5h", Used: 3.0, Limit: 12.0, Unit: "usd"},
				{Key: "weekly", Used: 8.0, Limit: 30.0, Unit: "usd"},
				{Key: "monthly", Used: 15.79, Limit: 60.0, Unit: "usd"},
			},
		},
		{
			Provider: "opencode/opencode-go",
			Label:    "opencode/opencode-go",
			Status:   schema.StatusOK,
			Windows: []schema.UsageWindow{
				{Key: "5h", Used: 1.08, Limit: 12.0, Unit: "usd"},
				{Key: "weekly", Used: 6.6, Limit: 30.0, Unit: "usd"},
				{Key: "monthly", Used: 26.47, Limit: 60.0, Unit: "usd"},
			},
		},
	}

	got := RenderStatus(usages, time.Time{})

	// Section header
	if !strings.Contains(got, "opencode") {
		t.Errorf("expected opencode section header, got: %q", got)
	}
	// Sub-provider labels stripped of prefix
	if !strings.Contains(got, "openai") || strings.Contains(got, "opencode/openai") {
		t.Errorf("expected stripped sub-label 'openai', got: %q", got)
	}
	// Monthly pct for openai: 15.79/60 ≈ 26% used → 74% left
	if !strings.Contains(got, "74% left") {
		t.Errorf("expected 74%% left for opencode/openai monthly, got: %q", got)
	}
	// No dollars in compact output
	if strings.Contains(got, "$") {
		t.Errorf("should not contain dollar amounts in status output, got: %q", got)
	}
	// Footer counts opencode as one provider
	if !strings.Contains(got, "2 providers") {
		t.Errorf("expected footer '2 providers' (claude + opencode group), got: %q", got)
	}
}
