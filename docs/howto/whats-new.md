# What's new

## 0.0.0+20260215 — Additional changes

- **Date**: 2026-02-15
- **Version**: 0.0.0+20260215
- **Summary**: This release includes the changes listed below.

### Additional Changes

```json
{
  "New Features": [
    "Pinned workspaces are now handled in the macOS menu bar so the correct workspace stays available and easy to access. The macOS menu bar integration now includes pinned workspace handling in the menu bar item UI/state management.",
    "The menu bar now starts as part of the server process so it is available earlier and is managed more consistently. The macOS menu bar component is now initialized within the server lifecycle instead of being started separately.",
    "macOS notifications now use the native notification system instead of relying on AppleScript, improving reliability and behavior consistency. Notifications have been migrated from `osascript` to a native Cocoa implementation using `UserNotifications` (UNUserNotificationCenter)."
  ]
}

```