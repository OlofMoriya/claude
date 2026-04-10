# AGENTS Guide for Owl

## Purpose
- Capture the conventions agents rely on so that edits stay consistent with the human maintainers.
- Focus on workflows under `owl/` (Go 1.24) plus the helper docs `OWL_ARCHITECTURE.md` and `SKILLS.md`.
- No Cursor rule files or Copilot instruction files exist in this repo, so this document is the authoritative agent-facing brief.

## Repository Snapshot
- Monorepo-style Go project with entrypoint `main.go`, organized packages under `common_types`, `data`, `embeddings`, `http`, `logger`, `mode`, `models`, `services`, `tools`, and `tui`.
- `OWL_ARCHITECTURE.md` and `SKILLS.md` provide high-level system and tool summaries—review them before large refactors.
- Persisted state lives in SQLite/DuckDB via files in `data/db/`; HTTP server state uses the `data.MultiUserContext` repository.
- Tooling is embedded in the Go binary; helper shell scripts (e.g., `collect-text.sh`, `inspect_embeddings`) assume macOS.
- Tests currently focus on services such as `services/chunking_test.go`; expand coverage there as new logic appears.
- Vector embeddings rely on env vars `OWL_LOCAL_DATABASE` and `OWL_LOCAL_EMBEDDINGS_DATABASE` when storing/searching.
- No Makefile/Justfile; all commands are plain `go`/`bash`.

## Build & Dependency Commands
- `go env GOPATH` before building if the agent shell is fresh (Go 1.24 needed).
- Install deps: `go mod download` (only needed once per checkout).
- Keep modules tidy: `go mod tidy` when dependencies change—commit the result if go.sum updates.
- Build release binary: `go build -o owl ./...` (outputs `./owl`).
- Quick compile check without artifact: `go build ./...`.
- Run in CLI mode: `go run . -prompt "Hello"` inside `src/`.
- Cross-check for missing imports: `go list ./...` (fails fast on compile issues).

## Run Modes & Useful Flags
- Default CLI prompts interactively; pass `-prompt` to skip stdin.
- TUI mode: `go run . -tui` launches the Bubbletea interface defined in `tui/`.
- HTTP server: `go run . -serve -port 8080 [-secure]`; server wiring sits in `http/` and `models/claude`.
- Embeddings create/search: `go run . -embeddings -chunk path/to.md -prompt "context"` or `-search "phrase"`.
- Store markdown chunks without prompting by combining `-embeddings -chunk file.md -create_context`.
- History viewer: `go run . -view -context_name my_ctx -history 25`.
- Tool filters: `-tools group1,group2` toggles subsets defined via `ToolModel.GetGroups`.

## Test & Quality Workflow
- Run the full suite: `go test ./...` (only services currently have tests; extend coverage as you add packages).
- Target a single package: `go test ./services`.
- Target a single test: `go test ./services -run TestChunkMarkdown_LargeChunk -count=1`.
- Run benchmarks from `services/chunking_test.go`: `go test ./services -run BenchmarkChunkMarkdown -bench BenchmarkChunkMarkdown -benchmem`.
- Enable race detector for concurrency-heavy areas (`tools/`, `http/`): `go test -race ./...`.
- Static analysis baseline: `go vet ./...`; run before merging logic-heavy changes.
- Formatting gate: `gofmt -w $(git ls-files '*.go')` or format touched files only.
- Optional extras if available locally: `staticcheck ./...` and `golangci-lint run` (no config checked in, so use defaults).

## Environment & Secrets
- `.env` loading is handled via `github.com/joho/godotenv` inside `main.go`; create `.env` with API keys during local runs but do not commit it.
- Critical env vars: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, optional `GROK_API_KEY`, `OLLAMA_URL`, `OWL_LOCAL_DATABASE`, `OWL_LOCAL_EMBEDDINGS_DATABASE`.
- `tools/update_file_tool` also reads `OWL_REQUIRE_APPROVAL` to enforce manual diff approval.
- `tool` utilities rely on OS programs like `git`, `patch`, `tmux`, and `todo-tui`; ensure they are available when you invoke those flows.
- Avoid hardcoding secrets; fetch via `secrets.go` or env lookups, and plumb them down via configuration structs.

