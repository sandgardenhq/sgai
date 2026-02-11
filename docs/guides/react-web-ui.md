# React web UI (sgai serve)

Hey there! This guide walks through working on the `sgai` web interface, which is a React single-page application (SPA) that uses server-sent events (SSE) for real-time updates.

## What you’ll learn

- Where the frontend code lives
- How the frontend is built and embedded into the Go binary
- How to run a local frontend dev server
- Which endpoints the UI calls (`/api/v1/*` + SSE stream)

## Prerequisites

- `bun` (used to install dependencies, build, and run tests)
- A working `sgai` Go build environment

## Project layout

- Frontend source lives in `cmd/sgai/webapp/`.
- A production build outputs to `cmd/sgai/webapp/dist/`.
- The built assets are embedded into the `sgai` binary using Go’s `//go:embed`.

## Build the frontend

Run these commands from the repository root:

1. Install dependencies:

   ```sh
   cd cmd/sgai/webapp && bun install
   ```

2. Build the SPA into `dist/`:

   ```sh
   bun run build
   ```

## Build the Go binary (includes the embedded frontend)

`make build` runs the frontend build step (`bun install` + `bun run build.ts`) and then builds the Go binary.

```sh
make build
```

If frontend code changed, build the SPA first and then run the Go build:

```sh
cd cmd/sgai/webapp && bun run build && cd ../../..
make build
```

## Run the frontend dev server

The webapp includes a Bun dev server script:

```sh
cd cmd/sgai/webapp
bun run dev.ts
```

## API endpoints used by the React UI

The React frontend uses a typed API client implemented in `cmd/sgai/webapp/src/lib/api.ts`. It calls JSON endpoints under `/api/v1/`.

Common workspace endpoints include:

- `GET /api/v1/workspaces`
- `GET /api/v1/workspaces/{name}`
- `POST /api/v1/workspaces` (create)
- `POST /api/v1/workspaces/{name}/start`
- `POST /api/v1/workspaces/{name}/stop`
- `POST /api/v1/workspaces/{name}/reset`

Compose endpoints include:

- `GET /api/v1/compose?workspace={workspace}`
- `POST /api/v1/compose?workspace={workspace}` (supports `If-Match`)
- `GET /api/v1/compose/templates`
- `GET /api/v1/compose/preview?workspace={workspace}`
- `POST /api/v1/compose/draft?workspace={workspace}`

The UI also calls endpoints for agents, skills, snippets, models, and workspace actions like fork/merge/rename.

## Real-time updates (SSE)

The UI connects to an SSE stream at:

- `GET /api/v1/events/stream`

Client-side SSE state lives in `cmd/sgai/webapp/src/lib/sse-store.ts` and is consumed via React hooks in `cmd/sgai/webapp/src/hooks/useSSE.ts`.

The UI subscribes to event types including:

- `workspace:update`
- `session:update`
- `messages:new`
- `todos:update`
- `log:append`
- `changes:update`
- `events:new`
- `compose:update`

## Troubleshooting

### The UI shows a yellow “Disconnected” banner

The banner comes from `ConnectionStatusBanner` and appears when the SSE connection stays non-connected for more than 2 seconds.

Check these items:

1. Start `sgai serve` (the SSE stream lives on the Go server).
2. Confirm the browser can reach `/api/v1/events/stream`.

## Next steps

- Explore the route-based pages in `cmd/sgai/webapp/src/pages/`.
- Check the workspace dashboard layout in `cmd/sgai/webapp/src/pages/Dashboard.tsx`.
