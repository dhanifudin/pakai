package opencode

import (
	"context"
	"database/sql"
	"math"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/dhanifudin/pakai/internal/config"
	"github.com/dhanifudin/pakai/internal/schema"
	"github.com/dhanifudin/pakai/internal/testutil"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func currentMonthMillis(t *testing.T) (int64, int64) {
	t.Helper()
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0)
	return start.UnixMilli(), end.UnixMilli()
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestFetchAll_SingleProvider(t *testing.T) {
	db := newTestDB(t)
	start, _ := currentMonthMillis(t)
	mid := start + 1000

	err := testutil.SeedDB(db, []testutil.MessageRow{
		{ID: "m1", SessionID: "s1", CreatedAt: mid, UpdatedAt: mid, Role: "assistant", Cost: 2.50, Provider: "openai", TokensIn: 1000, TokensOut: 500},
		{ID: "m2", SessionID: "s1", CreatedAt: mid + 100, UpdatedAt: mid + 100, Role: "assistant", Cost: 1.50, Provider: "openai", TokensIn: 800, TokensOut: 300},
	})
	if err != nil {
		t.Fatalf("SeedDB: %v", err)
	}

	p := NewFromDB(db)
	ctx := context.Background()
	got, err := p.FetchAll(ctx)

	if err != nil {
		t.Fatalf("FetchAll error = %v, want nil", err)
	}

	if len(got) == 0 {
		t.Fatal("got 0 usages, want at least 1")
	}

	wantCost := 4.0
	if got[0].Used != wantCost {
		t.Errorf("got Used = %.2f, want %.2f", got[0].Used, wantCost)
	}
}

func TestFetchAll_MultipleProviders(t *testing.T) {
	db := newTestDB(t)
	start, _ := currentMonthMillis(t)
	mid := start + 1000

	err := testutil.SeedDB(db, []testutil.MessageRow{
		{ID: "m1", SessionID: "s1", CreatedAt: mid, UpdatedAt: mid, Role: "assistant", Cost: 3.00, Provider: "openai", TokensIn: 1000, TokensOut: 500},
		{ID: "m2", SessionID: "s1", CreatedAt: mid + 100, UpdatedAt: mid + 100, Role: "assistant", Cost: 2.00, Provider: "anthropic", TokensIn: 800, TokensOut: 300},
		{ID: "m3", SessionID: "s1", CreatedAt: mid + 200, UpdatedAt: mid + 200, Role: "user", Cost: 5.00, Provider: "openai", TokensIn: 0, TokensOut: 0},
	})
	if err != nil {
		t.Fatalf("SeedDB: %v", err)
	}

	p := NewFromDB(db)
	ctx := context.Background()
	got, err := p.FetchAll(ctx)

	if err != nil {
		t.Fatalf("FetchAll error = %v, want nil", err)
	}

	// Should have 2 providers (openai, anthropic)
	if len(got) < 2 {
		t.Fatalf("got %d providers, want 2", len(got))
	}

	// m3 is "user" role — should be excluded
	totalCost := 0.0
	for _, u := range got {
		totalCost += u.Used
	}
	wantTotal := 5.0
	if totalCost != wantTotal {
		t.Errorf("got total cost = %.2f, want %.2f", totalCost, wantTotal)
	}
}

func TestFetchAll_EmptyMonth(t *testing.T) {
	db := newTestDB(t)
	now := time.Now()
	// Insert data from previous month
	prevMonth := time.Date(now.Year(), now.Month()-1, 15, 0, 0, 0, 0, now.Location())
	prevMillis := prevMonth.UnixMilli()

	err := testutil.SeedDB(db, []testutil.MessageRow{
		{ID: "m1", SessionID: "s1", CreatedAt: prevMillis, UpdatedAt: prevMillis, Role: "assistant", Cost: 5.00, Provider: "openai", TokensIn: 1000, TokensOut: 500},
	})
	if err != nil {
		t.Fatalf("SeedDB: %v", err)
	}

	p := NewFromDB(db)
	ctx := context.Background()
	got, err := p.FetchAll(ctx)

	if err != nil {
		t.Fatalf("FetchAll error = %v, want nil", err)
	}

	// Previous month data should not be included
	if len(got) != 0 {
		t.Errorf("got %d usages from previous month, want 0", len(got))
	}
}

func TestID(t *testing.T) {
	p := New()
	got := p.ID()
	want := "opencode"
	if got != want {
		t.Errorf("got ID = %q, want %q", got, want)
	}
}

func TestDBPath(t *testing.T) {
	p := New()
	got := p.DBPath()
	if got == "" {
		t.Error("DBPath returned empty string")
	}
}

func TestFetchAll_DBError(t *testing.T) {
	db := newTestDB(t)
	db.Close()

	p := NewFromDB(db)
	ctx := context.Background()
	_, err := p.FetchAll(ctx)

	if err != nil {
		t.Logf("expected error on closed db: %v", err)
	} else {
		t.Error("expected error on closed database connection")
	}
}

func TestFetchAll_RealCostProviderGetsThreeWindows(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	db := newTestDB(t)
	start, _ := currentMonthMillis(t)

	// Place spend inside both the 5h and weekly windows.
	now := time.Now()
	recentMillis := now.Add(-1*time.Hour).UnixMilli()
	mid := start + 1000

	err := testutil.SeedDB(db, []testutil.MessageRow{
		// Older spend — counts for month and week, not 5h
		{ID: "m1", SessionID: "s1", CreatedAt: mid, UpdatedAt: mid, Role: "assistant", Cost: 6.0, Provider: "opencode-go", TokensIn: 100, TokensOut: 100},
		// Recent spend — counts for month, week, and 5h
		{ID: "m2", SessionID: "s1", CreatedAt: recentMillis, UpdatedAt: recentMillis, Role: "assistant", Cost: 6.0, Provider: "opencode-go", TokensIn: 100, TokensOut: 100},
	})
	if err != nil {
		t.Fatalf("SeedDB: %v", err)
	}

	p := NewFromDB(db)
	ctx := context.Background()
	got, err := p.FetchAll(ctx)
	if err != nil {
		t.Fatalf("FetchAll error = %v, want nil", err)
	}

	if len(got) != 1 {
		t.Fatalf("got %d providers, want 1", len(got))
	}

	u := got[0]
	if u.Provider != "opencode/opencode-go" {
		t.Fatalf("got provider %q, want opencode/opencode-go", u.Provider)
	}

	// Real-cost provider should get 3 windows (5h, weekly, monthly).
	if len(u.Windows) != 3 {
		t.Fatalf("got %d windows, want 3 (5h/weekly/monthly)", len(u.Windows))
	}

	// Limit should fall back to goMonthLimit (60) when not configured.
	if u.Limit != goMonthLimit {
		t.Errorf("got Limit = %.2f, want %.2f (goMonthLimit)", u.Limit, goMonthLimit)
	}

	// Monthly: 12/60 = 20%
	if math.Abs(u.Pct()-20) > 0.1 {
		t.Errorf("monthly pct = %.2f, want 20.00", u.Pct())
	}

	// 5h window: 6/12 = 50%
	var fiveH *schema.UsageWindow
	for i := range u.Windows {
		if u.Windows[i].Key == "5h" {
			fiveH = &u.Windows[i]
		}
	}
	if fiveH == nil {
		t.Fatal("missing 5h window")
	}
	if math.Abs(fiveH.Pct()-50) > 0.1 {
		t.Errorf("5h pct = %.2f, want 50.00", fiveH.Pct())
	}
}

func TestFetchAll_SharedSubscriptionEstimateForZeroCostProvider(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	db := newTestDB(t)
	start, _ := currentMonthMillis(t)
	mid := start + 1000

	err := testutil.SeedDB(db, []testutil.MessageRow{
		{ID: "m1", SessionID: "s1", CreatedAt: mid, UpdatedAt: mid, Role: "assistant", Cost: 0, Provider: "openai", TokensIn: 800, TokensOut: 200},
		{ID: "m2", SessionID: "s1", CreatedAt: mid + 100, UpdatedAt: mid + 100, Role: "assistant", Cost: 6, Provider: "opencode-go", TokensIn: 100, TokensOut: 100},
	})
	if err != nil {
		t.Fatalf("SeedDB: %v", err)
	}

	if err := config.SetKey("provider.opencode-go.limit", "10"); err != nil {
		t.Fatalf("SetKey: %v", err)
	}

	p := NewFromDB(db)
	ctx := context.Background()
	got, err := p.FetchAll(ctx)
	if err != nil {
		t.Fatalf("FetchAll error = %v, want nil", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d providers, want 2", len(got))
	}

	byProvider := map[string]*schema.Usage{}
	for _, u := range got {
		byProvider[u.Provider] = u
	}

	openai := byProvider["opencode/openai"]
	if openai == nil {
		t.Fatal("missing opencode/openai usage")
	}
	if openai.Limit != 10 {
		t.Fatalf("openai limit = %.2f, want 10.00", openai.Limit)
	}
	if openai.Warning == "" {
		t.Fatal("expected openai warning for estimated shared subscription usage")
	}
	if math.Abs(openai.Pct()-50) > 0.01 {
		t.Fatalf("openai pct = %.2f, want 50.00", openai.Pct())
	}
	if len(openai.Windows) != 1 {
		t.Fatalf("openai windows = %d, want 1 (zero-cost estimated only gets monthly)", len(openai.Windows))
	}
}
