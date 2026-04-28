# Owl Agent/Tool Roadmap

## Locked Decisions
- Retire tools: `internal_dbask_tool`, `issue_list`, and birdnest/QA lane.
- Keep `query_approval` for future reuse.
- Keep role names: `manager`, `secretary`.
- Add role: `planner` (developer planning).
- Add agent definition layer separate from tool definitions.
- `ask_questions`: batch-in / batch-out, TUI-first, full-screen, cancel as tool error.
- Calendar integration via `icalBuddy`, slim tool definition.
- Build/verify tool supports Go + C# with slim schema.

## Target Architecture
- Tool metadata: typed `ToolGroup` + typed `ToolDependency`.
- Agent metadata: separate definitions for `planner`, `developer`, `manager`, `secretary`.
- Resolution pipeline:
  1. base harness prompt
  2. context prompt
  3. agent prompt
  4. `-system` override/append (highest priority)
- Tool exposure:
  - by dependencies (capabilities)
  - by selected groups/tools in agent profile
  - optional explicit include/exclude overrides

## Phase 1 - Tool Retirement (hard delete)
- Delete:
  - `tools/internal_dbask_tool.go`
  - `tools/internal_dbask_tool_test.go`
  - `tools/issue_list_tool.go`
- Remove references in tests/docs:
  - dbask assertions/fixtures in `models/claude/claude-model_test.go`
  - references in `SKILLS.md`, `OWL_ARCHITECTURE.md`
- Keep `tools/query_approval.go` untouched.

### Acceptance
- No compile/test references to `internal_dbask_tool`, `issue_list`, `dbask`, `item-list.sh`.

## Phase 2 - Typed Enums + Compatibility
- Add:
  - `type ToolGroup string`
  - `type ToolDependency string`
- Introduce constants for initial sets:
  - groups: `dev`, `writer`, `chat`, `planner`, `developer`, `manager`, `secretary`
  - dependencies: `local_exec`, `tui_interactive`
- Migrate filtering internals to typed values.
- Keep CLI input compatible (string aliases mapped to typed groups).

### Acceptance
- Existing `-agent` behavior still works.
- Internal filtering is enum-safe.
- No behavior regression in tool selection.

## Phase 3 - Agent Definition Layer (main.go + prompt composition)
- Add agent definition model (file-backed or in-code map):
  - `name`, `description`
  - `system_prompt`
  - `tool_groups`
  - optional `include_tools`, `exclude_tools`
  - optional defaults (`model`, `history`, etc.)
- Update `main.go` flow:
  - resolve selected agent from flag/context default
  - resolve effective tool selection from agent + existing filters
  - compose effective system prompt while respecting `-system`
- Prompt precedence:
  - base prompt + context + agent + `-system` (highest)

### Acceptance
- Agent selection changes both prompt and tool set.
- `-system` still works and is respected.
- Context prompt still participates.

## Phase 4 - `ask_questions` Tool + TUI Questionnaire
- Add `ask_questions` tool:
  - batch input questions
  - batch output answers
- Add TUI full-screen questionnaire mode:
  - vim-style navigation
  - required vs optional validation
  - custom answer support
  - cancel returns tool error
- Non-TUI path returns explicit "requires TUI mode."

### Acceptance
- Model can ask a question batch and resume with returned answers.
- Works in TUI; blocked cleanly in CLI/HTTP.

## Phase 5 - Slim `icalBuddy` Calendar Tool
- Add read-only tool, e.g. `calendar_overview`.
- Slim schema:
  - required `query` (string)
  - optional `options` (string)
- Runtime allowlists:
  - read commands only
  - safe option subset only
  - timeout + output bounds

### Acceptance
- Calendar/task query works with minimal schema.
- No mutation/config commands allowed.

## Phase 6 - Slim Cross-Language Verify Tool (Go + C#)
- Add `project_check` with slim schema:
  - required `mode` string only (`auto:quick`, `go:full`, `csharp:quick`, etc.)
  - optional `target` only if needed for monorepos
- Command behavior:
  - Go quick/full
  - C# quick/full
  - auto detection fallback
- Return concise structured status.

### Acceptance
- Reliable checks for Go and C# from one tool.
- Safe, predictable command matrix.

## Phase 7 - Validation + Rollout
- Run:
  - `go test ./...`
  - targeted tests for tools/models/tui changes
- Search for stale references and remove.
- Update docs:
  - tools list
  - agent behavior
  - group/dependency taxonomy
- Roll out in PR-sized chunks:
  1. retirements
  2. enums
  3. agents/main flow
  4. ask_questions
  5. calendar + project_check

## Agent Prompt First Take
- `planner`: read-only planning, risk analysis, validation matrix.
- `developer`: execute minimal diffs, verify with tests/build.
- `manager`: personal/work operations, priorities, action lists.
- `secretary`: communication/scheduling/follow-up quality.

## Progress Log
- [x] Planning complete (initial roadmap drafted)
- [x] Phase 1 started
- [x] Phase 1 completed
- [x] Phase 2 started
- [x] Phase 2 completed
- [ ] Phase 3 started
- [ ] Phase 4 started
- [ ] Phase 5 started
- [ ] Phase 6 started
- [ ] Phase 7 started
