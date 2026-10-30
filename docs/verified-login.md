# Verified login (keyless identity mode)

Verified login gives a self-hosted Dekart distinct, verified user identities without a license key. A login layer in
front of Dekart (for example [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/) with Google) signs users in and
forwards the user's OIDC ID token. Dekart verifies the token itself and feeds the verified email into the existing
workspace, role (admin / editor / viewer) and sharing engine.

It is a separate mode. It is not one of the license-gated SSO flags (`DEKART_REQUIRE_OIDC`, `DEKART_REQUIRE_GOOGLE_OAUTH`,
`DEKART_REQUIRE_IAP`, `DEKART_REQUIRE_AMAZON_OIDC`), it does not change `src/server/app/licensecheck.go`, and it needs no
`DEKART_LICENSE_KEY`.

## What Dekart checks

On every request the forwarded token (`Authorization: Bearer <id token>`) must be:

- signed by a key in the configured JWKS (RS256/384/512 or ES256/384/512; `none` and HMAC algorithms are rejected),
- unexpired, and carrying an `exp` claim (a token without expiry is rejected),
- issued by `DEKART_VERIFIED_LOGIN_ISSUER` and addressed to `DEKART_VERIFIED_LOGIN_AUDIENCE`,
- carrying an `email` claim with `email_verified: true`,
- for an email or domain on the allow lists, when either list is set.

A request that fails any check gets `401 Unauthorized`. There is no anonymous fallback and no playground access: every
request except `GET /health` needs a valid token, whatever the HTTP method. The only other exception is a CORS preflight
(`OPTIONS`) answered by the gRPC or `/api/v1/` handlers, which return headers only. In this mode Dekart ignores and removes the client-controlled identity
inputs of other modes (`X-Dekart-Claim-Email`, `Sf-Context-Current-User`, `X-Dekart-Playground`), and refuses to start
with `DEKART_DEV_CLAIMS=1` or any other `DEKART_REQUIRE_*` identity mode.

## Environment variables

| Variable | Required | Meaning |
| --- | --- | --- |
| `DEKART_REQUIRE_VERIFIED_LOGIN` | yes | `1` enables the mode. |
| `DEKART_VERIFIED_LOGIN_JWKS_URL` | yes | URL of the issuer's public keys. Google: `https://www.googleapis.com/oauth2/v3/certs`. |
| `DEKART_VERIFIED_LOGIN_ISSUER` | yes | Expected `iss`. Google: `https://accounts.google.com`. |
| `DEKART_VERIFIED_LOGIN_AUDIENCE` | yes | Expected `aud`. For Google, the OAuth client id of the login layer. |
| `DEKART_VERIFIED_LOGIN_ALLOWED_EMAILS` | no | Comma separated emails that may sign in (case insensitive). |
| `DEKART_VERIFIED_LOGIN_ALLOWED_DOMAINS` | no | Comma separated email domains that may sign in (exact match, no sub-domains). |

When both allow lists are empty, every verified email that passes the login layer is accepted. With the default
workspace, every accepted identity joins it with `DEKART_DEFAULT_WORKSPACE_ROLE`, so set an allow list here or in the
login layer unless any signed-in account should see workspace-shared maps. Use `DEKART_DEFAULT_WORKSPACE_ADMIN` and
`DEKART_DEFAULT_WORKSPACE_ROLE=viewer` for the usual setup.

## Deployment shape

Dekart must be reachable only through the login layer: bind it to `127.0.0.1` (or a private network) and expose only the
login layer's port. The verification protects against forged tokens, not against a network path that lets users reach
Dekart without the login layer.

Example with oauth2-proxy and Google, Dekart on `127.0.0.1:8080`:

```sh
oauth2-proxy \
  --provider=google \
  --client-id="$GOOGLE_CLIENT_ID" --client-secret="$GOOGLE_CLIENT_SECRET" --cookie-secret="$COOKIE_SECRET" \
  --redirect-url=https://dekart.example.com/oauth2/callback \
  --http-address=0.0.0.0:4180 --upstream=http://127.0.0.1:8080 \
  --pass-authorization-header=true --cookie-refresh=50m --cookie-secure=true \
  --email-domain='*'
```

```sh
DEKART_REQUIRE_VERIFIED_LOGIN=1
DEKART_VERIFIED_LOGIN_JWKS_URL=https://www.googleapis.com/oauth2/v3/certs
DEKART_VERIFIED_LOGIN_ISSUER=https://accounts.google.com
DEKART_VERIFIED_LOGIN_AUDIENCE=<the same Google client id>
DEKART_DEFAULT_WORKSPACE_ADMIN=admin@example.com
DEKART_DEFAULT_WORKSPACE_ROLE=viewer
```

Google ID tokens last one hour. Keep the login layer's session refresh shorter than that (`--cookie-refresh` above), so the
forwarded token does not expire while the user is signed in; an expired token is rejected with 401 until the login layer
forwards a fresh one.

## Limits

- The `Authorization` header carries the ID token, so device-flow and MCP bearer tokens are not available in this mode.
- Dekart does not hold the user's Google access token, so connections that pass the user's own Google credentials to a
  warehouse (BigQuery user OAuth) are not available. Postgres connections work as usual.

## Modifications in this fork (AGPL-3.0)

This fork of [dekart-xyz/dekart](https://github.com/dekart-xyz/dekart) is distributed under the same AGPL-3.0 terms and keeps
the upstream `LICENSE` and `LICENSE-COMMERCIAL` files unchanged. Changes relative to upstream, all in this repository's git
history:

- `src/server/user/verifiedlogin.go`, `src/server/user/oidc.go`, `src/server/user/claims.go`, `src/server/app/app.go`:
  the verified login mode described above. The license checks in `src/server/app/licensecheck.go` are unchanged.
- `src/server/dekart/workspace.go`: role, member and invite queries resolve rows that share a timestamp to the latest one.
- `.github/workflows`: upstream-only build jobs run only in `dekart-xyz/dekart`; a PR declaration check was added.
