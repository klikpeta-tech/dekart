package user

import (
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// verifiedLoginTokenHeader carries the login layer's ID token (oauth2-proxy --pass-authorization-header).
const verifiedLoginTokenHeader = "Authorization"

// splitList parses a comma separated env value into trimmed non-empty entries.
func splitList(value string) []string {
	entries := []string{}
	for _, entry := range strings.Split(value, ",") {
		if entry = strings.TrimSpace(entry); entry != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}

// emailAllowed reports whether the email matches the allowed emails or domains; empty lists allow every email.
func emailAllowed(email string, allowedEmails, allowedDomains []string) bool {
	if len(allowedEmails) == 0 && len(allowedDomains) == 0 {
		return true
	}
	for _, allowed := range allowedEmails {
		if strings.EqualFold(email, allowed) {
			return true
		}
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	for _, domain := range allowedDomains {
		if strings.EqualFold(email[at+1:], domain) {
			return true
		}
	}
	return false
}

// emailVerified accepts the OIDC boolean claim and the string form some providers emit.
func emailVerified(value interface{}) bool {
	switch verified := value.(type) {
	case bool:
		return verified
	case string:
		return verified == "true"
	}
	return false
}

// validateVerifiedLogin returns claims only for a signed, unexpired ID token of an allowed, verified email; otherwise nil.
// The request never falls back to an anonymous identity.
func (c ClaimsCheck) validateVerifiedLogin(r *http.Request) *Claims {
	tokenString, ok := normalizeBearerToken(r.Header.Get(verifiedLoginTokenHeader))
	if !ok {
		log.Warn().Msg("Verified login token is missing or malformed")
		return nil
	}
	claims, err := c.oidcJWT.verifyToken(tokenString)
	if err != nil {
		log.Warn().Err(err).Msg("Verified login token verification failed")
		return nil
	}
	// jwt only checks exp when present, a token that never expires is not acceptable.
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		log.Warn().Msg("Verified login token has no valid expiry")
		return nil
	}
	email, _ := claims[defaultOIDCEmailClaim].(string)
	if email == "" {
		log.Warn().Msg("Verified login token has no email claim")
		return nil
	}
	if !emailVerified(claims["email_verified"]) {
		log.Warn().Msg("Verified login token email is not verified")
		return nil
	}
	if !emailAllowed(email, splitList(c.VerifiedLoginAllowedEmails), splitList(c.VerifiedLoginAllowedDomains)) {
		log.Warn().Msg("Verified login email is not allowed")
		return nil
	}
	return &Claims{Email: email}
}

// RemoveUntrustedIdentityHeaders drops client-controlled identity inputs that other modes honor.
func RemoveUntrustedIdentityHeaders(r *http.Request) {
	// Playground gives anonymous access to a shared workspace, so a signed-in user must not select it.
	r.Header.Del("X-Dekart-Playground")
	r.Header.Del("X-Dekart-Claim-Email")
	r.Header.Del("Sf-Context-Current-User")
}
