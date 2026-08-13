package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/appconfig"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/logsearch"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/mcpserver"
	"github.com/olaola-chat/psl-test-logs-mcp/internal/runtimeinfo"
)

const defaultConfigPath = "/etc/psl-test-logs-mcp/sources.json"

func main() {
	logger := log.New(os.Stderr, "psl-test-logs-mcp ", log.LstdFlags|log.LUTC)
	configPath := strings.TrimSpace(os.Getenv("PSL_LOG_MCP_CONFIG"))
	if configPath == "" {
		configPath = defaultConfigPath
	}
	flags := flag.NewFlagSet("psl-test-logs-mcp", flag.ExitOnError)
	flags.StringVar(&configPath, "config", configPath, "absolute path to the JSON source configuration")
	_ = flags.Parse(os.Args[1:])

	config, err := appconfig.Load(configPath)
	if err != nil {
		logger.Fatal(err)
	}
	service, err := logsearch.NewService(config, logger)
	if err != nil {
		logger.Fatal(err)
	}
	runtimeService, err := runtimeinfo.NewService(config, logger)
	if err != nil {
		logger.Fatal(err)
	}
	server := mcpserver.New(service, runtimeService)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Fatal(err)
	}
}
