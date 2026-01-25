# Task Plan: Mobile-first UI Redesign with shadcn Components

**Goal:** #97ade68
**Project:** vega-hub
**Started:** 2026-01-24

---

## Overview

Redesign vega-hub UI to surface all vega-missile data (registry, projects, goals, planning files) with a mobile-first approach that scales to desktop. Use shadcn/ui components for consistent design.

---

## Phase 1: Foundation [complete]

### Tasks
- [x] Install shadcn components (card, badge, button, tabs, sheet, input, scroll-area, separator, skeleton)
- [x] Create Layout component with responsive navigation
  - [x] Header.tsx - top header with title, connection status
  - [x] Sidebar.tsx - desktop sidebar (>1024px)
  - [x] BottomNav.tsx - mobile/tablet bottom navigation (<1024px)
  - [x] Layout.tsx - orchestrates responsive layout switching
- [x] Set up React Router with pages:
  - [x] Home.tsx - dashboard
  - [x] Projects.tsx - projects list
  - [x] Goals.tsx - goals list with filters
  - [x] History.tsx - completed goals
- [x] Implement useSSE hook (extracted from App.tsx)
- [x] Implement useMobile hook (detect viewport)
- [x] Create GoalSheet component with tabs (Overview, Phases, Q&A)

### Acceptance Criteria
- [x] App has bottom nav on mobile, sidebar on desktop
- [x] Navigation works between all 4 pages
- [x] SSE events are received and can trigger state updates
- [x] Tailwind breakpoints: <640px mobile, 640-1024px tablet, >1024px desktop
- [x] Build compiles successfully

---

## Phase 2: Core Views [pending]

### Tasks
- [ ] Home dashboard with:
  - [x] Alert banner for pending questions
  - [x] Stats row (Active, Iced, Completed, Projects)
  - [x] Active goals cards (max 5)
  - [ ] Recent activity timeline
- [ ] Goals list with:
  - [x] Filter chips (All, Active, Iced, Completed)
  - [ ] Sort dropdown
  - [x] GoalCard.tsx component (inline in Goals page)
- [x] GoalSheet.tsx (bottom sheet mobile, right sheet desktop)
  - [x] Sticky header with goal info
  - [x] Tabs: Overview, Phases, Q&A
  - [ ] Add Planning and Timeline tabs
- [ ] Shared components:
  - [ ] StatusBadge.tsx
  - [ ] ActivityItem.tsx
  - [ ] EmptyState.tsx

---

## Phase 3: Projects & History [pending]

### Tasks
- [x] Projects view with ProjectCard (inline)
- [ ] ProjectSheet.tsx with tabs
- [x] History view with search
- [ ] New API endpoints (backend):
  - [ ] GET /api/projects
  - [ ] GET /api/projects/{name}
  - [ ] GET /api/goals/{id}/planning
  - [ ] GET /api/goals/{id}/history

---

## Phase 4: Polish [pending]

### Tasks
- [ ] Command palette (Cmd+K)
- [ ] Toast notifications via SSE
- [x] Loading states (Skeleton)
- [x] Empty states
- [ ] Error handling

---

## Errors Encountered

| Error | Attempt | Resolution |
|-------|---------|------------|
| TS6133: unused imports | 1 | Removed unused navigate, Snowflake, CardDescription, Badge imports |

---

## Files Modified

| Phase | Files |
|-------|-------|
| 1 | web/package.json, web/tailwind.config.js, web/components.json, web/tsconfig.json, web/vite.config.ts, web/src/index.css, web/src/App.tsx, web/src/lib/utils.ts, web/src/lib/types.ts, web/src/components/ui/*, web/src/components/layout/*, web/src/components/goals/*, web/src/pages/*, web/src/hooks/* |
