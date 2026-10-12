package domain

type Capability string

const (
	TextOnly      Capability = "TEXT_ONLY"
	ASTSymbols    Capability = "AST_SYMBOLS"
	StaticCalls   Capability = "STATIC_CALLS"
	TypeRelations Capability = "TYPE_RELATIONS"
)

type RelationKind string

const (
	RelationCalls             RelationKind = "CALLS"
	RelationCallsMethod       RelationKind = "CALLS_METHOD"
	RelationInterfaceDispatch RelationKind = "INTERFACE_DISPATCH"
)

// SourceRelation links indexed artifacts. Evidence is located in repository-relative source.
type SourceRelation struct {
	Kind     RelationKind
	TargetID string
	Location string
	Excerpt  string
}

// FileAnalysis describes actual processing, including degraded text fallback.
type FileAnalysis struct {
	File         string       `json:"file"`
	Language     string       `json:"language"`
	Adapter      string       `json:"adapter"`
	Capabilities []Capability `json:"capabilities"`
	Fallback     bool         `json:"fallback,omitempty"`
}
