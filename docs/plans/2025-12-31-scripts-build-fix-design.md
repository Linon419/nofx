# Scripts Build Fix Design
Date: 2025-12-31
Status: Accepted

## Goal
Avoid duplicate main() build errors in the scripts directory when running `go test ./...`.

## Approach
Move each script into its own subdirectory under `scripts/`, keeping `package main` and
all runtime behavior unchanged. Each script becomes an isolated Go command package,
so the test runner can build them independently without name collisions. Update any
user-facing instructions to use the new path style, e.g. `go run ./scripts/<name>`.

## Scope
- `scripts/cleanup_duplicates` -> `main.go`
- `scripts/clear_orders` -> `main.go`
- `scripts/diagnose_orders` -> `main.go`
- `scripts/fix_order_data` -> `main.go`
- `scripts/migrate_encryption` -> `main.go`
- Update docs and inline hints that referenced the old `scripts/*.go` paths.

## Risks
External references to the old paths could break if not updated. No runtime logic
changes are expected.

## Test Plan
- `go test ./...` should no longer fail due to duplicate main functions.
- Optional: `go run ./scripts/<name>` to validate each script starts.
