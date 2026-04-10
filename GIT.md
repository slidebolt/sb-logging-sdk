# Git Workflow for sb-logging-sdk

This repository contains the Slidebolt Logging SDK, providing the client interfaces and implementation for interacting with the Slidebolt logging system.

## Dependencies
- **Internal:**
  - `sb-messenger-sdk`: Used for sending log messages over the messaging system.
- **External:** 
  - `github.com/nats-io/nats.go`: Communication with NATS.

## Build Process
- **Type:** Pure Go Library (Shared Module).
- **Consumption:** Imported as a module dependency in other Go projects via `go.mod`.
- **Artifacts:** No standalone binary or executable is produced.
- **Validation:** 
  - Validated through unit tests: `go test -v ./...`
  - Validated by its consumers during their respective build/test cycles.

## Pre-requisites & Publishing
As a logging SDK, `sb-logging-sdk` should be updated whenever its internal messaging dependency (`sb-messenger-sdk`) is updated.

**Before publishing:**
1. Determine current tag: `git tag | sort -V | tail -n 1`
2. Ensure all local tests pass: `go test -v ./...`

**Publishing Order:**
1. Ensure `sb-messenger-sdk` is tagged and pushed (e.g., `v1.0.4`).
2. Update `sb-logging-sdk/go.mod` to reference the latest `sb-messenger-sdk` tag.
3. Determine next semantic version for `sb-logging-sdk` (e.g., `v1.0.0`).
4. Commit and push the changes to `main`.
5. Tag the repository: `git tag v1.0.0`.
6. Push the tag: `git push origin v1.0.0`.

## Update Workflow & Verification
1. **Modify:** Update logging client logic in `client.go` or `logging.go`.
2. **Verify Local:**
   - Run `go mod tidy`.
   - Run `go test ./...`.
3. **Commit:** Ensure the commit message clearly describes the SDK change.
4. **Tag & Push:** (Follow the Publishing Order above).
