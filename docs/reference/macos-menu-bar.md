# macOS menu bar and notifications

This page explains what the `sgai` macOS menu bar shows, and how local notifications work.

## Prerequisites

- A Mac running `sgai`
- The `sgai` menu bar app is running (for example, after `sgai serve` starts)

## Menu bar: which items appear

The menu bar shows a subset of projects.

An item appears when at least one of these is true:

- The project is pinned.
- The project needs input.
- The project is stopped.

## Menu bar: item icons and labels

The menu bar uses different symbols in the item label to help you quickly spot status.

- Needs input: the item is labeled as needing attention.
- Running and pinned: `▶ <name> (Running)`
- Pinned (not running): `○ <name>`
- Stopped: `■ <name> (Stopped)`
- Other items: `<name>`

## Local notifications on macOS

On macOS, `sgai` sends local notifications using native macOS notification APIs.

- When the `sgai` process has a bundle identifier, notifications use the UserNotifications framework.
- When the `sgai` process does not have a bundle identifier, notifications fall back to the legacy `NSUserNotification` API.
