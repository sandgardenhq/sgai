# Managing dependency updates with Dependabot

Hey there! This project uses GitHub Dependabot to propose Go module dependency updates as pull requests.

This guide shows:

- Where Dependabot is configured
- What it is configured to update
- How to review and validate Dependabot pull requests in this repository

## Prerequisites

- Access to create/merge pull requests in the repository
- Go tooling installed (the repository uses `go.mod`/`go.sum`)

## Where Dependabot is configured

Dependabot configuration lives in [`.github/dependabot.yml`](../../.github/dependabot.yml).

The configuration:

- Uses `version: 2`
- Enables the `gomod` package ecosystem
- Targets the repository root directory (`/`)
- Runs on a `weekly` schedule
- Allows updates for all dependency types (`allow: dependency-type: "all"`)

## Review a Dependabot pull request

1. Open the Dependabot pull request.
2. Review the files changed. Go module updates typically change `go.mod` and `go.sum`.
3. Run the same checks that CI runs for pull requests:

   ```sh
   make build
   make test
   ```

   At this point, you should see both commands complete successfully.

## What CI runs for Dependabot pull requests

GitHub Actions runs the `Go` workflow in [`.github/workflows/go.yml`](../../.github/workflows/go.yml) for pull requests to `main`.

That workflow:

- Builds via `make build`
- Tests via `make test`
- Runs on `ubuntu-latest` and `macos-latest`

## Troubleshooting

### `go.sum` changes look noisy

`go.sum` records cryptographic hashes for module content and `go.mod` files. A Dependabot update can add or change many entries.

If `make build` and `make test` succeed and the pull request looks reasonable, `go.sum` churn is expected for Go module updates.

## Next steps

- Keep the Dependabot settings in [`.github/dependabot.yml`](../../.github/dependabot.yml) aligned with the repository’s update policy (directory, schedule, ecosystems).
- Use the CI workflow results from [`.github/workflows/go.yml`](../../.github/workflows/go.yml) as the baseline signal for whether an update is safe to merge.
