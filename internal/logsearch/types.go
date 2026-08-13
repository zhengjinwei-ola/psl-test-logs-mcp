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

type TraceInput struct {
	TraceID string   `json:"trace_id" jsonschema:"literal trace ID to correlate"`
	Sources []string `json:"sources" jsonschema:"one to eight configured log source names"`
	Limit   int      `json:"limit,omitempty" jsonschema:"maximum result lines across all sources"`
}

type TraceEntry struct {
	Source string `json:"source"`
	File   string `json:"file"`
	Line   string `json:"line"`
}

type TraceOutput struct {
	Entries      []TraceEntry `json:"entries"`
	ScannedFiles int          `json:"scanned_files"`
	ScannedBytes int64        `json:"scanned_bytes"`
	Truncated    bool         `json:"truncated"`
}
