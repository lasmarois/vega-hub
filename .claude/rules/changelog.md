---
paths:
  - "CHANGELOG.md"
  - "VERSION"
---

# Changelog Management

> Following [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions.

## When to Update CHANGELOG.md

Update for **user-facing changes only**:
- New features or capabilities
- Breaking changes or removed functionality
- Bug fixes that affect users
- Security patches

**Do NOT add entries for:**
- Internal refactoring with no user impact
- Documentation-only changes
- Test additions/fixes
- CI/CD workflow changes
- Code style/formatting changes

## Format

```markdown
## [Unreleased]

### Added
- New feature description

### Changed
- Modified behavior description

### Deprecated
- Feature scheduled for removal

### Removed
- Removed feature description

### Fixed
- Bug fix description

### Security
- Security patch description
```

## Section Types

| Section | Purpose |
|---------|---------|
| `Added` | New features |
| `Changed` | Changes to existing functionality |
| `Deprecated` | Features marked for future removal |
| `Removed` | Features that were removed |
| `Fixed` | Bug fixes |
| `Security` | Security vulnerability fixes |

## Rules

1. **Keep entries in `[Unreleased]`** until release - never create version sections manually
2. **Write for humans** - describe the impact, not the implementation
3. **One line per change** - be concise but complete
4. **Use present tense** - "Add feature" not "Added feature"
5. **No commit hashes or PR numbers** - the changelog is for users

## Release Process

1. Move entries from `[Unreleased]` to new version section `[X.Y.Z] - YYYY-MM-DD`
2. Bump version in `VERSION` file (follows [Semantic Versioning](https://semver.org/))
3. Commit and push to master
4. The release workflow automatically:
   - Builds multi-platform binaries (linux/darwin × amd64/arm64)
   - Creates a git tag
   - Creates a GitHub release with changelog as release notes

## Version Bumping

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes, incompatible API changes
- **MINOR** (1.0.0 → 1.1.0): New features, backward compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes, backward compatible
