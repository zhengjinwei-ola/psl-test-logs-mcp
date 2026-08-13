package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "sources.json")
	contents := `{"sources":[{"name":"gk-user","patterns":["` + filepath.ToSlash(directory) + `/*.log"]}]}`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.MaxFiles != defaultMaxFiles || config.MaxResults != defaultMaxResults || config.MaxScanBytes != defaultMaxScanBytes {
		t.Fatalf("defaults not applied: %+v", config)
	}
}

func TestLoadRejectsUnsafeOrAmbiguousConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     string
	}{
		{name: "relative pattern", contents: `{"sources":[{"name":"user","patterns":["logs/*.log"]}]}`, want: "must be absolute"},
		{name: "parent traversal", contents: `{"sources":[{"name":"user","patterns":["/var/log/../secret/*.log"]}]}`, want: "cannot contain .."},
		{name: "duplicate source", contents: `{"sources":[{"name":"user","patterns":["/var/log/a/*.log"]},{"name":"user","patterns":["/var/log/b/*.log"]}]}`, want: "duplicated"},
		{name: "unknown field", contents: `{"sources":[{"name":"user","patterns":["/var/log/a/*.log"]}],"write":true}`, want: "unknown field"},
		{name: "runtime traversal", contents: `{"sources":[{"name":"user","patterns":["/var/log/a/*.log"]}],"runtime_sources":[{"name":"user","root":"/srv/user","binaries":["../bin/app"]}]}`, want: "without traversal"},
		{name: "runtime absolute binary", contents: `{"sources":[{"name":"user","patterns":["/var/log/a/*.log"]}],"runtime_sources":[{"name":"user","root":"/srv/user","binaries":["/bin/app"]}]}`, want: "clean relative"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "sources.json")
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}
