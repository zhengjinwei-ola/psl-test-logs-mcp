package runtimeinfo

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/olaola-chat/psl-test-logs-mcp/internal/appconfig"
	"gopkg.in/yaml.v3"
)

const maxConfigBytes = 2 << 20

type runtimeSource struct {
	name        string
	root        string
	binaries    []string
	configFiles []appconfig.RuntimeConfigFile
}

type Service struct {
	sources map[string]runtimeSource
	logger  *log.Logger
}

func NewService(config appconfig.Config, logger *log.Logger) (*Service, error) {
	sources := make(map[string]runtimeSource, len(config.RuntimeSources))
	for _, configured := range config.RuntimeSources {
		root := filepath.Clean(configured.Root)
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}
		for _, file := range configured.ConfigFiles {
			for _, key := range file.AllowedKeys {
				if sensitiveConfigKey(key) {
					return nil, fmt.Errorf("runtime source %q config key %q is sensitive", configured.Name, key)
				}
			}
		}
		sources[configured.Name] = runtimeSource{
			name: configured.Name, root: root,
			binaries:    append([]string(nil), configured.Binaries...),
			configFiles: append([]appconfig.RuntimeConfigFile(nil), configured.ConfigFiles...),
		}
	}
	return &Service{sources: sources, logger: logger}, nil
}

func (s *Service) ListSources(context.Context, ListSourcesInput) (ListSourcesOutput, error) {
	names := make([]string, 0, len(s.sources))
	for name := range s.sources {
		names = append(names, name)
	}
	sort.Strings(names)
	output := ListSourcesOutput{Sources: make([]SourceInfo, 0, len(names))}
	for _, name := range names {
		output.Sources = append(output.Sources, SourceInfo{Name: name})
	}
	return output, nil
}

func (s *Service) Get(ctx context.Context, input GetInput) (GetOutput, error) {
	started := time.Now()
	source, ok := s.sources[input.Source]
	if !ok {
		return GetOutput{}, fmt.Errorf("unknown runtime source %q", input.Source)
	}
	output := GetOutput{
		Source:      input.Source,
		Binaries:    make([]BinaryInfo, 0, len(source.binaries)),
		ConfigFiles: make([]ConfigInfo, 0, len(source.configFiles)),
	}
	for _, name := range source.binaries {
		if err := ctx.Err(); err != nil {
			return GetOutput{}, err
		}
		output.Binaries = append(output.Binaries, inspectBinary(source.root, name))
	}
	for _, file := range source.configFiles {
		if err := ctx.Err(); err != nil {
			return GetOutput{}, err
		}
		output.ConfigFiles = append(output.ConfigFiles, inspectConfig(source.root, file))
	}
	if s.logger != nil {
		s.logger.Printf("tool=get_test_service_runtime source=%q binaries=%d configs=%d duration_ms=%d",
			input.Source, len(output.Binaries), len(output.ConfigFiles), time.Since(started).Milliseconds())
	}
	return output, nil
}

func inspectBinary(root, name string) BinaryInfo {
	output := BinaryInfo{Name: name}
	path, err := resolveWithinRoot(root, name)
	if err != nil {
		output.Error = err.Error()
		return output
	}
	stat, err := os.Stat(path)
	if err != nil || !stat.Mode().IsRegular() {
		output.Error = safeFileError(err, "binary is not a regular file")
		return output
	}
	output.SizeBytes = stat.Size()
	output.ModifiedAt = stat.ModTime().UTC().Format(time.RFC3339)
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		output.Error = "Go build metadata unavailable"
		return output
	}
	output.GoVersion = info.GoVersion
	output.Module = info.Main.Path
	output.Version = info.Main.Version
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			output.VCSRevision = setting.Value
		case "vcs.time":
			output.VCSTime = setting.Value
		case "vcs.modified":
			output.VCSModified = setting.Value
		}
	}
	return output
}

func inspectConfig(root string, configured appconfig.RuntimeConfigFile) ConfigInfo {
	output := ConfigInfo{Name: configured.Path}
	path, err := resolveWithinRoot(root, configured.Path)
	if err != nil {
		output.Error = err.Error()
		return output
	}
	stat, err := os.Stat(path)
	if err != nil || !stat.Mode().IsRegular() {
		output.Error = safeFileError(err, "config is not a regular file")
		return output
	}
	if stat.Size() > maxConfigBytes {
		output.Error = fmt.Sprintf("config exceeds %d bytes", maxConfigBytes)
		return output
	}
	data, err := os.ReadFile(path)
	if err != nil {
		output.Error = "config cannot be read"
		return output
	}
	digest := sha256.Sum256(data)
	output.SizeBytes = stat.Size()
	output.ModifiedAt = stat.ModTime().UTC().Format(time.RFC3339)
	output.SHA256 = hex.EncodeToString(digest[:])
	if len(configured.AllowedKeys) == 0 {
		return output
	}
	var decoded map[string]any
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		output.Error = "config format cannot be parsed"
		return output
	}
	output.Values = make(map[string]string)
	for _, key := range configured.AllowedKeys {
		value, ok := lookupScalar(decoded, strings.Split(key, "."))
		if ok {
			output.Values[key] = value
		}
	}
	return output
}

func resolveWithinRoot(root, relative string) (string, error) {
	path := filepath.Join(root, relative)
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("configured file is unavailable")
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("configured file escapes runtime root")
	}
	return resolved, nil
}

func lookupScalar(current any, path []string) (string, bool) {
	for _, part := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return "", false
		}
		current, ok = object[part]
		if !ok {
			return "", false
		}
	}
	switch value := current.(type) {
	case string:
		if len(value) > 512 {
			return value[:512] + "...[TRUNCATED]", true
		}
		return value, true
	case bool, int, int64, uint64, float64:
		return fmt.Sprint(value), true
	default:
		return "", false
	}
}

func sensitiveConfigKey(key string) bool {
	lower := strings.ToLower(key)
	for _, token := range []string{"password", "passwd", "secret", "token", "cookie", "session", "authorization", "credential", "private_key", "access_key", ".master", ".slaves", "otel.path"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func safeFileError(err error, fallback string) string {
	if err != nil {
		return "configured file is unavailable"
	}
	return fallback
}
