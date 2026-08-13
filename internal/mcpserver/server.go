package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/logsearch"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/runtimeinfo"
)

const (
	Name    = "psl-test-logs"
	Version = "v0.2.0"
)

type logService interface {
	ListSources(ctx context.Context, input logsearch.ListSourcesInput) (logsearch.ListSourcesOutput, error)
	Search(ctx context.Context, input logsearch.SearchInput) (logsearch.SearchOutput, error)
	Trace(ctx context.Context, input logsearch.TraceInput) (logsearch.TraceOutput, error)
}

type runtimeService interface {
	ListSources(ctx context.Context, input runtimeinfo.ListSourcesInput) (runtimeinfo.ListSourcesOutput, error)
	Get(ctx context.Context, input runtimeinfo.GetInput) (runtimeinfo.GetOutput, error)
}

func New(service logService, runtime runtimeService) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: Name, Version: Version}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_test_log_sources",
		Description: "List the configured allowlisted PSL test log source names. Filesystem paths are not exposed.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input logsearch.ListSourcesInput) (*mcp.CallToolResult, logsearch.ListSourcesOutput, error) {
		output, err := service.ListSources(ctx, input)
		return nil, output, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "search_test_logs",
		Description: "Search recent content from one configured PSL test log source using bounded literal matching. " +
			"The server enforces allowlisted files, scan/output limits, path containment, and sensitive-value redaction.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input logsearch.SearchInput) (*mcp.CallToolResult, logsearch.SearchOutput, error) {
		output, err := service.Search(ctx, input)
		return nil, output, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "trace_test_logs",
		Description: "Correlate one literal trace ID across one to eight allowlisted PSL test log sources. " +
			"The server applies one shared scan/output budget and redacts sensitive values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input logsearch.TraceInput) (*mcp.CallToolResult, logsearch.TraceOutput, error) {
		output, err := service.Trace(ctx, input)
		return nil, output, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_test_runtime_sources",
		Description: "List allowlisted PSL test service runtime source names without exposing filesystem paths.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input runtimeinfo.ListSourcesInput) (*mcp.CallToolResult, runtimeinfo.ListSourcesOutput, error) {
		output, err := runtime.ListSources(ctx, input)
		return nil, output, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_test_service_runtime",
		Description: "Inspect allowlisted PSL test service binaries and configuration metadata. " +
			"Returns Go build/VCS information, config hashes, and only explicitly allowlisted non-sensitive config values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input runtimeinfo.GetInput) (*mcp.CallToolResult, runtimeinfo.GetOutput, error) {
		output, err := runtime.Get(ctx, input)
		return nil, output, err
	})
	return server
}
