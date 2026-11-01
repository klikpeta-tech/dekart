package app

import (
	"bytes"
	"database/sql"
	"dekart/src/server/dekart"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// Stock SSO modes and Postgres metadata must start without a license key; a failed check exits the test binary.
func TestRequireValidStartupLicenseStartsWithoutKey(t *testing.T) {
	for _, flag := range []string{"DEKART_REQUIRE_GOOGLE_OAUTH", "DEKART_REQUIRE_OIDC", "DEKART_REQUIRE_IAP", "DEKART_REQUIRE_AMAZON_OIDC"} {
		t.Run(flag, func(t *testing.T) {
			clearStartupLicenseEnv(t)
			t.Setenv(flag, "1")

			state := RequireValidStartupLicense(StartupLicenseConfig{RequireForPostgresMetadata: true})

			require.False(t, state.Required)
			require.Nil(t, state.ExpiresAt)
		})
	}
}

// startLicenseGateInstance runs the real handler with Google OAuth required on a migrated SQLite database.
func startLicenseGateInstance(t *testing.T, licenseKey string) *httptest.Server {
	clearStartupLicenseEnv(t)
	t.Setenv("DEKART_REQUIRE_GOOGLE_OAUTH", "1")
	t.Setenv("DEKART_GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("DEKART_GOOGLE_OAUTH_SECRET", "client-secret")
	// The dev claim header stands in for a signed-in Google user; requests without it stay anonymous.
	t.Setenv("DEKART_DEV_CLAIMS", "1")
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", "admin@example.com")
	t.Setenv("DEKART_SQLITE_DB_PATH", "")
	t.Setenv("DEKART_CLOUD", "")
	if licenseKey != "" {
		t.Setenv("DEKART_LICENSE_KEY", licenseKey)
	}

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "dekart.db"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	driver, err := migratesqlite3.WithInstance(db, &migratesqlite3.Config{})
	require.NoError(t, err)
	migrationsDir, err := filepath.Abs("../../../sqlite/migrations")
	require.NoError(t, err)
	m, err := migrate.NewWithDatabaseInstance("file://"+migrationsDir, "sqlite3", driver)
	require.NoError(t, err)
	require.NoError(t, m.Up())

	state := RequireValidStartupLicense(StartupLicenseConfig{})
	server := dekart.NewServerWithRuntimeLicense(db, nil, nil, dekart.RuntimeLicenseState{Required: state.Required, ExpiresAt: state.ExpiresAt})
	instance := httptest.NewServer(Configure(server, db).Handler)
	t.Cleanup(instance.Close)
	return instance
}

func callMCP(t *testing.T, instance *httptest.Server, email, tool string) (int, string) {
	body, err := json.Marshal(map[string]any{"name": tool, "arguments": map[string]any{}})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, instance.URL+"/api/v1/mcp/call", bytes.NewReader(body))
	require.NoError(t, err)
	if email != "" {
		req.Header.Set("X-Dekart-Claim-Email", email)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	var payload bytes.Buffer
	_, err = payload.ReadFrom(res.Body)
	require.NoError(t, err)
	return res.StatusCode, payload.String()
}

// An absent, invalid or expired key must never lock a workspace, and Google OAuth must still refuse anonymous callers.
func TestLicenseKeyNeverLocksWorkspacesAndAuthStaysRequired(t *testing.T) {
	keys := map[string]string{
		"no key":               "",
		"invalid key":          "not-a-license-key",
		"expired signed token": "eyJhbGciOiJSUzI1NiJ9.eyJleHAiOjF9.c2lnbmF0dXJl",
	}
	for name, key := range keys {
		t.Run(name, func(t *testing.T) {
			instance := startLicenseGateInstance(t, key)

			status, payload := callMCP(t, instance, "admin@example.com", "create_report")
			require.Equal(t, http.StatusOK, status, payload)

			status, _ = callMCP(t, instance, "", "create_report")
			require.Equal(t, http.StatusUnauthorized, status)
		})
	}
}
