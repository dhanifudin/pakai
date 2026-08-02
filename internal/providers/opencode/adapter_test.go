package opencode

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
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

	now := time.Now()
	recentMillis := now.UnixMilli()
	olderMillis := start + 1000

	err := testutil.SeedDB(db, []testutil.MessageRow{
		{ID: "m1", SessionID: "s1", CreatedAt: olderMillis, UpdatedAt: olderMillis, Role: "assistant", Cost: 6.0, Provider: "opencode-go", TokensIn: 100, TokensOut: 100},
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

	var fiveH *schema.UsageWindow
	for i := range u.Windows {
		if u.Windows[i].Key == "5h" {
			fiveH = &u.Windows[i]
		}
	}
	if fiveH == nil {
		t.Fatal("missing 5h window")
	}
	wantFiveHPct := 50.0
	if olderMillis >= now.Add(-5*time.Hour).UnixMilli() {
		wantFiveHPct = 100
	}
	if math.Abs(fiveH.Pct()-wantFiveHPct) > 0.1 {
		t.Errorf("5h pct = %.2f, want %.2f", fiveH.Pct(), wantFiveHPct)
	}
}

func TestReadZenGoKey_Missing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if got := readZenGoKey(); got != "" {
		t.Errorf("expected empty key for missing auth.json, got %q", got)
	}
}

func TestReadZenGoKey_Valid(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	authPath := filepath.Join(dir, ".local", "share", "opencode", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]authEntry{
		"opencode-go": {Type: "api", Key: "sk-test-123"},
	})
	if err := os.WriteFile(authPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	got := readZenGoKey()
	if got != "sk-test-123" {
		t.Errorf("got %q, want sk-test-123", got)
	}
}

func TestZenLimitNameToKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"5h", "5h"},
		{"5-hour", "5h"},
		{"five-hour", "5h"},
		{"weekly", "weekly"},
		{"week", "weekly"},
		{"monthly", "monthly"},
		{"month", "monthly"},
		{"unknown", "monthly"},
		{"", "monthly"},
	}
	for _, c := range cases {
		got := zenLimitNameToKey(c.in)
		if got != c.want {
			t.Errorf("zenLimitNameToKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestProbeZenGo_Exhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain;charset=UTF-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"CreditsError","message":"Insufficient balance."}}`))
	}))
	defer srv.Close()

	// Temporarily point probeZenGo at the test server by using a custom RoundTripper.
	// Since probeZenGo hardcodes the URL, we test it indirectly via the zenProbe struct.
	// Instead, test the parsing logic directly by calling probeZenGo with a test server URL.
	got := probeZenGoURL(context.Background(), "sk-test", srv.URL+"/chat/completions")
	if !got.exhausted {
		t.Errorf("expected exhausted=true for CreditsError response")
	}
	if got.windowKey != "" {
		t.Errorf("expected no windowKey for CreditsError, got %q", got.windowKey)
	}
}

