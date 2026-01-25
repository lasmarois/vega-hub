# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2025-01-25

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