## Code Style – Imports & Formatting
- Use `gofmt`/`goimports` to keep imports grouped: standard library, third-party, then local `owl/...` modules.
- Alias local packages only when names collide (see `commontypes "owl/common_types"`). Keep alias names descriptive.
- Keep import lists alphabetized within each block.
- Prefer multi-line import blocks; avoid dot-imports and blank identifier imports unless referencing generated code.
- Stick to tabs for indentation (Go default) and limit line length to something readable (~120 chars) but prioritize clarity.

## Code Style – Types & Data Structures
- Exported structs/interfaces require doc comments (`// Context ...`) because they are part of the CLI/library API.
- JSON tags use lower snake_case and align with DB columns (`History struct` in `data/history-model.go`).
- Favor value receivers for stateless helpers and pointer receivers when mutating or avoiding copies (e.g., `ToolRunner`).
- For shared config, prefer explicit structs (see `embeddings.Config`) rather than passing maps.
- When storing tool responses, reuse `commontypes.ToolResponse` to keep streaming handlers consistent.

## Code Style – Error Handling & Logging
- Return `error` values up the stack; only `panic` in truly fatal CLI contexts (e.g., invalid `-context_name` for history view).
- Wrap errors with context via `fmt.Errorf("action failed: %w", err)` when bubbling up.
- User-facing status goes through `logger.Screen` with optional `color.Color`; non-interactive logs written via `logger.Debug`.
- Honor the non-blocking channel semantics in `logger.Screen` when emitting frequent updates (do not block the UI).
- When operating in server/TUI mode, prefer structured errors over `panic`; respond via HTTP or UI handlers instead.
- Validate user input early (examples in `FileUpdateTool.Run`) and exit with descriptive errors rather than silent failures.

## Code Style – Naming & Structure
- Packages stay lower_snake or single word; files like `history-model.go` describe domain objects.
- Exported names use PascalCase (`ChunkMarkdown`, `ToolRunner`). Private helpers like `splitOnHeaders` use camelCase.
- Keep flag variables and package-level configs grouped and commented in `main.go`.
- Prefer short helper functions (see `launchTUI`, `view_history`) rather than gigantic conditional blocks.
- Maintain consistent naming of contexts/history (`context_name` flag, `data.Context`). Avoid mixing `camelCase` and `snake_case` in Go code.

## Code Style – Tests
- Follow standard `testing` package conventions; one file per unit like `services/chunking_test.go` with `TestXxx` and `BenchmarkXxx` functions.
- Use table-driven tests when checking multiple inputs; keep assertions specific (`t.Errorf("expected 3 chunks, got %d", len(chunks))`).
- Keep benchmarks deterministic and reuse builders, as seen in `BenchmarkChunkMarkdown`.
- Cover constants and guardrail logic (see `TestChunkSizes` verifying chunk size relationships); mimic that style when adding new constants.
- When testing streaming/model code, stub interfaces (`commontypes.ResponseHandler`) rather than hitting real APIs.

## Tools Package Expectations
- Every tool implements `ToolModel` (definition, name, run, history, groups) and self-registers via `init()` calling `tools.Register`.
- Guard file paths in writer/updater tools: reject `..`, `/`, `~` just like `write_file_tool.go` and `update_file_tool.go` do.
- Schema descriptions live in `Tool.InputSchema`; keep them specific so the AI call-site knows required fields.
- Use the group filter system: `GetGroups` should tag tools (`dev`, `writer`, etc.) so `-tools` flag filtering remains meaningful.
- For interactive approvals, respect `OWL_REQUIRE_APPROVAL`, `TMUX`, and fallback behaviors from `diff_viewer` implementations.
- Prefer returning human-readable strings from `Run` for direct display in the CLI/TUI.

## Services & Embeddings Notes
- Text chunking lives in `services/chunking.go`; keep thresholds (Optimal/Max/Min) tuned for ~1536-dim embeddings and document any changes.
- `embeddings.Run` decides between storing and querying; configuration uses `embeddings.Config` (check `embeddings/embeddings.go`).
- Long-running services should stream progress through the response handler channel when possible.
- When touching chunking logic, also update `services/chunking_test.go` to cover regressions.
- File-backed DB helpers exist for SQLite (`data/sqllite-db.go`, `multi-user-sqllite-db.go`) and DuckDB (`embeddings-db-duckdb.go`); keep migrations in sync across them.

