package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/logsearch"
)

type stubService struct {
	searchCalls int
}

func (s *stubService) ListSources(context.Context, logsearch.ListSourcesInput) (logsearch.ListSourcesOutput, error) {
	return logsearch.ListSourcesOutput{Sources: []logsearch.SourceInfo{{Name: "gk-user"}}}, nil
}

func (s *stubService) Search(_ context.Context, input logsearch.SearchInput) (logsearch.SearchOutput, error) {
	s.searchCalls++
	return logsearch.SearchOutput{Source: input.Source, Matches: []logsearch.LogEntry{{File: "app.log", Line: "match"}}}, nil
}

func TestToolsCanBeCalledOverMCP(t *testing.T) {
	ctx := context.Background()
	service := &stubService{}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(service).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	listResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "list_test_log_sources", Arguments: map[string]any{}})
	if err != nil || listResult.IsError {
		t.Fatalf("list result = %+v, error = %v", listResult, err)
	}
	searchResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_test_logs",
		Arguments: map[string]any{"source": "gk-user", "query": "uid=1", "limit": float64(10)},
	})
	if err != nil || searchResult.IsError || service.searchCalls != 1 {
		t.Fatalf("search result = %+v, error = %v, calls = %d", searchResult, err, service.searchCalls)
	}
}
