# Vega Missile Executor Workflow

> This project is managed by Vega Missile for cross-project goal coordination.

## How Inheritance Works

**Automatic context inheritance.** Claude Code walks up the directory tree and loads all `CLAUDE.md` files and `.claude/rules/` it finds.

Being under `workspaces/` means you automatically inherit:
- Manager's goal registry
- Manager's orchestration rules
- Manager's conventions

No explicit imports needed.

## vega-hub Integration

**All executor communication flows through vega-hub** - a central binary that:
- Tracks executor lifecycle (start/stop)
- Routes questions to humans via web UI
- Writes all updates to goal markdown files
- Provides real-time visibility to the manager

This design enables future remote execution - executors don't need direct filesystem access.

## Hooks

| Hook | What It Does |
|------|--------------|
| `SessionStart` | Checks skill dependencies, injects goal context |
| `PreToolUse` (AskUserQuestion) | Routes questions to vega-hub, blocks until answered |
| `Stop` | Notifies vega-hub that executor stopped |

The SessionStart hook:
1. Checks if `planning-with-files` skill is installed
2. Warns with install instructions if missing
3. Injects goal context and reminders

All hooks communicate with vega-hub via HTTP. vega-hub handles markdown writes, SSE events, and notifications.

## Asking Questions

You can ask the human questions directly using `AskUserQuestion`. The hook intercepts it and routes through vega-hub:

1. You call `AskUserQuestion`
2. Hook POSTs to vega-hub (blocks)
3. Human sees question in web UI
4. Human answers
5. Hook returns answer to you
6. vega-hub logs Q&A to goal markdown

**Use this when you need clarification** - don't guess or make assumptions.

## Required Skill: planning-with-files

The `planning-with-files` skill is **REQUIRED** for vega-missile executors.

**Install with:**
```bash
claude plugins install OthmanAdi/planning-with-files
```

Source: https://github.com/OthmanAdi/planning-with-files

If the skill is missing, the SessionStart hook will warn you with installation instructions.

## Executor Session Checklist

**On every session start:**

1. Confirm you're in the correct goal worktree
2. Load `planning-with-files` skill:
   ```
   Skill(skill: "planning-with-files")
   ```
3. Create/update planning files at **worktree root**:
   - `./task_plan.md`
   - `./findings.md`
   - `./progress.md`

## Planning Files

### During Work

| File | Location |
|------|----------|
| task_plan.md | Worktree root (`./`) |
| findings.md | Worktree root (`./`) |
| progress.md | Worktree root (`./`) |

Planning files stay at root while work is in progress. They are working documents.

### On Explicit Archive Request

When manager deems the goal complete and asks you to archive:

```
docs/planning/history/goal-N/
├── task_plan.md
├── findings.md
└── progress.md
```

### On Ice (Pausing)

If manager decides to pause work:

```
docs/planning/iced/goal-N/
├── task_plan.md
├── findings.md
└── progress.md
```

## Goal Workflow

### Starting Work

1. Confirm you're in the correct goal worktree
2. Load `planning-with-files` skill
3. Create planning files at root
4. Work through phases
5. Commit work incrementally
6. Use `AskUserQuestion` when you need human input

### Stopping Work

You can stop at any time. The Stop hook notifies vega-hub, which:
- Logs "executor stopped" to goal markdown
- Sends desktop notification to human
- Updates UI with executor status

When you stop:
- Your commits are on the goal branch
- Planning files remain at worktree root
- Manager can check progress via git history
- Manager can read your planning files for context
- Manager decides next steps (continue, archive, ice)

### When Manager Asks to Archive (Goal Complete)

Only when the manager explicitly asks you to archive:

1. Move planning files:
   ```bash
   mkdir -p docs/planning/history/goal-N
   mv task_plan.md findings.md progress.md docs/planning/history/goal-N/
   ```
2. Commit the archive:
   ```bash
   git add docs/planning/
   git commit -m "docs(planning): archive goal #N"
   ```
3. Exit - manager runs `complete-goal.sh` to merge and cleanup

### When Manager Asks to Ice (Pausing)

Only when the manager explicitly asks to pause:

1. Move planning files to iced:
   ```bash
   mkdir -p docs/planning/iced/goal-N
   mv task_plan.md findings.md progress.md docs/planning/iced/goal-N/
   ```
2. Commit:
   ```bash
   git add docs/planning/
   git commit -m "docs(planning): ice goal #N - <reason>"
   ```
3. Exit - manager runs `ice-goal.sh` to cleanup worktree

## Git Commit Convention

```
<type>(<scope>): <description>

Goal: #N
Phase: X/Y

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

## Why Archive Planning Files?

Planning files contain valuable context:
- Task breakdowns and decisions
- Research findings
- Progress logs and lessons learned

Archiving them in `docs/planning/` preserves this knowledge for:
- Future reference when revisiting the feature
- Understanding why decisions were made
- Onboarding new team members to the codebase

**But archival is a deliberate step**, not automatic. The manager decides when a goal is truly complete.
