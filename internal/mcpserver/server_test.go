package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/logsearch"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/runtimeinfo"
)

type stubService struct {
	searchCalls int
}

type stubRuntimeService struct {
	getCalls int
}

func (s *stubService) ListSources(context.Context, logsearch.ListSourcesInput) (logsearch.ListSourcesOutput, error) {
	return logsearch.ListSourcesOutput{Sources: []logsearch.SourceInfo{{Name: "gk-user"}}}, nil
}

func (s *stubService) Search(_ context.Context, input logsearch.SearchInput) (logsearch.SearchOutput, error) {
	s.searchCalls++
	return logsearch.SearchOutput{Source: input.Source, Matches: []logsearch.LogEntry{{File: "app.log", Line: "match"}}}, nil
}

func (s *stubService) Trace(_ context.Context, input logsearch.TraceInput) (logsearch.TraceOutput, error) {
	return logsearch.TraceOutput{Entries: []logsearch.TraceEntry{{Source: input.Sources[0], File: "app.log", Line: "trace"}}}, nil
}

func (s *stubRuntimeService) ListSources(context.Context, runtimeinfo.ListSourcesInput) (runtimeinfo.ListSourcesOutput, error) {
	return runtimeinfo.ListSourcesOutput{Sources: []runtimeinfo.SourceInfo{{Name: "gk-user"}}}, nil
}

func (s *stubRuntimeService) Get(_ context.Context, input runtimeinfo.GetInput) (runtimeinfo.GetOutput, error) {
	s.getCalls++
	return runtimeinfo.GetOutput{Source: input.Source}, nil
}

func TestToolsCanBeCalledOverMCP(t *testing.T) {
	ctx := context.Background()
	service := &stubService{}
	runtimeService := &stubRuntimeService{}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := New(service, runtimeService).Connect(ctx, serverTransport, nil)
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
	traceResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "trace_test_logs", Arguments: map[string]any{"trace_id": "abcdef123456", "sources": []any{"gk-user"}},
	})
	if err != nil || traceResult.IsError {
		t.Fatalf("trace result = %+v, error = %v", traceResult, err)
	}
	runtimeResult, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_test_service_runtime", Arguments: map[string]any{"source": "gk-user"},
	})
	if err != nil || runtimeResult.IsError || runtimeService.getCalls != 1 {
		t.Fatalf("runtime result = %+v, error = %v, calls = %d", runtimeResult, err, runtimeService.getCalls)
	}
}
