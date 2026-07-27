// Package codescan provides code intelligence by extracting structural
// information (packages, symbols, imports, dependencies) from source code.
package codescan

import "time"

// SymbolKind identifies the kind of a code symbol.
type SymbolKind string

const (
	SymFunc      SymbolKind = "func"
	SymMethod    SymbolKind = "method"
	SymType      SymbolKind = "type"
	SymInterface SymbolKind = "interface"
	SymStruct    SymbolKind = "struct"
	SymClass     SymbolKind = "class"
	SymConst     SymbolKind = "const"
	SymVar       SymbolKind = "var"
	SymEnum      SymbolKind = "enum"
	SymTrait     SymbolKind = "trait"
)

// Symbol represents a named code entity (function, type, class, etc.).
type Symbol struct {
	Name      string     `json:"name"`
	Kind      SymbolKind `json:"kind"`
	Exported  bool       `json:"exported"`
	Signature string     `json:"signature,omitempty"` // e.g. "func(ctx context.Context, id string) (*User, error)"
	Receiver  string     `json:"receiver,omitempty"`  // for methods: the receiver type
	File      string     `json:"file,omitempty"`      // relative file path (set during aggregation)
	Line      int        `json:"line"`                // 1-indexed line number
	Doc       string     `json:"doc,omitempty"`       // doc comment (first line only for brevity)
	AIDesc    string     `json:"ai_desc,omitempty"`   // LLM-generated description (deep mode only)
}

// FileInfo holds extracted information about a single source file.
type FileInfo struct {
	Path      string   `json:"path"`     // relative to project root
	Language  string   `json:"language"` // "go", "typescript", "python", etc.
	Package   string   `json:"package"`  // package/module this file belongs to
	Imports   []string `json:"imports,omitempty"`
	Symbols   []Symbol `json:"symbols,omitempty"`
	LineCount int      `json:"line_count"`
}

// Package represents a logical grouping of files (Go package, Python module, etc.).
type Package struct {
	Name       string   `json:"name"`
	Path       string   `json:"path"` // relative directory path
	Language   string   `json:"language"`
	Files      []string `json:"files"`
	Symbols    []Symbol `json:"symbols"`               // all exported symbols
	Imports    []string `json:"imports"`               // all unique imports
	ImportedBy []string `json:"imported_by,omitempty"` // packages that import this one
	LineCount  int      `json:"line_count"`
	FileCount  int      `json:"file_count"`
	Doc        string   `json:"doc,omitempty"`     // package-level doc
	AIDesc     string   `json:"ai_desc,omitempty"` // LLM-generated description (deep mode only)
}

// DepEdge represents a dependency relationship between two packages.
type DepEdge struct {
	From string `json:"from"` // importing package path
	To   string `json:"to"`   // imported package path
}

// HotFile represents a file ranked by importance (edit frequency, import count).
type HotFile struct {
	Path        string  `json:"path"`
	Score       float64 `json:"score"`
	EditCount   int     `json:"edit_count,omitempty"`   // git commits touching this file
	ImportCount int     `json:"import_count,omitempty"` // number of files importing this
	Reason      string  `json:"reason,omitempty"`       // why it's hot
}

// ConfigVar represents a detected environment variable read in source code.
type ConfigVar struct {
	Name     string `json:"name"`
	Source   string `json:"source"` // "env"
	File     string `json:"file"`
	Line     int    `json:"line"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required"`
}

// Result holds the complete code intelligence output for a project.
type Result struct {
	ProjectRoot string            `json:"project_root"`
	Packages    []Package         `json:"packages"`
	DepGraph    []DepEdge         `json:"dep_graph,omitempty"`
	HotFiles    []HotFile         `json:"hot_files,omitempty"`
	ConfigVars  []ConfigVar       `json:"config_vars,omitempty"`
	Modules     []Module          `json:"modules,omitempty"`   // detected submodules
	Endpoints   []Endpoint        `json:"endpoints,omitempty"` // detected API endpoints
	Files       []FileInfo        `json:"-"`                   // intermediate; not serialized
	Checksums   map[string]string `json:"checksums,omitempty"` // file path -> sha256
	Complete    bool              `json:"complete"`
	Diagnostics []ScanDiagnostic  `json:"diagnostics,omitempty"`
	Stats       ScanStats         `json:"stats"`
}

// ScanStats reports source inventory and parse reuse for one complete-tree scan.
type ScanStats struct {
	FilesInventoried int           `json:"files_inventoried"`
	Reparsed         int           `json:"reparsed"`
	Unchanged        int           `json:"unchanged"`
	Added            int           `json:"added"`
	Changed          int           `json:"changed"`
	Deleted          int           `json:"deleted"`
	Elapsed          time.Duration `json:"elapsed"`
}

// ScanDiagnostic records a source that could not be inventoried authoritatively.
type ScanDiagnostic struct {
	Path  string `json:"path"`
	Phase string `json:"phase"`
	Error string `json:"error"`
}

// Module represents a detected submodule/subproject in a multi-module build.
type Module struct {
	Name     string   `json:"name"`
	Dir      string   `json:"dir"`      // relative directory
	Kind     string   `json:"kind"`     // "gradle-module", "npm-workspace", etc.
	Packages []string `json:"packages"` // package paths belonging to this module
}

// Parser extracts symbols and structure from source files.
type Parser interface {
	// Languages returns the set of languages this parser supports.
	Languages() []string

	// ParseFile extracts symbols, imports, and metadata from a single file.
	ParseFile(path string, content []byte) (*FileInfo, error)
}

// Endpoint represents a detected API endpoint (REST, GraphQL, gRPC).
type Endpoint struct {
	Method   string `json:"method"`            // HTTP method: GET, POST, etc. or "QUERY"/"MUTATION" for GraphQL, "RPC" for gRPC
	Path     string `json:"path"`              // route path: /api/users/:id
	Handler  string `json:"handler,omitempty"` // handler function/method name
	File     string `json:"file"`              // file where the endpoint is defined
	Line     int    `json:"line"`              // line number
	Protocol string `json:"protocol"`          // "rest", "graphql", "grpc", "websocket"
}
