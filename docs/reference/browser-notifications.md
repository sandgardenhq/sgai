# Web dashboard browser notifications

The web dashboard can show **native browser notifications** when a workspace starts needing human input.

## What triggers a notification

A notification fires when a workspace transitions from `needsInput=false` to `needsInput=true`.

Notes:

- The check applies to workspaces and any nested forks.
- Notifications use the workspace name as the browser notification `tag`, which allows the browser to de-duplicate notifications for the same workspace.

## Enable notifications in the dashboard

1. Open the web dashboard in a browser.
2. Find the banner at the top: “Enable browser notifications to get alerted when a workspace needs your input.”
3. Select **Enable**.
4. When the browser asks, grant notification permission.

At this point, the banner disappears after permission is granted or denied.

## Dismiss the permission banner

Select **Dismiss** to hide the banner without responding to the browser permission prompt.

The dismissal is saved in the browser using `localStorage` under the key `notification-permission-dismissed`.

## Limitations

- Notifications only fire while the dashboard is open in a browser tab.
- No Service Worker notifications are used.
- Browsers without the `window.Notification` API do not show notifications or the permission banner.

## Troubleshooting

### No banner appears

- Check whether the browser supports the Notification API.
- Check whether notification permission is already set to something other than `default` (for example, `granted` or `denied`).
- Check whether the banner was dismissed previously (`notification-permission-dismissed=true` in `localStorage`).

### Notifications do not appear

1. Confirm the dashboard tab stays open.
2. Confirm the browser permission is `granted`.
3. Confirm a workspace actually transitioned into `needsInput=true`.

### Clicking a notification does not bring the dashboard forward

The notification click handler calls `window.focus()`.
Some browsers or OS configurations may still prevent focus changes.