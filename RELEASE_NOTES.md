# Release Notes

## 0.0.0+20260316 — Security dependency updates

- **Date**: 2026-03-16
- **Version**: 0.0.0+20260316
- **Summary**: This release includes minor security-related dependency updates for Go dependencies and a webapp production dependency.

### Security Updates

- Refreshed the indirect Go dependency `golang.org/x/tools` from `0.42.0` to `0.43.0`, also updated indirect `golang.org/x/sys` from `0.41.0` to `0.42.0` in `go.mod`, and refreshed the corresponding `go.sum` checksum entries.
- Updated the webapp production dependency `lucide-react` in `cmd/sgai/webapp/package.json` from `^0.575.0` to `^0.577.0`.
- Strengthened the indirect Go dependency `go.yaml.in/yaml/v2` in `go.mod` from `v2.4.3` to `v2.4.4` and updated the matching `go.sum` checksum entries.

## 0.0.0+20260127 — Improved CI test reliability

- **Date**: 2026-01-27
- **Version**: 0.0.0+20260127
- **Summary**: This release includes improved CI and cross-platform test reliability.

### Bug Fixes

- Fixed CI tests to run reliably on Ubuntu and macOS by consolidating execution into a single shared test entry point, correcting directory-dependent assumptions to be path-independent, and removing unused parameters from the notification integration.
