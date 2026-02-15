# What's new

## 0.0.0+20260215 — Updated macOS menu bar and notifications

- **Date**: 2026-02-15
- **Version**: 0.0.0+20260215
- **Summary**: This release includes refined macOS menu bar handling and native notifications.

### New Features

- Added pinned workspace handling to the macOS menu bar so the correct workspace stays available and easy to access.
- Updated the macOS menu bar to start as part of the server process so it is available earlier and more consistently managed.
- Updated macOS notifications to use the native `UNUserNotificationCenter` API instead of `osascript` for improved reliability and consistent behavior.