## HTTP, CLI, and TUI Layers
- CLI flow is orchestrated in `main.go` with flag parsing then mode dispatch; keep new flags grouped in `init()`.
- HTTP server behaviors live in `http/` and rely on `server.HttpResponseHandler`; wire any new models through there.
- TUI pipeline uses Bubbletea (`tui/`); be mindful of concurrency—UI updates must be channeled via model messages.
- Model selection is performed by `picker.GetModelForQuery`; extend it there rather than branching ad hoc.
- Response handlers (CLI, HTTP, tool) implement `commontypes.ResponseHandler`; ensure you call `FinalText` exactly once per request.

## Logging & Diagnostics
- Keep `logger.Init` writing to `~/.owl/debug.log`; do not reinitialize the logger mid-run.
- Diagnostics for agent interventions should be short and human-readable; color accents use `fatih/color` and should remain optional.
- Diff viewers (`tools/diff_viewer.go`, `tools/diff_viewer_tmux.go`) expect ANSI-capable terminals; detect `TMUX` before launching the tmux variant.
- When running headless (CI), skip diff approvals or degrade gracefully (see `isTerminal` in `update_file_tool.go`).

## Workflow Recommendations
- Before editing, inspect `git status` and `git diff` to avoid stomping user work; do not revert files you did not touch.
- Prefer `apply_patch` for surgical edits; keep diffs minimal and documented when referencing this AGENTS guide.
- Update `SKILLS.md` if you add/remove tools so the agent instructions stay accurate.
- Reference this AGENTS.md whenever you need build/test/style conventions; expand it if new rules emerge.

## Data Layer & Persistence
- SQLite helpers live in `data/sqllite-db.go` and friends; keep schema migrations identical between single-user and multi-user repositories.
- DuckDB embedding storage sits in `data/embeddings-db-duckdb.go`; align schema with the SQLite vec setup when evolving fields.
- Repository structs like `data.Context` and `data.History` map 1:1 with DB rows; update JSON tags alongside SQL column changes.
- When introducing new persistence, extend `data/utils.go` for shared helpers rather than duplicating SQL snippets.
- Respect file paths under `data/db/`; do not assume the DB file name—derive from `OWL_LOCAL_DATABASE`.
- Add integration tests when changing DB logic; stub DB handles when writing unit tests to keep them fast.

## Models & AI Providers
- `picker.GetModelForQuery` decides which concrete model to instantiate; extend its switch rather than branching elsewhere.
- Claude implementation lives in `models/claude`; follow its pattern for streaming, thinking tokens, and tool wiring.
- Ollama support (`models/ollama`) reads `OLLAMA_URL` and optional `OLLAMA_API_KEY`; document any new env vars you add.
- Model interfaces expect `SetResponseHandler` calls before `CreateRequest`; enforce this ordering when composing new flows.
- Use `commontypes.PayloadModifiers` to pass attachments (PDF, image, web search) rather than ad-hoc parameters.
- Keep HTTP clients in model packages configurable for testing; dependency inject via struct fields when possible.

## External Scripts & Utilities
- `collect-text.sh` and `inspect_embeddings` target macOS defaults; mention platform assumptions when editing them.
- Tool scripts under `tools/` invoke `git`, `patch`, `tmux`, and `todo-tui`; guard for their presence and fail with actionable errors.
- The `diff_viewer_tmux` path checks `TMUX`; degrade gracefully when the env var is missing even if requested explicitly.
- Shell out commands with contextual logging using `logger.Screen` so CLI users see progress.
- Respect the `OWL_DIFF_VIEWER` override when launching diff approvals.
- When adding new scripts, keep them POSIX sh-compatible and executable (run `chmod +x`).

## Contribution Checklist for Agents
- Read this AGENTS.md plus `OWL_ARCHITECTURE.md`/`SKILLS.md` before editing areas you have not touched recently.
- Confirm Go version 1.24+ via `go version` if builds suddenly fail.
- Run unit tests (`go test ./...`) before handing code back; rerun focused tests after touching sensitive logic.
- Format Go files you edit with `gofmt -w file.go`; do not rely on editors to do it implicitly.
- Scan for TODO/FIXME markers after changes to ensure no temporary hacks remain.
- When you introduce new commands or workflows, update this file (and `SKILLS.md` if tool-related) so future agents stay aligned.
