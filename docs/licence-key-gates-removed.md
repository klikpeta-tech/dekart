# Licence-key gates removed (fork)

This fork of [dekart-xyz/dekart](https://github.com/dekart-xyz/dekart) starts and runs without a `DEKART_LICENSE_KEY`.

## What changed

`RequireValidStartupLicense` in `src/server/app/licensecheck.go` no longer validates or requires a key. It logs that the check is
disabled and returns an empty state (not required, no expiry). Nothing else in the server changed. As a result:

- `DEKART_REQUIRE_GOOGLE_OAUTH`, `DEKART_REQUIRE_OIDC`, `DEKART_REQUIRE_IAP` and `DEKART_REQUIRE_AMAZON_OIDC` start without a key.
- Postgres metadata (`DEKART_POSTGRES_*`) starts without a key.
- A missing, invalid or expired key never stops startup and never makes a workspace read-only, so the "license key expired"
  banner and "Extend Key" button never appear. The runtime expiry code in `src/server/dekart/server.go` is unchanged and stays
  inert because the state it receives has no expiry.
- The role, workspace and sharing engine is unchanged. The plan label stays `COMMUNITY` (or `PREMIUM` when a key is set); both
  count as self-hosted and behave the same.

`ValidateStartupLicense`, `enabledSSOEnvVars` and their tests are kept as upstream wrote them but are no longer called at startup.
The call site in `src/server/main.go` and the `AllowExpiredKeyStartup` option are also unchanged, so
`DEKART_DEV_ALLOW_EXPIRED_LICENSE_KEY_STARTUP` is now inert. This keeps the diff to one function body, so merges from upstream stay small.

## What stays the same

Authentication itself is not weakened. Each stock mode still needs its own configuration (for Google OAuth,
`DEKART_GOOGLE_OAUTH_CLIENT_ID` and `DEKART_GOOGLE_OAUTH_SECRET`), and requests without a valid identity are still refused.
`src/server/app/licensegates_test.go` checks startup without a key for each mode, that an absent, invalid or expired key does not lock
a workspace, and that anonymous callers are refused when Google OAuth is required.

## Why

The fork runs a self-hosted instance with Dekart's built-in Google sign-in and per-user roles, which upstream gates behind a
free-to-request key. The gate is a startup check, not part of the role engine, so it can be neutralised in one place.

## AGPL-3.0

Upstream marks bypassing this check as a modification under the AGPL that requires publishing the changed source. This fork is
public under the same AGPL-3.0 terms, keeps the upstream `LICENSE` and `LICENSE-COMMERCIAL` files unchanged, and lists its
modifications in the git history and in this document.
