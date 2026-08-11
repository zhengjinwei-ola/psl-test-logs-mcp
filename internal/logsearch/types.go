package logsearch

type ListSourcesInput struct{}

type SourceInfo struct {
	Name string `json:"name"`
}

type ListSourcesOutput struct {
	Sources []SourceInfo `json:"sources"`
}

type SearchInput struct {
	Source        string `json:"source" jsonschema:"configured log source name"`
	Query         string `json:"query,omitempty" jsonschema:"literal text to match; empty returns recent lines"`
	Limit         int    `json:"limit,omitempty" jsonschema:"maximum result lines"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" jsonschema:"use case-sensitive literal matching"`
}

type LogEntry struct {
	File string `json:"file"`
	Line string `json:"line"`
}

type SearchOutput struct {
	Source       string     `json:"source"`
	Matches      []LogEntry `json:"matches"`
	ScannedFiles int        `json:"scanned_files"`
	ScannedBytes int64      `json:"scanned_bytes"`
	Truncated    bool       `json:"truncated"`
}
