package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureRoundTripper struct {
	header http.Header
}

func (rt *captureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.header = req.Header.Clone()
	return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody, Request: req}, nil
}

func TestRequiresOpencodeSkipsInternalMCP(t *testing.T) {
	assert.False(t, requiresOpencode("internal-mcp"))
	assert.False(t, requiresOpencode("help"))
	assert.True(t, requiresOpencode("serve"))
}

func TestRunInternalMCPValidatesArguments(t *testing.T) {
	err := runInternalMCP(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "usage")

	err = runInternalMCP(context.Background(), []string{"", "agent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mcp url is required")

	err = runInternalMCP(context.Background(), []string{"http://127.0.0.1:1/mcp", ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent identity is required")
}

func TestIdentityRoundTripperSetsAgentIdentityHeader(t *testing.T) {
	capture := &captureRoundTripper{}
	rt := identityRoundTripper{identity: "builder|model|variant", base: capture}
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	require.NoError(t, err)
	_, err = rt.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, "builder|model|variant", capture.header.Get(sgaiAgentIdentityHeader))
}

func TestIsNormalMCPShutdown(t *testing.T) {
	assert.True(t, isNormalMCPShutdown(nil))
	assert.True(t, isNormalMCPShutdown(context.Canceled))
	assert.True(t, isNormalMCPShutdown(mcp.ErrConnectionClosed))
	assert.False(t, isNormalMCPShutdown(assert.AnError))
}

func TestInternalMCPProxyForwardsIdentity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	upstream := mcp.NewServer(&mcp.Implementation{Name: "upstream"}, nil)
	mcp.AddTool(upstream, &mcp.Tool{Name: "identity", InputSchema: &jsonschema.Schema{Type: "object"}}, func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		identity := ""
		if req.Extra != nil {
			identity = req.Extra.Header.Get(sgaiAgentIdentityHeader)
		}
		return textResult(identity), nil, nil
	})
	mcp.AddTool(upstream, &mcp.Tool{Name: "meta", InputSchema: &jsonschema.Schema{Type: "object"}}, func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return textResult(fmt.Sprint(req.Params.Meta["progressToken"])), nil, nil
	})
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return upstream
	}, nil))
	t.Cleanup(httpServer.Close)

	proxy, closeProxy, errProxy := buildInternalMCPProxyServer(ctx, httpServer.URL, "builder|model|variant")
	require.NoError(t, errProxy)
	t.Cleanup(closeProxy)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- proxy.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	session, errConnect := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, errConnect)
	t.Cleanup(func() {
		_ = session.Close()
	})

	tools, errTools := session.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, errTools)
	require.Len(t, tools.Tools, 2)

	result, errCall := session.CallTool(ctx, &mcp.CallToolParams{Name: "identity"})
	require.NoError(t, errCall)
	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "builder|model|variant", text.Text)

	result, errCall = session.CallTool(ctx, &mcp.CallToolParams{Meta: mcp.Meta{"progressToken": "token-123"}, Name: "meta"})
	require.NoError(t, errCall)
	require.Len(t, result.Content, 1)
	text, ok = result.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, "token-123", text.Text)

	cancel()
	assert.True(t, isNormalMCPShutdown(<-serverDone))
}
