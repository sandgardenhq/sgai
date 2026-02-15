# Workspace event streaming (SSE)

`sgai` streams workspace updates over Server-Sent Events (SSE).

## Connect

Connect to the global workspace-update stream:

```text
GET /api/v1/events/stream
```

Connect to a specific workspace stream:

```text
GET /api/v1/workspaces/{name}/events/stream
```

The server sets these headers:

- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`
- `X-Accel-Buffering: no`

## Initial snapshot event

The stream sends an initial `snapshot` event.

The snapshot payload contains:

- `name`
- `running`
- `needsInput`
- `status`

## Workspace update event

The global stream can send a `workspace:update` event.

The event payload includes a `workspace` field with the workspace name.
