# vega-hub UI Redesign

> Mobile-first dashboard for vega-missile orchestration.

**Date:** 2026-01-24
**Status:** Design Complete

---

## Overview

Redesign vega-hub UI to surface all vega-missile data (registry, projects, goals, planning files) with a mobile-first approach that scales to desktop.

## Design Principles

1. **Mobile-first** - Design for phone, enhance for desktop
2. **Progressive disclosure** - Dashboard → drill into details
3. **Real-time** - SSE updates, toast notifications
4. **Keyboard-friendly** - Command palette (⌘K) for power users

---

## Navigation Structure

```
Bottom Nav (Mobile) / Sidebar (Desktop)
├── 🏠 Home (Dashboard)
│   ├── Needs Attention (pending questions)
│   ├── Active Goals (cards)
│   └── Recent Activity (timeline)
│
├── 📁 Projects
│   ├── Project cards with stats
│   └── Tap → Project detail sheet
│
├── 🎯 Goals
│   ├── Filter: All / Active / Iced / Completed
│   ├── Sort: Recent / Project / Status
│   └── Goal cards → detail sheet
│
└── 📋 History
    ├── Search + project filter
    ├── Grouped by date
    └── Completed goals (read-only detail)
```

---

## Layout

### Mobile (< 640px)
- Bottom navigation bar (4 tabs)
- Full-screen views
- Goal/project detail: bottom sheet (90% height)
- Swipe down to dismiss

### Desktop (> 1024px)
- Left sidebar (240px fixed)
- Main content area
- Goal/project detail: right sheet (480px)
- Click outside to dismiss

### Tablet (640-1024px)
- Bottom nav, 2-column card grid
- Sheets slide up from bottom

---

## Views

### Home (Dashboard)

1. **Alert Banner** - Pending questions (conditional)
2. **Stats Row** - 4 cards: Active, Iced, Completed, Projects
3. **Active Goals** - Goal cards (max 5)
4. **Recent Activity** - Timeline of events

### Projects

- Project cards with stats (active/iced/done counts)
- Base branch badge, upstream link
- Tap → Sheet with tabs: Overview, Goals, Settings

### Goals

- Filter chips: All | Active | Iced | Completed
- Sort dropdown: Recent, Project, Status
- Grouped sections with counts
- Completed section collapsed by default

### History

- Search input + project filter
- Grouped by date (Today, Yesterday, etc.)
- Simplified cards (read-only)

---

## Goal Detail Sheet

**Sticky Header:**
- Goal ID + Title
- Project badge + Phase
- Status badge

**Action Bar:**
- Resume Executor (primary)
- View Output, Ice Goal (secondary)
- Answer input (when WAITING)

**Tabs:**
| Tab | Content |
|-----|---------|
| Overview | Description, acceptance criteria, notes |
| Phases | Phase cards with task checklists |
| Planning | task_plan.md, findings.md, progress.md |
| Q&A | Pending questions + full history |
| Timeline | Executor activity log |

---

## Supporting Features

### Command Palette (⌘K)
- Search goals/projects
- Quick navigation
- Pending questions at top

### Real-time (SSE)
| Event | Response |
|-------|----------|
| question | Toast + badge update |
| answered | Refresh Q&A |
| executor_started | Toast + status update |
| executor_stopped | Toast with "View Output" |
| goal_updated | Refresh goal |
| registry_updated | Refresh list |

### Toast Notifications
- Position: bottom-right (desktop), top-center (mobile)
- Types: info, warning, destructive, with action

---

## Component Structure

```
src/
├── components/
│   ├── layout/
│   │   ├── Header.tsx
│   │   ├── Sidebar.tsx
│   │   ├── BottomNav.tsx
│   │   └── Layout.tsx
│   ├── goals/
│   │   ├── GoalCard.tsx
│   │   ├── GoalSheet.tsx
│   │   ├── GoalOverview.tsx
│   │   ├── GoalPhases.tsx
│   │   ├── GoalPlanning.tsx
│   │   ├── GoalQA.tsx
│   │   └── GoalTimeline.tsx
│   ├── projects/
│   │   ├── ProjectCard.tsx
│   │   └── ProjectSheet.tsx
│   └── shared/
│       ├── StatusBadge.tsx
│       ├── ActivityItem.tsx
│       └── EmptyState.tsx
├── pages/
│   ├── Home.tsx
│   ├── Projects.tsx
│   ├── Goals.tsx
│   └── History.tsx
├── hooks/
│   ├── useSSE.ts
│   ├── useGoals.ts
│   ├── useProjects.ts
│   └── useMobile.ts
└── lib/
    ├── api.ts
    └── utils.ts
```

---

## New API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /api/projects` | List projects with stats |
| `GET /api/projects/{name}` | Project detail |
| `GET /api/goals/{id}/planning` | Read planning files |
| `GET /api/goals/{id}/history` | Read archived planning |

---

## shadcn Components

```bash
npx shadcn@latest add card badge button tabs sheet \
  command dialog input select scroll-area collapsible \
  separator dropdown-menu avatar skeleton toast alert
```

---

## Responsive Breakpoints

| Width | Layout |
|-------|--------|
| < 640px | Bottom nav, full-width, sheet slides up |
| 640-1024px | Bottom nav, 2-col grid, sheet slides up |
| > 1024px | Sidebar, multi-column, sheet slides right |

---

## Implementation Phases

### Phase 1: Foundation
- [ ] Install shadcn components
- [ ] Create Layout with responsive nav
- [ ] Set up React Router
- [ ] Implement SSE hook

### Phase 2: Core Views
- [ ] Home dashboard
- [ ] Goals list with filters
- [ ] Goal detail sheet (all tabs)

### Phase 3: Projects & History
- [ ] Projects view + sheet
- [ ] History view with search
- [ ] New API endpoints

### Phase 4: Polish
- [ ] Command palette
- [ ] Toast notifications
- [ ] Loading states
- [ ] Empty states
- [ ] Error handling

---

## Data Sources

| UI Element | Source |
|------------|--------|
| Goal list | `goals/REGISTRY.md` |
| Goal detail | `goals/active/{id}.md` or `goals/history/{id}.md` |
| Project list | `projects/index.md` |
| Project detail | `projects/{name}.md` |
| Planning files | Worktree root: `task_plan.md`, `findings.md`, `progress.md` |
| Archived planning | `docs/planning/history/goal-{id}/` |
| Executor activity | `## Executor Activity` section in goal markdown |
| Q&A history | `## Executor Questions` section in goal markdown |
