package runtimeinfo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olaola-chat/psl-test-logs-mcp/internal/appconfig"
)

func TestServiceReturnsAllowlistedConfigAndDigest(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "configs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "configs", "config.yaml"), []byte("server:\n  env: dev\ndata:\n  redis:\n    password: secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(appconfig.Config{RuntimeSources: []appconfig.RuntimeSource{{
		Name: "gk-user", Root: root,
		ConfigFiles: []appconfig.RuntimeConfigFile{{Path: "configs/config.yaml", AllowedKeys: []string{"server.env"}}},
	}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	output, err := service.Get(context.Background(), GetInput{Source: "gk-user"})
	if err != nil {
		t.Fatal(err)
	}
	if output.ConfigFiles[0].Values["server.env"] != "dev" || len(output.ConfigFiles[0].SHA256) != 64 {
		t.Fatalf("output = %+v", output)
	}
}

func TestServiceRejectsSensitiveConfiguredKey(t *testing.T) {
	root := t.TempDir()
	_, err := NewService(appconfig.Config{RuntimeSources: []appconfig.RuntimeSource{{
		Name: "gk-user", Root: root,
		ConfigFiles: []appconfig.RuntimeConfigFile{{Path: "config.yaml", AllowedKeys: []string{"data.redis.password"}}},
	}}}, nil)
	if err == nil || !strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("error = %v", err)
	}
}

func TestServiceRejectsUnknownSource(t *testing.T) {
	service, err := NewService(appconfig.Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), GetInput{Source: "missing"}); err == nil {
		t.Fatal("expected unknown source error")
	}
}
