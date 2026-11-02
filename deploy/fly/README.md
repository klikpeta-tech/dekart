# Deploying the production line to Fly (app klikpeta-geo)

`deploy/fly/fly.toml` runs the fork's `release/v0.26-klikpeta` build on the existing app `klikpeta-geo` (geo.klikpeta.com).
It keeps the app's volume `dekart_data`, SQLite data, Tigris bucket and every environment setting of the stock-image setup;
only `[build]` changes, from `dekartxyz/dekart:0.26` to this repository's `Dockerfile`.

The release line keeps Dekart's own migration track exactly at upstream v0.26.0 (last migration `000052_report_history_retention`);
fork-owned tables live in a separate track, see [docs/fork-branches.md](../../docs/fork-branches.md).
Check that before every deploy; the command must print nothing:

```sh
git diff --stat v0.26.0 HEAD -- sqlite/migrations migrations
```

## Before deploying

Every `flyctl` call goes through the Automic Vault wrapper and needs approval.

```sh
flyctl volumes list --app klikpeta-geo                       # note the dekart_data volume id
flyctl volumes snapshots create <volume-id> --app klikpeta-geo
flyctl volumes snapshots list <volume-id>                    # note the new snapshot id
```

Confirm the newest `sqlite-backups/` object in the Tigris bucket `klikpeta-geo-backup` is recent (Dekart writes one every 5 minutes
while the machine runs). Record both ids with the deploy.

## Deploy

From the repository root, on the commit to deploy (the Dockerfile copies an `.npmrc` that upstream CI writes; an empty file is enough
because no package uses the private registry, and it stays untracked):

```sh
touch .npmrc
flyctl deploy . --config deploy/fly/fly.toml --dockerfile Dockerfile --app klikpeta-geo --ha=false --remote-only
```

`DEKART_LICENSE_KEY` stays set as an app secret. This build ignores it, and the stock image needs it for a rollback until the key expires.

## Verify

- `flyctl logs --app klikpeta-geo --no-tail` shows `License key checks are disabled in this build`.
- Sign in with Google at https://geo.klikpeta.com, open a saved map, run a query on the saved connection, check Workspace members.
- No "upgrade to SSO" or free-key prompt appears.

## Roll back

The schema is unchanged, so rolling back is an image swap to the stock image:

```sh
flyctl deploy --app klikpeta-geo --config deploy/fly/fly.toml --image dekartxyz/dekart:0.26 --ha=false
```

This works while the licence key in `DEKART_LICENSE_KEY` is valid (until 2026-11-29). The data stays on the same volume.

Restoring data from the pre-deploy snapshot is a manual, captain-approved recovery done case by case. It is never scripted here,
and never done by deleting a volume.
