# Findings: Goal Completion Detection

## Existing Patterns Analyzed

### parser.go
- Uses `bufio.Scanner` for line-by-line parsing
- Regex patterns for markdown parsing: `regexp.MustCompile(...)`
- `PhaseDetail` struct with `Status` field: "pending", "in_progress", "complete"
- Task completion tracked via `Task.Completed` bool
- Goal file locations: `goals/active/`, `goals/iced/`, `goals/history/`
- Worktree info in goal file's `## Worktree` section

### state.go
- Uses mutex for thread safety
- JSONL append-only storage for state events
- State transitions with validation
- Helper functions like `isValidGoalID()`

### Test Patterns (parser_test.go)
- `setupTestDir(t)` helper creates temp directory
- `writeFile(t, path, content)` helper
- Table-driven tests for edge cases
- Checks both success and error cases

## Key Insights

### Planning File Detection
The planning-with-files skill creates `task_plan.md` at worktree root.
Need to parse:
- Phase checkboxes: `- [x]` vs `- [ ]`
- Phase status markers: `**Status:** complete`
- All phases must be complete for goal completion

### Worktree Path Resolution
From goal file's `## Worktree` section:
```markdown
## Worktree
- **Branch**: goal-a3a5377-...
- **Project**: vega-hub
- **Path**: workspaces/vega-hub/goal-a3a5377-...
```

Use this to find `task_plan.md` at:
`{vega-missile-dir}/{worktree-path}/task_plan.md`

### Acceptance Criteria
Already parsed by `ParseGoalDetail()` as `detail.Acceptance []string`.
Need to extend to track completion status of each criterion.

### Commit Detection
Look for recent commits on goal branch containing:
- "complete", "done", "finish" + goal ID
- Commit message patterns suggesting completion

## Files to Create

1. `internal/goals/completion.go` - Main detection logic
2. `internal/goals/completion_test.go` - Tests
