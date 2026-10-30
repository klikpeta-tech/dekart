package user

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/require"
)

const (
	verifiedLoginTestIssuer   = "https://accounts.example.com"
	verifiedLoginTestAudience = "dekart-client-id"
)

func verifiedLoginClaimsCheck(t *testing.T, jwksURL, allowedEmails, allowedDomains string) ClaimsCheck {
	return NewClaimsCheck(ClaimsCheckConfig{
		RequireVerifiedLogin:        true,
		VerifiedLoginJWKSURL:        jwksURL,
		VerifiedLoginIssuer:         verifiedLoginTestIssuer,
		VerifiedLoginAudience:       verifiedLoginTestAudience,
		VerifiedLoginAllowedEmails:  allowedEmails,
		VerifiedLoginAllowedDomains: allowedDomains,
	}, nil)
}

// validIDToken returns the claims of an ID token the login layer would forward, with overrides applied.
func validIDToken(overrides map[string]interface{}) map[string]interface{} {
	claims := map[string]interface{}{
		"iss":            verifiedLoginTestIssuer,
		"aud":            verifiedLoginTestAudience,
		"email":          "alice@example.com",
		"email_verified": true,
		"exp":            time.Now().Add(time.Hour).Unix(),
	}
	for key, value := range overrides {
		if value == nil {
			delete(claims, key)
		} else {
			claims[key] = value
		}
	}
	return claims
}

func verifiedLoginRequest(token string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mcp/call", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestVerifiedLoginAcceptsSignedUnexpiredVerifiedToken(t *testing.T) {
	key, jwksURL, _ := startJWKSServer(t)
	check := verifiedLoginClaimsCheck(t, jwksURL, "", "")

	claims := GetClaims(check.GetContext(verifiedLoginRequest(createOIDCToken(t, key, validIDToken(nil)))))

	require.NotNil(t, claims)
	require.Equal(t, "alice@example.com", claims.Email)
}

// Every invalid login must end without claims, never an anonymous identity.
func TestVerifiedLoginRejectsInvalidLogins(t *testing.T) {
	key, jwksURL, _ := startJWKSServer(t)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	check := verifiedLoginClaimsCheck(t, jwksURL, "", "")
	noneToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims(validIDToken(nil))).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	hmacToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(validIDToken(nil))).SignedString([]byte("secret"))
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
	}{
		{"missing login", ""},
		{"not a token", "not-a-jwt"},
		{"forged signature", createOIDCToken(t, otherKey, validIDToken(nil))},
		{"unsigned alg none", noneToken},
		{"symmetric alg", hmacToken},
		{"expired", createOIDCToken(t, key, validIDToken(map[string]interface{}{"exp": time.Now().Add(-time.Minute).Unix()}))},
		{"no expiry", createOIDCToken(t, key, validIDToken(map[string]interface{}{"exp": nil}))},
		{"wrong issuer", createOIDCToken(t, key, validIDToken(map[string]interface{}{"iss": "https://evil.example.com"}))},
		{"wrong audience", createOIDCToken(t, key, validIDToken(map[string]interface{}{"aud": "other-client"}))},
		{"no audience", createOIDCToken(t, key, validIDToken(map[string]interface{}{"aud": nil}))},
		{"email not verified", createOIDCToken(t, key, validIDToken(map[string]interface{}{"email_verified": false}))},
		{"email_verified as string", createOIDCToken(t, key, validIDToken(map[string]interface{}{"email_verified": "true"}))},
		{"email_verified missing", createOIDCToken(t, key, validIDToken(map[string]interface{}{"email_verified": nil}))},
		{"no email", createOIDCToken(t, key, validIDToken(map[string]interface{}{"email": nil}))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Nil(t, GetClaims(check.GetContext(verifiedLoginRequest(test.token))))
		})
	}
}

func TestVerifiedLoginAllowedEmailsAndDomains(t *testing.T) {
	key, jwksURL, _ := startJWKSServer(t)
	check := verifiedLoginClaimsCheck(t, jwksURL, "bob@other.com, Carol@Other.com", "example.com")
	tests := []struct {
		email   string
		allowed bool
	}{
		{"alice@example.com", true},
		{"alice@EXAMPLE.com", true},
		{"carol@other.com", true},
		{"dave@other.com", false},
		{"alice@sub.example.com", false},
		{"alice@example.com.evil.com", false},
	}
	for _, test := range tests {
		t.Run(test.email, func(t *testing.T) {
			token := createOIDCToken(t, key, validIDToken(map[string]interface{}{"email": test.email}))
			require.Equal(t, test.allowed, GetClaims(check.GetContext(verifiedLoginRequest(token))) != nil)
		})
	}
}

// Identity headers other modes trust must neither create an identity nor replace the verified one.
func TestVerifiedLoginIgnoresClientIdentityHeaders(t *testing.T) {
	key, jwksURL, _ := startJWKSServer(t)
	check := verifiedLoginClaimsCheck(t, jwksURL, "", "")
	headers := map[string]string{
		"X-Dekart-Claim-Email":     "admin@example.com",
		"Sf-Context-Current-User":  "admin",
		"X-Forwarded-Access-Token": createOIDCToken(t, key, validIDToken(map[string]interface{}{"email": "admin@example.com"})),
		"X-Forwarded-Email":        "admin@example.com",
		"X-Goog-IAP-JWT-Assertion": "x",
		"X-Dekart-Logged-In":       "true",
	}

	forgedOnly := verifiedLoginRequest("")
	signedIn := verifiedLoginRequest(createOIDCToken(t, key, validIDToken(nil)))
	for name, value := range headers {
		forgedOnly.Header.Set(name, value)
		signedIn.Header.Set(name, value)
	}

	require.Nil(t, GetClaims(check.GetContext(forgedOnly)))
	require.Equal(t, "alice@example.com", GetClaims(check.GetContext(signedIn)).Email)
}

func TestRemoveUntrustedIdentityHeadersDropsPlaygroundAndForgedIdentity(t *testing.T) {
	req := verifiedLoginRequest("")
	req.Header.Set("X-Dekart-Playground", "true")
	req.Header.Set("X-Dekart-Claim-Email", "admin@example.com")
	req.Header.Set("Sf-Context-Current-User", "admin")
	req.Header.Set("X-Dekart-Workspace-Id", "workspace")

	RemoveUntrustedIdentityHeaders(req)

	require.Empty(t, req.Header.Get("X-Dekart-Playground"))
	require.Empty(t, req.Header.Get("X-Dekart-Claim-Email"))
	require.Empty(t, req.Header.Get("Sf-Context-Current-User"))
	require.Equal(t, "workspace", req.Header.Get("X-Dekart-Workspace-Id"))
}
