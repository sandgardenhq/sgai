# MCP remote HTTP transport

SGAI exposes its Model Context Protocol (MCP) tools over HTTP.
This allows the workflow engine to connect to an MCP server using a URL, rather than using standard input/output.

## How the MCP server is addressed

SGAI configures the workflow engine to use an MCP server URL through the `SGAI_MCP_URL` environment variable.
Agent identity metadata is also passed via `SGAI_AGENT_IDENTITY`.

## HTTP endpoint and headers

The MCP server is hosted as an HTTP endpoint mounted at `/mcp`.
Agent identity is provided on each request using the `X-SGAI-Agent-Identity` header.

The header supports up to three `|`-separated values:

- `name`
- `model`
- `variant`

If the header is missing (or the name portion is empty), the agent name defaults to `coordinator`.

## Local development notes

- The MCP HTTP server binds to `127.0.0.1` and uses a dynamically assigned port.
- Each workspace starts its own MCP HTTP server for the lifetime of that workspace session.
