package graph

// NodeKind represents the type of Terraform block
type NodeKind string

const (
	KindResource  NodeKind = "resource"
	KindData      NodeKind = "data"
	KindModule    NodeKind = "module"
	KindProvider  NodeKind = "provider"
	KindVariable  NodeKind = "variable"
	KindLocal     NodeKind = "local"
	KindOutput    NodeKind = "output"
	KindTerraform NodeKind = "terraform"
)

// SourceSpan identifies where a block lives in the source files
type SourceSpan struct {
	File      string `json:"file"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	StartCol  int    `json:"startCol,omitempty"`
	EndCol    int    `json:"endCol,omitempty"`
}

// Attribute represents a single attribute value in a block
type Attribute struct {
	Name       string      `json:"name"`
	Value      interface{} `json:"value,omitempty"`
	Expression string      `json:"expression,omitempty"`
	IsComputed bool        `json:"isComputed,omitempty"`
	Type       string      `json:"type,omitempty"`
	References []string    `json:"references,omitempty"`
}

// NestedBlock represents a nested block within a resource/data block
type NestedBlock struct {
	Type       string      `json:"type"`
	Labels     []string    `json:"labels,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
	RawHCL     string      `json:"rawHCL,omitempty"`
}

// GraphNode represents a single node in the Terraform graph
type GraphNode struct {
	ID           string        `json:"id"`
	Kind         NodeKind      `json:"kind"`
	ResourceType string        `json:"resourceType,omitempty"`
	Name         string        `json:"name"`
	Address      string        `json:"address"`
	Provider     string        `json:"provider,omitempty"`
	Source       SourceSpan    `json:"source"`
	Attributes   []Attribute   `json:"attributes,omitempty"`
	NestedBlocks []NestedBlock `json:"nestedBlocks,omitempty"`
	RawHCL       string        `json:"rawHCL,omitempty"`
	DependsOn    []string      `json:"dependsOn,omitempty"`
	// For variables
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
	VarType     string      `json:"varType,omitempty"`
	// For modules
	ModuleSource  string `json:"moduleSource,omitempty"`
	ModuleVersion string `json:"moduleVersion,omitempty"`
}

// EdgeKind represents the type of dependency between nodes
type EdgeKind string

const (
	EdgeReference  EdgeKind = "reference"
	EdgeDependsOn  EdgeKind = "depends_on"
	EdgeProvider   EdgeKind = "provider"
	EdgeModule     EdgeKind = "module"
	EdgeContains   EdgeKind = "contains"
)

// GraphEdge represents a connection between two nodes
type GraphEdge struct {
	ID     string   `json:"id"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Kind   EdgeKind `json:"kind"`
	Label  string   `json:"label,omitempty"`
}

// Diagnostic represents a validation issue
type Diagnostic struct {
	Severity string     `json:"severity"` // error, warning
	Summary  string     `json:"summary"`
	Detail   string     `json:"detail,omitempty"`
	Range    *SourceSpan `json:"range,omitempty"`
	NodeID   string     `json:"nodeId,omitempty"`
}

// PlanAction represents what Terraform plans to do with a resource
type PlanAction string

const (
	PlanCreate    PlanAction = "create"
	PlanUpdate    PlanAction = "update"
	PlanDelete    PlanAction = "delete"
	PlanReplace   PlanAction = "replace"
	PlanNoOp      PlanAction = "no-op"
	PlanRead      PlanAction = "read"
)

// PlanChange represents a planned change for a resource
type PlanChange struct {
	Address string                 `json:"address"`
	Action  PlanAction             `json:"action"`
	Before  map[string]interface{} `json:"before,omitempty"`
	After   map[string]interface{} `json:"after,omitempty"`
	AfterUnknown map[string]interface{} `json:"afterUnknown,omitempty"`
}

// WorkspaceGraph is the full graph response
type WorkspaceGraph struct {
	Nodes       []GraphNode  `json:"nodes"`
	Edges       []GraphEdge  `json:"edges"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Files       []string     `json:"files"`
}