func TestProbeZenGo_WindowLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("retry-after", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"type":"error","error":{"type":"GoUsageLimitError","metadata":{"limitName":"monthly","workspace":"wrk_123"}}}`))
	}))
	defer srv.Close()

	got := probeZenGoURL(context.Background(), "sk-test", srv.URL+"/chat/completions")
	if got.exhausted {
		t.Errorf("expected exhausted=false for GoUsageLimitError")
	}
	if got.windowKey != "monthly" {
		t.Errorf("got windowKey=%q, want monthly", got.windowKey)
	}
	if got.retryAfterSec != 3600 {
		t.Errorf("got retryAfterSec=%d, want 3600", got.retryAfterSec)
	}
}

func TestFetchAll_SharedSubscriptionEstimateForZeroCostProvider(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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

func writeOpenCodeAuth(t *testing.T, home string, entries map[string]authEntry) {
	t.Helper()
	authDir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(entries)
	if err := os.WriteFile(filepath.Join(authDir, "auth.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestReadOpenAICreds_Missing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, _, ok := readOpenAICreds()
	if ok {
		t.Error("expected ok=false for missing auth.json")
	}
}

func TestReadOpenAICreds_Valid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeOpenCodeAuth(t, home, map[string]authEntry{
		"openai": {Type: "oauth", Access: "tok-abc", AccountID: "acct-123"},
	})
	access, accountID, ok := readOpenAICreds()
	if !ok {
		t.Fatal("expected ok=true")
	}
	if access != "tok-abc" {
		t.Errorf("access = %q, want tok-abc", access)
	}
	if accountID != "acct-123" {
		t.Errorf("accountID = %q, want acct-123", accountID)
	}
}

func TestReadOpenAICreds_APIType(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeOpenCodeAuth(t, home, map[string]authEntry{
		"openai": {Type: "api", Key: "sk-xyz"},
	})
	_, _, ok := readOpenAICreds()
	if ok {
		t.Error("expected ok=false for non-oauth type")
	}
}

func TestFetchOpenAIUsageWindows_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"rate_limit":{"primary_window":{"used_percent":42},"secondary_window":{"used_percent":15,"reset_at":9999999}}}`))
	}))
	defer srv.Close()

	old := openaiUsageURL
	openaiUsageURL = srv.URL
	defer func() { openaiUsageURL = old }()

	windows, err := fetchOpenAIUsageWindows(context.Background(), "tok", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(windows) != 2 {
		t.Fatalf("got %d windows, want 2", len(windows))
	}
	if windows[0].Key != "5h" || windows[0].Used != 42 {
		t.Errorf("5h window = %+v", windows[0])
	}
	if windows[1].Key != "weekly" || windows[1].Used != 15 {
		t.Errorf("weekly window = %+v", windows[1])
	}
}

func TestFetchOpenAIUsageWindows_Expired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	old := openaiUsageURL
	openaiUsageURL = srv.URL
	defer func() { openaiUsageURL = old }()

	_, err := fetchOpenAIUsageWindows(context.Background(), "tok", "")
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired error, got: %v", err)
	}
}

func TestFetchAll_OpenAIUsesRealAPI(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	writeOpenCodeAuth(t, home, map[string]authEntry{
		"openai": {Type: "oauth", Access: "tok", AccountID: "acct"},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"rate_limit":{"primary_window":{"used_percent":10},"secondary_window":{"used_percent":5,"reset_at":9999999}}}`))
	}))
	defer srv.Close()

	old := openaiUsageURL
	openaiUsageURL = srv.URL
	defer func() { openaiUsageURL = old }()

	db := newTestDB(t)
	start, _ := currentMonthMillis(t)
	mid := start + 1000
	if err := testutil.SeedDB(db, []testutil.MessageRow{
		{ID: "m1", SessionID: "s1", CreatedAt: mid, UpdatedAt: mid, Role: "assistant", Cost: 0, Provider: "openai", TokensIn: 500, TokensOut: 100},
	}); err != nil {
		t.Fatalf("SeedDB: %v", err)
	}

	p := NewFromDB(db)
	got, err := p.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll error: %v", err)
	}

	var openai *schema.Usage
	for _, u := range got {
		if u.Provider == "opencode/openai" {
			openai = u
		}
	}
	if openai == nil {
		t.Fatal("missing opencode/openai in results")
	}
	if openai.Status != schema.StatusOK {
		t.Errorf("status = %v, want OK", openai.Status)
	}
	if openai.Unit != "percent" {
		t.Errorf("unit = %q, want percent", openai.Unit)
	}
	if len(openai.Windows) != 2 {
		t.Fatalf("got %d windows, want 2 (5h + weekly)", len(openai.Windows))
	}
	if openai.Warning != "" {
		t.Errorf("expected no warning (real API used), got: %q", openai.Warning)
	}
}
