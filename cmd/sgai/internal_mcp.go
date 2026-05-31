package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const sgaiAgentIdentityHeader = "X-Sgai-Agent-Identity"

type identityRoundTripper struct {
	identity string
	base     http.RoundTripper
}

func (rt identityRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	next := req.Clone(req.Context())
	next.Header.Set(sgaiAgentIdentityHeader, rt.identity)
	return base.RoundTrip(next)
}

func runInternalMCP(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: sgai internal-mcp <mcp-url> <agent-identity>")
	}
	mcpURL := strings.TrimSpace(args[0])
	agentIdentity := strings.TrimSpace(args[1])
	if mcpURL == "" {
		return fmt.Errorf("mcp url is required")
	}
	if agentIdentity == "" {
		return fmt.Errorf("agent identity is required")
	}

	server, closeProxy, errProxy := buildInternalMCPProxyServer(ctx, mcpURL, agentIdentity)
	if errProxy != nil {
		return errProxy
	}
	defer closeProxy()

	errRun := server.Run(ctx, &mcp.StdioTransport{})
	if isNormalMCPShutdown(errRun) {
		return nil
	}
	return errRun
}

func buildInternalMCPProxyServer(ctx context.Context, mcpURL, agentIdentity string) (*mcp.Server, func(), error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "sgai-internal-proxy"}, nil)
	clientSession, errConnect := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: mcpURL,
		HTTPClient: &http.Client{
			Timeout:   0,
			Transport: identityRoundTripper{identity: agentIdentity},
		},
	}, nil)
	if errConnect != nil {
		return nil, nil, fmt.Errorf("connecting to upstream mcp: %w", errConnect)
	}
	closeProxy := func() {
		_ = clientSession.Close()
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "sgai"}, nil)
	tools, errTools := listAllMCPTools(ctx, clientSession)
	if errTools != nil {
		closeProxy()
		return nil, nil, errTools
	}
	for _, tool := range tools {
		upstreamTool := *tool
		server.AddTool(&upstreamTool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return clientSession.CallTool(ctx, &mcp.CallToolParams{
				Meta:      req.Params.Meta,
				Name:      req.Params.Name,
				Arguments: req.Params.Arguments,
			})
		})
	}
	return server, closeProxy, nil
}

func listAllMCPTools(ctx context.Context, clientSession *mcp.ClientSession) ([]*mcp.Tool, error) {
	var result []*mcp.Tool
	var cursor string
	for {
		tools, errTools := clientSession.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if errTools != nil {
			return nil, fmt.Errorf("listing upstream mcp tools: %w", errTools)
		}
		result = append(result, tools.Tools...)
		if tools.NextCursor == "" {
			return result, nil
		}
		cursor = tools.NextCursor
	}
}

func isNormalMCPShutdown(err error) bool {
	if err == nil {
		return true
	}
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, mcp.ErrConnectionClosed)
}
