package logsearch

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/olaola-chat/psl-test-logs-mcp/internal/appconfig"
)

type source struct {
	name     string
	patterns []string
	roots    []string
}

type Service struct {
	sources        map[string]source
	maxFiles       int
	maxScanBytes   int64
	maxResults     int
	maxLineBytes   int
	maxOutputBytes int
	redactor       redactor
	logger         *log.Logger
}

type candidate struct {
	path    string
	modTime time.Time
	size    int64
}

func NewService(config appconfig.Config, logger *log.Logger) (*Service, error) {
	sources := make(map[string]source, len(config.Sources))
	for _, configured := range config.Sources {
		item := source{name: configured.Name, patterns: append([]string(nil), configured.Patterns...)}
		for _, pattern := range configured.Patterns {
			root, err := patternRoot(pattern)
			if err != nil {
				return nil, fmt.Errorf("source %q: %w", configured.Name, err)
			}
			resolved, err := filepath.EvalSymlinks(root)
			if err != nil {
				return nil, fmt.Errorf("source %q root %q: %w", configured.Name, root, err)
			}
			item.roots = append(item.roots, resolved)
		}
		sources[item.name] = item
	}
	return &Service{
		sources:        sources,
		maxFiles:       config.MaxFiles,
		maxScanBytes:   config.MaxScanBytes,
		maxResults:     config.MaxResults,
		maxLineBytes:   config.MaxLineBytes,
		maxOutputBytes: config.MaxOutputBytes,
		redactor:       newRedactor(),
		logger:         logger,
	}, nil
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

func (s *Service) Search(ctx context.Context, input SearchInput) (SearchOutput, error) {
	started := time.Now()
	configured, ok := s.sources[input.Source]
	if !ok {
		return SearchOutput{}, fmt.Errorf("unknown log source %q", input.Source)
	}
	limit := input.Limit
	if limit == 0 {
		limit = 200
	}
	if limit < 1 || limit > s.maxResults {
		return SearchOutput{}, fmt.Errorf("limit must be between 1 and %d", s.maxResults)
	}
	if len(input.Query) > 512 {
		return SearchOutput{}, fmt.Errorf("query is longer than 512 bytes")
	}

	files, filesTruncated, err := s.resolveFiles(configured)
	if err != nil {
		return SearchOutput{}, err
	}
	output := SearchOutput{Source: input.Source, Matches: make([]LogEntry, 0), Truncated: filesTruncated}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return SearchOutput{}, err
		}
		remaining := s.maxScanBytes - output.ScannedBytes
		if remaining <= 0 {
			output.Truncated = true
			break
		}
		readLimit := file.size
		if readLimit > remaining {
			readLimit = remaining
			output.Truncated = true
		}
		data, readBytes, err := readTail(file.path, readLimit)
		if err != nil {
			return SearchOutput{}, fmt.Errorf("read log file: %w", err)
		}
		output.ScannedFiles++
		output.ScannedBytes += readBytes
		s.collectMatches(ctx, &output, filepath.Base(file.path), data, input, limit)
		if err := ctx.Err(); err != nil {
			return SearchOutput{}, err
		}
		if len(output.Matches) >= limit || outputSize(output.Matches) >= s.maxOutputBytes {
			output.Truncated = true
			break
		}
	}
	if s.logger != nil {
		hash := sha256.Sum256([]byte(input.Query))
		s.logger.Printf("tool=search_test_logs source=%q query_hash=%x files=%d bytes=%d matches=%d truncated=%t duration_ms=%d",
			input.Source, hash[:8], output.ScannedFiles, output.ScannedBytes, len(output.Matches), output.Truncated, time.Since(started).Milliseconds())
	}
	return output, nil
}

func (s *Service) collectMatches(ctx context.Context, output *SearchOutput, name string, data []byte, input SearchInput, limit int) {
	lines := bytes.Split(data, []byte{'\n'})
	query := input.Query
	if !input.CaseSensitive {
		query = strings.ToLower(query)
	}
	for index := len(lines) - 1; index >= 0 && len(output.Matches) < limit; index-- {
		if ctx.Err() != nil {
			return
		}
		line := strings.TrimSuffix(string(lines[index]), "\r")
		if line == "" {
			continue
		}
		candidateLine := line
		if !input.CaseSensitive {
			candidateLine = strings.ToLower(candidateLine)
		}
		if query != "" && !strings.Contains(candidateLine, query) {
			continue
		}
		if len(line) > s.maxLineBytes {
			line = line[:s.maxLineBytes] + "...[TRUNCATED]"
			output.Truncated = true
		}
		line = s.redactor.Redact(line)
		if outputSize(output.Matches)+len(name)+len(line) > s.maxOutputBytes {
			output.Truncated = true
			return
		}
		output.Matches = append(output.Matches, LogEntry{File: name, Line: line})
	}
}

func (s *Service) resolveFiles(configured source) ([]candidate, bool, error) {
	unique := make(map[string]candidate)
	for index, pattern := range configured.patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, false, fmt.Errorf("invalid configured glob for %q", configured.name)
		}
		for _, match := range matches {
			resolved, err := filepath.EvalSymlinks(match)
			if err != nil {
				continue
			}
			if !withinRoot(configured.roots[index], resolved) {
				continue
			}
			info, err := os.Stat(resolved)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			unique[resolved] = candidate{path: resolved, modTime: info.ModTime(), size: info.Size()}
		}
	}
	files := make([]candidate, 0, len(unique))
	for _, file := range unique {
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.After(files[j].modTime) })
	truncated := len(files) > s.maxFiles
	if truncated {
		files = files[:s.maxFiles]
	}
	return files, truncated, nil
}

func patternRoot(pattern string) (string, error) {
	index := strings.IndexAny(pattern, "*?[")
	root := pattern
	if index >= 0 {
		root = filepath.Dir(pattern[:index])
	} else {
		root = filepath.Dir(pattern)
	}
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("pattern root %q: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("pattern root %q is not a directory", root)
	}
	return root, nil
}

func withinRoot(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func readTail(path string, limit int64) ([]byte, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	offset := info.Size() - limit
	if offset < 0 {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, 0, err
	}
	readBytes := int64(len(data))
	if offset > 0 {
		if newline := bytes.IndexByte(data, '\n'); newline >= 0 {
			data = data[newline+1:]
		} else {
			data = nil
		}
	}
	return data, readBytes, nil
}

func outputSize(matches []LogEntry) int {
	total := 0
	for _, match := range matches {
		total += len(match.File) + len(match.Line)
	}
	return total
}
