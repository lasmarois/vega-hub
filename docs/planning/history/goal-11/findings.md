# Findings & Decisions

## Requirements
From Goal #11 definition:
- UI shows all active goals from registry
- Each goal shows current executor status
- Pending questions highlighted with notification badge
- Can answer questions from goal detail view
- Real-time updates when executors start/stop/ask
- Q&A history visible per goal

Target UI mockup:
```
┌─────────────────────────────────────────────────┐
│  vega-hub                                  🔔 2 │
├─────────────────────────────────────────────────┤
│  Active Goals                                   │
│                                                 │
│  ● Goal #10: Build vega-hub MVP   [WAITING] 🔴  │
│    └─ Executor asking: "OAuth or JWT?"          │
│                                                 │
│  ● Goal #11: Complete UI          [RUNNING]     │
│    └─ Phase 2/4 - last commit 3m ago            │
│                                                 │
│  ● Goal #12: Fix login bug        [STOPPED]     │
│    └─ Completed 15m ago                         │
├─────────────────────────────────────────────────┤
│  [Click goal for details, Q&A history, logs]    │
└─────────────────────────────────────────────────┘
```

## Research Findings

### vega-hub MVP Architecture
- **Backend:** Go 1.23, single binary with embedded frontend
- **Frontend:** React 18.3 + TypeScript + Tailwind CSS + Vite
- **Real-time:** SSE stream at `/api/events`
- **State:** In-memory (Hub struct), no database

### Existing API Endpoints
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/ask` | POST | Submit question (blocks) |
| `/api/answer/{id}` | POST | Answer pending question |
| `/api/questions` | GET | List pending questions |
| `/api/executors` | GET | List active executors |
| `/api/executor/register` | POST | Register executor |
| `/api/executor/stop` | POST | Stop executor |
| `/api/events` | GET | SSE stream |
| `/api/health` | GET | Health check |

### Existing SSE Events
- `connected` - Initial connection
- `question` - New question from executor
- `answered` - Question answered
- `executor_started` - Executor registered
- `executor_stopped` - Executor stopped

### Hub Data Structures
```go
type Hub struct {
  dir string                     // Vega-missile directory
  questions map[string]*Question // Pending questions
  executors map[string]*Executor // Active executors
  subscribers map[chan Event]bool // SSE subscribers
  mdWriter *markdown.Writer
}

type Question struct {
  ID, SessionID, Question string
  GoalID int
  Options []Option
  CreatedAt time.Time
  answerCh chan string
}

type Executor struct {
  SessionID string
  GoalID int
  CWD string
  StartedAt time.Time
}
```

### REGISTRY.md Format (need to verify)
Need to check the actual format in the vega-missile repo.

### Goal Markdown File Format
From goal #11 file:
- `## Phases` section with phase headings
- `- [ ]` checkboxes for tasks
- `## Status` section with current phase info
- `## Executor Activity` section added by vega-hub

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| New `/internal/goals/` package | Separation of concerns |
| Parse markdown with regex | Simple, no external deps needed |
| Goal list on page load + SSE updates | Reliable initial state |
| Executor.GoalID links to goals | Already tracked per executor |

## Issues Encountered
| Issue | Resolution |
|-------|------------|

## Resources
- Goal definition: `/home/nmarois/new_git/vega-missile-roadmap/goals/active/11.md`
- vega-hub source: `/home/nmarois/new_git/vega-missile-roadmap/workspaces/vega-hub/main/`
- Design doc: referenced in goal file

## Visual/Browser Findings
N/A - working from codebase exploration

---
*Update this file after every 2 view/browser/search operations*
