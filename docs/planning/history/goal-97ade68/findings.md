# Findings: vega-hub UI Redesign

**Goal:** #97ade68
**Updated:** 2026-01-24

---

## Current State Analysis

### Existing Frontend Structure
- Single `App.tsx` file (~820 lines) with all logic
- Uses React 18.3.1 with TypeScript
- Tailwind CSS already configured
- Vite as build tool
- No React Router - single page app
- No component library - all custom styles

### Existing Features
1. Goal list in left panel (1/3 width)
2. Goal detail in right panel (2/3 width)
3. SSE connection for real-time updates
4. Question answering interface
5. Executor spawning and output viewing
6. Phase progress display

### Current UI Limitations
- Desktop-only layout (fixed 1/3 + 2/3 split)
- No mobile responsiveness
- All components in one file
- No navigation - single view only
- No shadcn components

---

## Design Doc Key Points

### Responsive Breakpoints
| Width | Layout |
|-------|--------|
| < 640px | Bottom nav, full-width, sheet slides up |
| 640-1024px | Bottom nav, 2-col grid, sheet slides up |
| > 1024px | Sidebar, multi-column, sheet slides right |

### Navigation Structure
- Home (Dashboard) - /
- Projects - /projects
- Goals - /goals
- History - /history

### shadcn Components Needed
```bash
npx shadcn@latest add card badge button tabs sheet \
  command dialog input select scroll-area collapsible \
  separator dropdown-menu avatar skeleton toast alert
```

---

## Technical Decisions

### Decision 1: shadcn Installation Method
**Choice:** Use shadcn CLI with manual config
**Reason:** Project already has Tailwind, need to configure paths correctly

### Decision 2: Routing
**Choice:** React Router v6
**Reason:** Standard React routing, supports nested layouts

### Decision 3: State Management
**Choice:** Keep useState/useEffect, extract to custom hooks
**Reason:** App is not complex enough to need Redux/Zustand yet

### Decision 4: Component Organization
```
src/
├── components/
│   ├── layout/       # Layout components
│   ├── goals/        # Goal-related components
│   ├── projects/     # Project components
│   ├── shared/       # Reusable components
│   └── ui/           # shadcn components
├── pages/            # Route pages
├── hooks/            # Custom hooks
└── lib/              # Utilities
```

---

## API Endpoints (Existing)

| Endpoint | Purpose |
|----------|---------|
| GET /api/goals | List all goals |
| GET /api/goals/{id} | Goal detail |
| GET /api/goals/{id}/status | Goal status from planning files |
| GET /api/goals/{id}/output | Executor output |
| GET /api/goals/{id}/spawn | Spawn executor |
| POST /api/answer/{id} | Answer question |
| GET /api/events | SSE stream |

### New Endpoints Needed (Phase 3)
| Endpoint | Purpose |
|----------|---------|
| GET /api/projects | List projects with stats |
| GET /api/projects/{name} | Project detail |
| GET /api/goals/{id}/planning | Read planning files |
| GET /api/goals/{id}/history | Read archived planning |
