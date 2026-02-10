# Phase 16 (complete): Distribution

- Created `.goreleaser.yaml` with builds for macOS and Linux (amd64, arm64), ldflags for version/commit/date injection, tar.gz archives with checksums, and Homebrew tap formula generation
- Created `.github/workflows/ci.yaml` — runs tests and golangci-lint on push to main and on PRs
- Created `.github/workflows/release.yaml` — runs goreleaser on version tag push (`v*`) to build binaries, create GitHub releases, and publish the Homebrew formula
- Verified `go install` compatibility: module root has `package main` with proper module path (`github.com/dslh/zh`)
- Confirmed build vars were already wired via Makefile ldflags into `cmd.Version`, `cmd.Commit`, `cmd.Date`
- Added `dist/` to `.gitignore` for goreleaser output
