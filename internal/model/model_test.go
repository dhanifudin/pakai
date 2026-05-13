package model

import (
	"testing"
	"time"
)

func TestStateConstants(t *testing.T) {
	if StateOK != "ok" {
		t.Errorf("StateOK = %q, want %q", StateOK, "ok")
	}
	if StateError != "error" {
		t.Errorf("StateError = %q, want %q", StateError, "error")
	}
	if StateStale != "stale" {
		t.Errorf("StateStale = %q, want %q", StateStale, "stale")
	}
}

func TestUsageStruct(t *testing.T) {
	now := time.Now()
	pct := 75.5
	cost := 4.20

	u := Usage{
		Provider:    "claude",
		Label:       "Claude Pro",
		State:       StateOK,
		Percent:     &pct,
		Cost:        &cost,
		RefreshedAt: now,
		ErrorMsg:    "",
	}

	if u.Provider != "claude" {
		t.Errorf("Provider = %q, want %q", u.Provider, "claude")
	}
	if u.Label != "Claude Pro" {
		t.Errorf("Label = %q, want %q", u.Label, "Claude Pro")
	}
	if u.State != StateOK {
		t.Errorf("State = %q, want %q", u.State, StateOK)
	}
	if *u.Percent != 75.5 {
		t.Errorf("Percent = %f, want %f", *u.Percent, 75.5)
	}
	if *u.Cost != 4.20 {
		t.Errorf("Cost = %f, want %f", *u.Cost, 4.20)
	}
	if !u.RefreshedAt.Equal(now) {
		t.Errorf("RefreshedAt = %v, want %v", u.RefreshedAt, now)
	}
	if u.ErrorMsg != "" {
		t.Errorf("ErrorMsg = %q, want %q", u.ErrorMsg, "")
	}
}

func TestUsageNilPointers(t *testing.T) {
	u := Usage{
		Provider: "opencode",
		Label:    "OpenCode",
		State:    StateError,
		ErrorMsg: "connection refused",
	}

	if u.Percent != nil {
		t.Error("Percent should be nil when not set")
	}
	if u.Cost != nil {
		t.Error("Cost should be nil when not set")
	}
}

func TestUsageZeroTime(t *testing.T) {
	u := Usage{}
	if !u.RefreshedAt.IsZero() {
		t.Error("RefreshedAt should be zero value by default")
	}
}
