package app

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"dekart/src/proto"
	"dekart/src/server/dekart"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/golang-migrate/migrate/v4"
	migratesqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	gproto "google.golang.org/protobuf/proto"
)

const (
	verifiedLoginIssuer   = "https://accounts.example.com"
	verifiedLoginAudience = "dekart-client-id"
)

// verifiedLoginInstance is a real Dekart handler on a migrated SQLite database behind verified login.
type verifiedLoginInstance struct {
	t      *testing.T
	server *httptest.Server
	key    *rsa.PrivateKey
}

func newVerifiedLoginInstance(t *testing.T) *verifiedLoginInstance {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	jwks, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kid": "test-kid", "kty": "RSA", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
	}}})
	require.NoError(t, err)
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(jwksServer.Close)

	t.Setenv("DEKART_REQUIRE_VERIFIED_LOGIN", "1")
	t.Setenv("DEKART_VERIFIED_LOGIN_JWKS_URL", jwksServer.URL)
	t.Setenv("DEKART_VERIFIED_LOGIN_ISSUER", verifiedLoginIssuer)
	t.Setenv("DEKART_VERIFIED_LOGIN_AUDIENCE", verifiedLoginAudience)
	staticDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><title>Dekart</title></html>"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(staticDir, "app.js"), []byte("console.log(1)"), 0o600))
	t.Setenv("DEKART_STATIC_FILES", staticDir)
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ADMIN", "admin@example.com")
	t.Setenv("DEKART_DEFAULT_WORKSPACE_ROLE", "viewer")
	t.Setenv("DEKART_SQLITE_DB_PATH", "")
	t.Setenv("DEKART_CLOUD", "")
	t.Setenv("DEKART_ALLOW_WORKSPACE_CREATION", "")
	t.Setenv("DEKART_LICENSE_KEY", "")

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

	httpServer := Configure(dekart.NewServer(db, nil, nil), db)
	server := httptest.NewServer(httpServer.Handler)
	t.Cleanup(server.Close)
	return &verifiedLoginInstance{t: t, server: server, key: key}
}

