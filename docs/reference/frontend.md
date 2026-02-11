# Frontend (React web dashboard)

The SGAI web dashboard is a React single-page application (SPA). The Go `sgai` binary embeds the built frontend assets.

## What you'll do here

- Install frontend dependencies
- Run the frontend in development (with an API proxy)
- Build the production frontend assets
- Run frontend unit/component tests

## Prerequisites

- [bun](https://bun.sh) installed
- A running SGAI backend to proxy to when using the dev server

## Project location

The frontend project lives at `cmd/sgai/webapp/`.

## Install dependencies

1. Change into the webapp directory:

   ```sh
   cd cmd/sgai/webapp
   ```

2. Install dependencies:

   ```sh
   bun install
   ```

## Run the dev server

The repo includes a Bun-based dev server script that serves the SPA and proxies API requests.

1. Start the SGAI backend.

2. In another terminal, start the dev server:

   ```sh
   cd cmd/sgai/webapp
   bun run dev.ts
   ```

3. Use the dev server URL printed in your terminal.

### Configure the API proxy target

The dev server proxies `/api/*` requests to a backend target.

- Set `API_TARGET` to change the backend URL (defaults to `http://127.0.0.1:8181`).
- Set `DEV_PORT` to change the dev server port (defaults to `5173`).

Example:

```sh
API_TARGET=http://127.0.0.1:8181 DEV_PORT=5173 bun run dev.ts
```

## Build for production

1. Build the frontend into `dist/`:

   ```sh
   cd cmd/sgai/webapp
   bun run build
   ```

2. Build the Go binary (which embeds the built assets):

   ```sh
   cd ../../..
   make build
   ```

## Run frontend tests

Run unit/component tests:

```sh
cd cmd/sgai/webapp
bun test src/
```

## Next steps

- See the CLI reference: [CLI](./cli.md)
- Configure runtime behavior via environment variables: [Environment variables](./environment-variables.md)