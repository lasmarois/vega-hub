# Task Plan: Goal Completion Automation - Phase 1

Goal: #a3a5377
Phase: 1/4

## Objective

Create `internal/goals/completion.go` with detection logic to identify when a goal is complete based on:
1. Planning file progress (all phases marked done in task_plan.md)
2. Goal file acceptance criteria (all checked)
3. Commit messages (semantic signals)

## Phases

### Phase 1: Completion Detection Logic (Current)
- [x] Analyze existing patterns in parser.go and state.go
- [ ] Create `completion.go` with CompletionStatus struct
- [ ] Implement planning file parsing (task_plan.md)
- [ ] Implement goal file acceptance criteria parsing
- [ ] Implement commit message detection
- [ ] Create comprehensive test file
- [ ] Run tests and ensure passing

### Phase 2: API Integration
- [ ] Add `GET /api/goals/:id/completion-status` endpoint
- [ ] Include completion status in goal detail response
- [ ] Add SSE event when completion detected

### Phase 3: UI Integration
- [ ] Show completion status badge on goal cards
- [ ] Add "Mark Complete" prompt when completion detected
- [ ] Show missing items checklist view

### Phase 4: Auto-Update Option
- [ ] Add config option `autoCompleteGoals`
- [ ] Implement auto-update when completion detected
- [ ] Send notification when auto-completed

## Design Decisions

### CompletionStatus Structure
```go
type CompletionStatus struct {
    Complete  bool              `json:"complete"`
    Signals   []CompletionSignal `json:"signals"`
    Missing   []string          `json:"missing"`
    Confidence float64          `json:"confidence"` // 0.0-1.0
}

type CompletionSignal struct {
    Type    string `json:"type"`    // "planning_file", "acceptance", "commit"
    Source  string `json:"source"`  // File path or commit hash
    Message string `json:"message"` // Human-readable description
}
```

### Detection Strategy
- Conservative approach: better to miss a completion than false positive
- Require multiple signals for high confidence
- Each signal type has independent detection logic

### Patterns to Follow
- Use same directory traversal patterns as parser.go
- Use similar regex patterns for markdown parsing
- Follow test patterns from parser_test.go
- Maintain thread-safe design consistent with state.go
