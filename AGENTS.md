# AGENTS.md

## Scope

These instructions apply to the whole repository.

## Project Summary

- `norm` is a lightweight Go ORM library.
- `norm` provides DB-facing execution helpers built on `database/sql`.
- `nsql` builds SQL for select/count/insert/update/delete/upsert flows.
- `ngen` generates model and CRUD scaffolding from database metadata.
- `driver` contains driver names and the internal placeholder token used before driver-specific conversion.

See [README.md](README.md) for public usage examples and API-level overview.

## Environment

- Required Go version: `1.24+` from [go.mod](go.mod).
- This is a library repo, not an application repo.

## Safe Default Commands

- Prefer `go build ./...` for a quick repo-wide sanity check.
- Prefer targeted tests over `go test ./...`.
- For SQL builder changes under `nsql/`, start with `go test ./nsql -run TestGenerator`.

## Test Boundaries

- Do not run `go test ./...` by default. Some tests depend on a live database or external DSN.
- [norm_test.go](norm_test.go) opens a local MySQL datasource and behaves like integration testing.
- [ngen/gen_test.go](ngen/gen_test.go) contains a hard-coded external DSN and absolute output paths; treat it as an example, not a safe default test.
- `nsql` tests are the safest starting point because many only print generated SQL.

## Editing Guidance

- Preserve the current package split: execution logic stays in `norm`, SQL generation stays in `nsql`, code generation stays in `ngen`.
- Follow the existing chaining style for ORM builders.
- Keep driver placeholder handling consistent with [driver/driver.go](driver/driver.go) and conversion logic in [norm.go](norm.go).
- When changing generated SQL shape, update or add focused tests in `nsql` first.
- Avoid broad refactors unless the task explicitly requires them.

## Useful Entry Points

- [norm.go](norm.go): datasource handling and query execution helpers.
- [nsql/condition.go](nsql/condition.go): query condition primitives.
- [nsql/orm.go](nsql/orm.go): shared ORM builder behavior.
- [nsql/select_orm.go](nsql/select_orm.go): select SQL generation patterns.
- [nsql/update_orm.go](nsql/update_orm.go): update and batch update SQL generation.
- [ngen/gen.go](ngen/gen.go): generator entry point.