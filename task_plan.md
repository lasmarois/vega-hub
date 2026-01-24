# Task Plan: Complete vega-hub Dashboard UI

## Goal
Build the full goal-centric dashboard UI for vega-hub that shows all active goals, their executor status, pending questions with notification badges, and Q&A history per goal.

## Current Phase
Phase 4 - Polish & Integration

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
- [ ] Test with real executors
- [ ] Ensure all acceptance criteria met
- **Status:** in_progress

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
