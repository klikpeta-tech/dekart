# Branches of the klikpeta-tech/dekart fork

The fork keeps two lines:

| Branch | What it is |
| --- | --- |
| `main` | Tracks upstream [dekart-xyz/dekart](https://github.com/dekart-xyz/dekart) plus the fork's fixes. It moves with upstream, including new database migrations. |
| `release/v0.26-klikpeta` | What runs in production on geo.klikpeta.com: upstream tag v0.26.0 plus the same fixes ported onto it. |

## Rules for the release line

- Fixes enter as pull requests into `release/v0.26-klikpeta` (not into `main`), validated through no-mistakes and merged by the captain.
- The database schema stays exactly at v0.26.0 (`sqlite/migrations` up to `000052`, `migrations` up to `000053`). No new migration lands on this branch.
  That keeps a rollback to the stock `dekartxyz/dekart:0.26` image a plain image swap, because Dekart exits at startup when its database
  is ahead of the image's migrations. `main` already carries newer upstream migrations (for example `http_sources`, workspace `deleted_at`), so
  a database that ran `main` cannot go back to the stock image.
- Deploy files live in `deploy/fly/`, see its README.
- A fix that should reach both lines is made on both: once as a PR to `main`, once as a port to the release line (hand-resolved where the code differs).

See [licence-key-gates-removed.md](licence-key-gates-removed.md) for the licence-check change and the AGPL notes.
