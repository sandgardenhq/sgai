# Slack Bot Integration

## Prerequisites

- `opencode` CLI installed and in PATH
- Slack workspace with admin access to create apps
- A running SGAI workspace directory (the `--root-dir`)

## Slack App Setup

1. Go to https://api.slack.com/apps, click **Create New App**, select **From scratch**.
2. Under **Settings > Socket Mode**, toggle on. Create an **App-Level Token** with scope `connections:write`. Save the `xapp-...` token.
3. Under **OAuth & Permissions > Scopes > Bot Token Scopes**, add:
   - `chat:write`
   - `app_mentions:read`
   - `channels:history` (and/or `groups:history`, `im:history`, `mpim:history` depending on channel types)
4. Under **Event Subscriptions**, toggle on. Subscribe to bot events:
   - `app_mention`
   - `message.channels` (and/or `message.groups`, `message.im`, `message.mpim`)
5. Under **Install App**, click **Install to Workspace**. Copy the `xoxb-...` Bot User OAuth Token.

## Configuration

| Variable | Required | Description |
|---|---|---|
| `SLACK_BOT_TOKEN` | Yes | Bot User OAuth Token (`xoxb-...`) |
| `SLACK_APP_TOKEN` | Yes | App-Level Token for Socket Mode (`xapp-...`) |
| `SLACK_ALLOWED_USERS` | Yes | Comma-separated Slack user IDs to allow (deny by default) |

## Running

```
SLACK_BOT_TOKEN=xoxb-... \
SLACK_APP_TOKEN=xapp-... \
SLACK_ALLOWED_USERS=U123ABC,U456DEF \
sgai slack-bot --root-dir /path/to/workspaces
```

Flags:
- `--root-dir` -- Root directory for workspaces (defaults to CWD)
- `--listen-addr` -- Internal HTTP/WebUI server address (default: `127.0.0.1:8080`)

The web UI is also available at the `--listen-addr` while the bot is running.

## Usage

- **@mention** the bot in any channel to start a conversation.
- The bot responds in a **thread** and tracks workspace context per thread.
- Ask the bot to **connect a workspace**: "connect to my-project workspace"
- Ask the bot to **list workspaces**: "what workspaces are available?"
- Ask the bot to **toggle event updates**: "enable event updates" (streams progress to thread)
- Ask the bot to **disconnect**: "disconnect this thread"
- When a workspace needs human input, the bot **proactively notifies** the thread.
- Only users listed in `SLACK_ALLOWED_USERS` can interact; unauthorized users are silently ignored.