// token returns the ID token the login layer forwards for the email.
func (i *verifiedLoginInstance) token(email string, overrides map[string]any) string {
	claims := jwt.MapClaims{
		"iss": verifiedLoginIssuer, "aud": verifiedLoginAudience,
		"email": email, "email_verified": true,
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	for key, value := range overrides {
		claims[key] = value
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-kid"
	signed, err := token.SignedString(i.key)
	require.NoError(i.t, err)
	return signed
}

func (i *verifiedLoginInstance) do(method, path, token string, body []byte, headers map[string]string) (int, string, http.Header) {
	req, err := http.NewRequest(method, i.server.URL+path, bytes.NewReader(body))
	require.NoError(i.t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(i.t, err)
	defer res.Body.Close()
	payload, err := io.ReadAll(res.Body)
	require.NoError(i.t, err)
	return res.StatusCode, string(payload), res.Header
}

// mcp calls an MCP tool and returns the HTTP status and body.
func (i *verifiedLoginInstance) mcp(token, tool string, arguments map[string]any, headers map[string]string) (int, string) {
	body, err := json.Marshal(map[string]any{"name": tool, "arguments": arguments})
	require.NoError(i.t, err)
	status, payload, _ := i.do(http.MethodPost, "/api/v1/mcp/call", token, body, headers)
	return status, payload
}

// join makes the first request of an identity, which adds it to the default workspace.
func (i *verifiedLoginInstance) join(token string) {
	status, _, _ := i.do(http.MethodGet, "/api/v1/mcp/tools", token, nil, nil)
	require.Equal(i.t, http.StatusOK, status)
}

// grpc calls a Dekart gRPC-web method and returns the grpc-status.
func (i *verifiedLoginInstance) grpc(token, method string, request gproto.Message) string {
	payload, err := gproto.Marshal(request)
	require.NoError(i.t, err)
	frame := append([]byte{0, 0, 0, 0, 0}, payload...)
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	status, body, headers := i.do(http.MethodPost, "/Dekart/"+method, token, frame, map[string]string{"Content-Type": "application/grpc-web+proto"})
	require.Equal(i.t, http.StatusOK, status)
	if grpcStatus := headers.Get("Grpc-Status"); grpcStatus != "" {
		return grpcStatus
	}
	// The status of a successful call arrives in the trailers frame at the end of the body.
	if _, trailer, found := strings.Cut(body, "grpc-status:"); found {
		grpcStatus, _, _ := strings.Cut(trailer, "\r\n")
		return strings.TrimSpace(grpcStatus)
	}
	i.t.Fatalf("no grpc-status in response %q", body)
	return ""
}

func (i *verifiedLoginInstance) createReport(token string) string {
	status, payload := i.mcp(token, "create_report", map[string]any{}, nil)
	require.Equal(i.t, http.StatusOK, status, payload)
	var response struct {
		Result struct {
			Report struct {
				ID string `json:"id"`
			} `json:"report"`
		} `json:"result"`
	}
	require.NoError(i.t, json.Unmarshal([]byte(payload), &response))
	return response.Result.Report.ID
}

// Anyone without a valid forwarded identity is rejected, whatever else the request claims.
func TestVerifiedLoginRejectsMissingForgedAndExpiredLogins(t *testing.T) {
	instance := newVerifiedLoginInstance(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	forged := &verifiedLoginInstance{t: t, key: otherKey}
	forgedIdentity := map[string]string{
		"X-Dekart-Claim-Email":    "admin@example.com",
		"Sf-Context-Current-User": "admin",
		"X-Dekart-Playground":     "true",
		"X-Forwarded-Email":       "admin@example.com",
	}
	logins := map[string]string{
		"missing":                  "",
		"forged":                   forged.token("admin@example.com", nil),
		"expired":                  instance.token("admin@example.com", map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}),
		"wrong audience":           instance.token("admin@example.com", map[string]any{"aud": "another-client"}),
		"email unproven":           instance.token("admin@example.com", map[string]any{"email_verified": false}),
		"email verified as string": instance.token("admin@example.com", map[string]any{"email_verified": "true"}),
	}

	for name, token := range logins {
		t.Run(name, func(t *testing.T) {
			for _, path := range []string{"/api/v1/mcp/tools", "/", "/reports/00000000-0000-0000-0000-000000000000"} {
				status, _, _ := instance.do(http.MethodGet, path, token, nil, forgedIdentity)
				require.Equal(t, http.StatusUnauthorized, status, path)
			}
			for _, path := range []string{"/", "/app.js", "/reports/00000000-0000-0000-0000-000000000000"} {
				status, _, _ := instance.do(http.MethodOptions, path, token, nil, forgedIdentity)
				require.Equal(t, http.StatusUnauthorized, status, "OPTIONS "+path)
			}
			status, _ := instance.mcp(token, "create_report", map[string]any{}, forgedIdentity)
			require.Equal(t, http.StatusUnauthorized, status)
			status, _, _ = instance.do(http.MethodPost, "/Dekart/GetEnv", token, []byte{0, 0, 0, 0, 0}, map[string]string{"Content-Type": "application/grpc-web+proto"})
			require.Equal(t, http.StatusUnauthorized, status)
		})
	}
}

// Roles of the workspace engine apply to the verified identity: viewer reads only, editor edits, private maps stay private.
func TestVerifiedLoginAppliesWorkspaceRoles(t *testing.T) {
	instance := newVerifiedLoginInstance(t)
	admin := instance.token("admin@example.com", nil)
	member := instance.token("member@example.com", nil)
	instance.join(admin)
	instance.join(member)

	reportID := instance.createReport(admin)

	// A private map is hidden from other members of the workspace.
	status, _ := instance.mcp(member, "get_report_properties", map[string]any{"report_id": reportID}, nil)
	require.Equal(t, http.StatusNotFound, status)

	// A viewer reads a shared map but cannot edit it or create maps, even when asking for the playground.
	require.Equal(t, "0", instance.grpc(admin, "SetDiscoverable", &proto.SetDiscoverableRequest{ReportId: reportID, Discoverable: true, AllowEdit: true}))
	status, _ = instance.mcp(member, "get_report_properties", map[string]any{"report_id": reportID}, nil)
	require.Equal(t, http.StatusOK, status)
	status, _ = instance.mcp(member, "update_report_title", map[string]any{"report_id": reportID, "title": "viewer edit"}, nil)
	require.Equal(t, http.StatusForbidden, status)
	status, _ = instance.mcp(member, "create_report", map[string]any{}, map[string]string{"X-Dekart-Playground": "true"})
	require.Equal(t, http.StatusForbidden, status)

	// A forged identity header cannot lift the viewer to the admin.
	status, _ = instance.mcp(member, "update_report_title", map[string]any{"report_id": reportID, "title": "forged"}, map[string]string{"X-Dekart-Claim-Email": "admin@example.com"})
	require.Equal(t, http.StatusForbidden, status)

	// After promotion the same login edits the map and creates maps.
	require.Equal(t, "0", instance.grpc(admin, "UpdateWorkspaceUser", &proto.UpdateWorkspaceUserRequest{
		Email: "member@example.com", UserUpdateType: proto.UpdateWorkspaceUserRequest_USER_UPDATE_TYPE_UPDATE, Role: proto.UserRole_ROLE_EDITOR,
	}))
	status, _ = instance.mcp(member, "update_report_title", map[string]any{"report_id": reportID, "title": "editor edit"}, nil)
	require.Equal(t, http.StatusOK, status)
	instance.createReport(member)
}

// Browsers send CORS preflights without credentials, so the gRPC-web and API preflights stay answerable.
func TestVerifiedLoginAnswersCORSPreflights(t *testing.T) {
	instance := newVerifiedLoginInstance(t)
	status, _, _ := instance.do(http.MethodOptions, "/api/v1/mcp/call", "", nil, map[string]string{
		"Origin": "http://localhost:3000", "Access-Control-Request-Method": "POST",
	})
	require.Less(t, status, http.StatusBadRequest)
	status, _, _ = instance.do(http.MethodOptions, "/Dekart/GetEnv", "", nil, map[string]string{
		"Origin": "http://localhost:3000", "Access-Control-Request-Method": "POST",
		"Access-Control-Request-Headers": "content-type,x-grpc-web",
	})
	require.Less(t, status, http.StatusBadRequest)
}
