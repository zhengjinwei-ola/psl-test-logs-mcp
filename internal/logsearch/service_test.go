package logsearch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/olaola-chat/psl-test-logs-mcp/internal/appconfig"
)

func TestSearchReturnsNewestMatchesAndRedactsSecrets(t *testing.T) {
	directory := t.TempDir()
	writeTestLog(t, filepath.Join(directory, "app.log"), strings.Join([]string{
		"old uid=816220425 password=hunter2",
		"unrelated line",
		"new UID=816220425 Authorization: Bearer abc.def",
	}, "\n"))
	service := newTestService(t, directory)

	output, err := service.Search(context.Background(), SearchInput{Source: "gk-user", Query: "uid=816220425", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(output.Matches))
	}
	if !strings.HasPrefix(output.Matches[0].Line, "new UID") {
		t.Fatalf("first match = %q, want newest line", output.Matches[0].Line)
	}
	for _, match := range output.Matches {
		if strings.Contains(match.Line, "hunter2") || strings.Contains(match.Line, "abc.def") {
			t.Fatalf("secret was not redacted: %q", match.Line)
		}
		if !strings.Contains(match.Line, "[REDACTED]") {
			t.Fatalf("redaction marker missing: %q", match.Line)
		}
	}
}

func TestSearchEnforcesSourceAndLimit(t *testing.T) {
	directory := t.TempDir()
	writeTestLog(t, filepath.Join(directory, "app.log"), "one\ntwo\nthree\n")
	service := newTestService(t, directory)

	if _, err := service.Search(context.Background(), SearchInput{Source: "missing"}); err == nil {
		t.Fatal("unknown source was accepted")
	}
	if _, err := service.Search(context.Background(), SearchInput{Source: "gk-user", Limit: 6}); err == nil {
		t.Fatal("limit above configured maximum was accepted")
	}
	output, err := service.Search(context.Background(), SearchInput{Source: "gk-user", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 2 || !output.Truncated {
		t.Fatalf("matches = %d, truncated = %t", len(output.Matches), output.Truncated)
	}
}

func TestSearchSkipsSymlinkOutsideAllowlistedRoot(t *testing.T) {
	directory := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.log")
	writeTestLog(t, outside, "must-not-leak\n")
	if err := os.Symlink(outside, filepath.Join(directory, "escape.log")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	service := newTestService(t, directory)

	output, err := service.Search(context.Background(), SearchInput{Source: "gk-user", Query: "must-not-leak", Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 0 || output.ScannedFiles != 0 {
		t.Fatalf("outside symlink was scanned: %+v", output)
	}
}

func TestSearchHonorsCancelledContext(t *testing.T) {
	directory := t.TempDir()
	writeTestLog(t, filepath.Join(directory, "app.log"), "line\n")
	service := newTestService(t, directory)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Search(ctx, SearchInput{Source: "gk-user", Limit: 1})
	if err == nil {
		t.Fatal("cancelled context was ignored")
	}
}

func TestReadTailReportsPhysicalBytesBeforeDroppingPartialLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	writeTestLog(t, path, "first-line\nsecond-line\n")
	data, readBytes, err := readTail(path, 15)
	if err != nil {
		t.Fatal(err)
	}
	if readBytes != 15 {
		t.Fatalf("read bytes = %d, want 15", readBytes)
	}
	if string(data) != "second-line\n" {
		t.Fatalf("tail data = %q", data)
	}
}

func newTestService(t *testing.T, directory string) *Service {
	t.Helper()
	service, err := NewService(appconfig.Config{
		Sources:        []appconfig.Source{{Name: "gk-user", Patterns: []string{filepath.Join(directory, "*.log")}}},
		MaxFiles:       3,
		MaxScanBytes:   4096,
		MaxResults:     5,
		MaxLineBytes:   256,
		MaxOutputBytes: 4096,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func writeTestLog(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
