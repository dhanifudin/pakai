package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCredentialsFromPiAuth(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"other":{"type":"api_key"},"openai-codex":{"type":"oauth","access":"access","refresh":"refresh","accountId":"account","expires":1}}`), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := (&Provider{credsPath: path, source: "pi"}).readCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" || got.AccountID != "account" {
		t.Fatalf("unexpected credentials: %+v", got)
	}
}

func TestReadCredentialsFromCodexCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(`{"tokens":{"access_token":"access","refresh_token":"refresh","account_id":"account"}}`), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := (&Provider{credsPath: path, source: "codex"}).readCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access" || got.RefreshToken != "refresh" || got.AccountID != "account" {
		t.Fatalf("unexpected credentials: %+v", got)
	}
}
