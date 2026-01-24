# Task Plan: Complete vega-hub Dashboard UI

## Goal
Build the full goal-centric dashboard UI for vega-hub that shows all active goals, their executor status, pending questions with notification badges, and Q&A history per goal.

## Current Phase
Phase 5 - Executor Control Panel

## Phases

### Phase 1: Backend API for Goals
- [x] Create goal parsing module to read `goals/REGISTRY.md`
- [x] Create goal detail parsing for `goals/active/{id}.md`
- [x] Add `GET /api/goals` endpoint to list goals from registry
- [x] Add `GET /api/goals/{id}` endpoint for goal details with Q&A history
- [x] Include executor status per goal
- **Status:** complete

### Phase 2: Goal List View
- [x] Create GoalList component with status badges
- [x] Add status indicators: RUNNING (green), WAITING (red), STOPPED (gray)
- [x] Add notification badge for pending questions
- [x] Wire up real-time updates via SSE
- [x] Update App.tsx to use goal-centric layout
- **Status:** complete

### Phase 3: Goal Detail View
- [x] Create GoalDetail component (right panel in split view)
- [x] Show phases and progress from goal file
- [x] Show Q&A history for the goal
- [x] Show executor activity (active sessions)
- [x] Allow answering questions from detail view
- **Status:** complete

### Phase 4: Polish & Integration
- [x] Implement responsive layout (split panel)
- [x] Add loading states
- [x] Add error handling (try/catch, console.error)
- [x] Committed implementation
- **Status:** complete

### Phase 5: Executor Control Panel
- [x] `POST /api/goals/{id}/spawn` - Spawn executor on goal
- [x] `GET /api/goals/{id}/status` - Read planning files for progress
- [x] Resume button (▶) - Spawn executor with optional context
- [x] Context input - Text area for additional instructions
- [x] Status display - Show current phase, recent actions from planning files
- [x] Live progress - Auto-refresh from `progress.md`
- **Status:** complete

## Summary (Phases 1-4)

Phases 1-4 complete. The goal-centric dashboard UI has been implemented with:

1. **Backend API** (`/api/goals`, `/api/goals/{id}`) - Parses REGISTRY.md and goal files
2. **Goal List View** - Split panel layout with status badges and notification counts
3. **Goal Detail View** - Shows phases, Q&A, executors, overview, acceptance criteria
4. **Real-time Updates** - SSE integration for all events (question, answered, executor_started/stopped)

Acceptance criteria status:
- [x] UI shows all active goals from registry
- [x] Each goal shows current executor status
- [x] Pending questions highlighted with notification badge
- [x] Can answer questions from goal detail view
- [x] Real-time updates when executors start/stop/ask
- [x] Q&A history visible per goal (via pending_questions)

## Key Questions
1. How is the REGISTRY.md formatted? Need to parse it for goal list
2. How are goal markdown files structured? Need to extract phases and status
3. Should goals be fetched on page load or streamed via SSE?
4. How to track "WAITING" vs "RUNNING" status for goals?

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| Add new `/internal/goals/` package for parsing | Keep goal logic separate from hub |
| Poll registry initially, SSE for updates | Simple implementation, reliable |
| Use existing Question struct for Q&A display | Reuse existing types |

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|

## Notes
- Backend already tracks executors and questions per goal
- React + Tailwind already set up from MVP
- SSE infrastructure exists, just need more event types
- REGISTRY.md format: need to check actual file structure
