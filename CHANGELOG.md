# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.1] - 2026-01-25

### Added
- `vega-hub goal cleanup` - Delete branch after MR/PR has been merged
  - Validates goal is in history (completed)
  - Finds and deletes the goal branch
  - AI-friendly error handling

This completes the MR/PR workflow:
1. `goal complete --no-merge` → preserves branch
2. Create MR/PR externally
3. MR/PR merged
4. `goal cleanup` → deletes branch

## [0.4.0] - 2026-01-25

### Added
- **Goal management CLI**
  - `vega-hub goal list` - List goals with filters (`--project`, `--status`)
  - `vega-hub goal create` - Create goal with worktree (`--base-branch`, `--no-worktree`)
  - `vega-hub goal complete` - Complete goal (merge, cleanup, `--no-merge`, `--force`)
  - `vega-hub goal ice` - Pause goal for later (`--force`)
- **Project management CLI**
  - `vega-hub project add` - Add project with git clone (`--branch`)
  - `vega-hub project list` - List registered projects
- **Executor management CLI**
  - `vega-hub executor spawn` - Spawn executor in goal worktree (`--prompt`)
  - `vega-hub executor list` - List active executors (`--goal`)
  - `vega-hub executor stop` - Stop running executor (`--reason`)
- **Worktree management CLI**
  - `vega-hub worktree create` - Create worktree for existing goal
  - `vega-hub worktree remove` - Remove worktree (`--force`)
  - `vega-hub worktree status` - Show worktree details (branch, uncommitted files)
- **Credentials validation CLI**
  - `vega-hub credentials check` - Validate git credentials for project
  - User detection via `os/user.Current()`
  - Support for GitHub (`gh auth`), GitLab (`glab auth`), SSH, and netrc
  - AI-friendly error messages with fix instructions
- **Port management improvements**
  - Spawn injects `VEGA_HUB_PORT` environment variable to executors
  - Lock file during startup to prevent race conditions

### Changed
- All CLI commands return structured JSON with `--json` flag
- Error responses include `options` array with actionable fixes
- Spawn API accepts `X-Vega-User` header for user tracking

## [0.3.0] - 2026-01-25

### Added
- Cobra CLI framework for subcommand support
- `vega-hub serve` command (HTTP server in foreground)
- `vega-hub start` command (daemon mode with automatic port selection)
- `vega-hub status` command (check if daemon is running)
- Global `--json` flag for structured JSON output
- Global `--quiet` flag for minimal output
- Global `--dir` flag to specify vega-missile directory
- AI-friendly error handling with exit codes and actionable messages

### Changed
- Refactored main.go to use Cobra root command
- Default behavior now shows help (use `serve` or `start` to run server)

## [0.2.0] - 2026-01-25

### Added
- Mobile-first responsive UI with shadcn components
- Goal list with filters (Active/Iced/Completed) and sorting
- GoalSheet with 5 tabs: Overview, Phases, Q&A, Planning, Timeline
- Expandable sidebar (toggle between 480px and 90vw)
- Notification center with bell icon dropdown
- Progress bars on goal cards
- Command palette (Cmd+K)
- Toast notifications for errors and status changes
- SSE auto-reconnect with connection status indicator
- File watching for real-time goal updates (fsnotify)

### Changed
- Redesigned navigation: sidebar on desktop, bottom nav on mobile
- Improved error handling with user-friendly messages

## [0.1.0] - 2025-01-24

### Added
- Initial release
- Go binary with embedded React UI
- Goal registry parsing from markdown
- Executor lifecycle management (spawn, register, stop)
- AskUserQuestion workflow via hooks
- SSE for real-time updates
- Q&A logging to goal markdown files
- Desktop notifications
