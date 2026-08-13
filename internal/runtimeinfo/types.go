package runtimeinfo

type ListSourcesInput struct{}

type SourceInfo struct {
	Name string `json:"name"`
}

type ListSourcesOutput struct {
	Sources []SourceInfo `json:"sources"`
}

type GetInput struct {
	Source string `json:"source" jsonschema:"configured runtime source name"`
}

type BinaryInfo struct {
	Name        string `json:"name"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
	GoVersion   string `json:"go_version,omitempty"`
	Module      string `json:"module,omitempty"`
	Version     string `json:"version,omitempty"`
	VCSRevision string `json:"vcs_revision,omitempty"`
	VCSTime     string `json:"vcs_time,omitempty"`
	VCSModified string `json:"vcs_modified,omitempty"`
	Error       string `json:"error,omitempty"`
}

type ConfigInfo struct {
	Name       string            `json:"name"`
	SizeBytes  int64             `json:"size_bytes,omitempty"`
	ModifiedAt string            `json:"modified_at,omitempty"`
	SHA256     string            `json:"sha256,omitempty"`
	Values     map[string]string `json:"values,omitempty"`
	Error      string            `json:"error,omitempty"`
}

type GetOutput struct {
	Source      string       `json:"source"`
	Binaries    []BinaryInfo `json:"binaries"`
	ConfigFiles []ConfigInfo `json:"config_files"`
}
