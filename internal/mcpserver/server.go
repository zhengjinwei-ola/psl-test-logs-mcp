package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/logsearch"
)

const (
	Name    = "psl-test-logs"
	Version = "v0.1.0"
)

type logService interface {
	ListSources(ctx context.Context, input logsearch.ListSourcesInput) (logsearch.ListSourcesOutput, error)
	Search(ctx context.Context, input logsearch.SearchInput) (logsearch.SearchOutput, error)
}

func New(service logService) *mcp.Server {
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
	return server
}
