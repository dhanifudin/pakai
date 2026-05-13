package cache

import (
	"testing"
	"time"

	"github.com/dhanifudin/pakai/internal/model"
)

func TestMemCacheSetAndGet(t *testing.T) {
	mc := NewMemCache(40 * time.Second)

	usage := model.Usage{
		Provider: "claude",
		Label:    "Claude Pro",
		State:    model.StateOK,
	}

	mc.Set("claude", usage)
	got, stale := mc.Get("claude")

	if stale {
		t.Error("fresh entry should not be stale")
	}
	if got.Provider != "claude" {
		t.Errorf("got Provider = %q, want %q", got.Provider, "claude")
	}
	if got.State != model.StateOK {
		t.Errorf("got State = %q, want %q", got.State, model.StateOK)
	}
}

func TestMemCacheStaleness(t *testing.T) {
	mc := NewMemCache(50 * time.Millisecond)

	usage := model.Usage{Provider: "claude", State: model.StateOK}
	mc.Set("claude", usage)

	_, stale := mc.Get("claude")
	if stale {
		t.Error("fresh entry should not be stale")
	}

	time.Sleep(60 * time.Millisecond)

	got, stale := mc.Get("claude")
	if !stale {
		t.Error("expired entry should be stale")
	}
	if got.Provider != "claude" {
		t.Errorf("stale entry should still return data, got Provider=%q", got.Provider)
	}
}

func TestMemCacheMissingKey(t *testing.T) {
	mc := NewMemCache(40 * time.Second)
	_, stale := mc.Get("nonexistent")
	if !stale {
		t.Error("missing key should return stale=true")
	}
}

func TestMemCacheOverwrite(t *testing.T) {
	mc := NewMemCache(40 * time.Second)

	usage1 := model.Usage{Provider: "claude", State: model.StateOK}
	usage2 := model.Usage{Provider: "claude", State: model.StateError, ErrorMsg: "fetch failed"}

	mc.Set("claude", usage1)
	mc.Set("claude", usage2) // overwrite

	got, _ := mc.Get("claude")
	if got.State != model.StateError {
		t.Errorf("got State = %q, want %q after overwrite", got.State, model.StateError)
	}
	if got.ErrorMsg != "fetch failed" {
		t.Errorf("got ErrorMsg = %q, want %q", got.ErrorMsg, "fetch failed")
	}
}

func TestMemCacheMultipleKeys(t *testing.T) {
	mc := NewMemCache(40 * time.Second)

	mc.Set("claude", model.Usage{Provider: "claude", State: model.StateOK})
	mc.Set("opencode", model.Usage{Provider: "opencode", State: model.StateError})

	claude, stale := mc.Get("claude")
	if stale || claude.Provider != "claude" {
		t.Errorf("claude entry wrong: %+v stale=%v", claude, stale)
	}

	oc, stale := mc.Get("opencode")
	if stale || oc.Provider != "opencode" {
		t.Errorf("opencode entry wrong: %+v stale=%v", oc, stale)
	}
}

func TestMemCacheAllEntries(t *testing.T) {
	mc := NewMemCache(40 * time.Second)

	mc.Set("claude", model.Usage{Provider: "claude"})
	mc.Set("opencode", model.Usage{Provider: "opencode"})

	all := mc.All()
	if len(all) != 2 {
		t.Errorf("got %d entries, want 2", len(all))
	}
}

func TestMemCacheDefaultTTL(t *testing.T) {
	mc := NewMemCache(0)
	usage := model.Usage{Provider: "claude"}
	mc.Set("claude", usage)
	_, stale := mc.Get("claude")
	if stale {
		t.Error("default TTL should make entry not immediately stale")
	}
}
