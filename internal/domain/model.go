package domain

type ChangeType string

const (
	AddTask                ChangeType = "ADD_TASK"
	RemoveTask             ChangeType = "REMOVE_TASK"
	ModifyTaskLabel        ChangeType = "MODIFY_TASK_LABEL"
	ModifyActorRole        ChangeType = "MODIFY_ACTOR_ROLE"
	AddGateway             ChangeType = "ADD_GATEWAY"
	RemoveGateway          ChangeType = "REMOVE_GATEWAY"
	ModifyGatewayCondition ChangeType = "MODIFY_GATEWAY_CONDITION"
	ModifyGatewaySemantics ChangeType = "MODIFY_GATEWAY_SEMANTICS"
	ModifySequenceFlow     ChangeType = "MODIFY_SEQUENCE_FLOW"
	AddDataRequirement     ChangeType = "ADD_DATA_REQUIREMENT"
	RemoveDataRequirement  ChangeType = "REMOVE_DATA_REQUIREMENT"
	AddState               ChangeType = "ADD_STATE"
	ModifyTerminalOutcome  ChangeType = "MODIFY_TERMINAL_OUTCOME"
	UnresolvedMatch        ChangeType = "UNRESOLVED_MATCH"
)

type RequiredAction string

const (
	ActionAdd        RequiredAction = "ADD"
	ActionModify     RequiredAction = "MODIFY"
	ActionRemove     RequiredAction = "REMOVE"
	ActionVerify     RequiredAction = "VERIFY"
	ActionNoAction   RequiredAction = "NO_ACTION"
	ActionUnresolved RequiredAction = "UNRESOLVED"
)

type AnalysisMethod string

const (
	MethodHybrid        AnalysisMethod = "hybrid"
	MethodBM25          AnalysisMethod = "bm25"
	MethodSemantic      AnalysisMethod = "semantic"
	MethodFusion        AnalysisMethod = "fusion"
	MethodConfiguration AnalysisMethod = "configuration"
)

func (method AnalysisMethod) Valid() bool {
	switch method {
	case MethodHybrid, MethodBM25, MethodSemantic, MethodFusion, MethodConfiguration:
		return true
	default:
		return false
	}
}

type Element struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	Type             string   `json:"type"`
	Actor            string   `json:"actor,omitempty"`
	State            string   `json:"state,omitempty"`
	Documentation    string   `json:"documentation,omitempty"`
	ProcessID        string   `json:"process_id,omitempty"`
	ParentID         string   `json:"parent_id,omitempty"`
	AttachedToRef    string   `json:"attached_to_ref,omitempty"`
	GatewayDirection string   `json:"gateway_direction,omitempty"`
	DefaultFlow      string   `json:"default_flow,omitempty"`
	EventDefinitions []string `json:"event_definitions,omitempty"`
	DataRequirements []string `json:"data_requirements,omitempty"`
	Incoming         []string `json:"incoming,omitempty"`
	Outgoing         []string `json:"outgoing,omitempty"`
}

type Flow struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	SourceRef string `json:"source_ref"`
	TargetRef string `json:"target_ref"`
	Condition string `json:"condition,omitempty"`
}

type Graph struct {
	ProcessNames map[string]string
	Elements     map[string]*Element
	Flows        map[string]*Flow
	Incoming     map[string][]string
	Outgoing     map[string][]string
	Warnings     []string
}

type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Receiver  string `json:"receiver,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type Artifact struct {
	ID        string
	File      string
	Module    string
	Layer     string
	Language  string
	Symbol    Symbol
	Content   string
	Tokens    []string
	Relations []SourceRelation
}

type Evidence struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	Excerpt  string `json:"excerpt"`
}

type ScoreBreakdown struct {
	EvidenceScore *float64 `json:"evidence_score,omitempty"`
	Dependency    float64  `json:"dependency,omitempty"`
	Semantic      float64  `json:"semantic,omitempty"`
	RRF           float64  `json:"rrf,omitempty"`
	TraceID       float64  `json:"trace_id"`
	Symbol        float64  `json:"symbol"`
	Path          float64  `json:"path"`
	Lexical       float64  `json:"lexical"`
	Structured    float64  `json:"structured"`
	BM25          float64  `json:"bm25"`
	Total         float64  `json:"total"`
}

type Impact struct {
	Module         string         `json:"module"`
	Layer          string         `json:"layer"`
	File           string         `json:"file"`
	Symbol         string         `json:"symbol,omitempty"`
	RequiredAction RequiredAction `json:"required_action"`
	Evidence       []Evidence     `json:"evidence"`
	ScoreBreakdown ScoreBreakdown `json:"score_breakdown"`
	Confidence     float64        `json:"confidence"`
}

type ChangePattern string

const (
	PatternInsertTask           ChangePattern = "INSERT_TASK"
	PatternRemoveTaskWithRewire ChangePattern = "REMOVE_TASK_WITH_REWIRE"
	PatternAddGatewayBranch     ChangePattern = "ADD_GATEWAY_BRANCH"
	PatternMoveTask             ChangePattern = "MOVE_TASK"
	PatternRemoveGatewayBranch  ChangePattern = "REMOVE_GATEWAY_BRANCH"
	PatternModifyGatewayBranch  ChangePattern = "MODIFY_GATEWAY_BRANCH"
)

type ElementContext struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

type ChangeContext struct {
	BranchChanges   []BranchChange   `json:"branch_changes,omitempty"`
	ConditionDelta  []string         `json:"condition_delta,omitempty"`
	BaselineAnchors []BaselineAnchor `json:"baseline_anchors,omitempty"`
	ProcessName     string           `json:"process_name,omitempty"`
	Pattern         ChangePattern    `json:"pattern"`
	Predecessors    []ElementContext `json:"predecessors,omitempty"`
	Successors      []ElementContext `json:"successors,omitempty"`
	BranchTerms     []string         `json:"branch_terms,omitempty"`
	RelatedFlowIDs  []string         `json:"related_flow_ids,omitempty"`
}

type BaselineAnchor struct {
	ElementContext
	Direction string   `json:"direction"`
	Hops      int      `json:"hops"`
	FlowPath  []string `json:"flow_path"`
}

type Change struct {
	RelatedFiles     []RelatedFile  `json:"related_files,omitempty"`
	ChangeType       ChangeType     `json:"change_type"`
	OldElement       *Element       `json:"old_element"`
	NewElement       *Element       `json:"new_element"`
	PartOf           string         `json:"part_of,omitempty"`
	Impacts          []Impact       `json:"impacts"`
	UnresolvedImpact []string       `json:"unresolved_impacts"`
	Reason           string         `json:"reason,omitempty"`
	Context          *ChangeContext `json:"context,omitempty"`
}

// RelatedFile is an unranked source association, not a proven required change.
type RelatedFile struct {
	RankedPosition int        `json:"ranked_position,omitempty"`
	File           string     `json:"file"`
	Module         string     `json:"module"`
	Layer          string     `json:"layer"`
	SeedFile       string     `json:"seed_file"`
	SeedFunction   string     `json:"seed_function"`
	Hops           int        `json:"hops"`
	Evidence       []Evidence `json:"evidence"`
}

type BranchChange struct {
	FlowID       string `json:"flow_id"`
	OldCondition string `json:"old_condition,omitempty"`
	NewCondition string `json:"new_condition,omitempty"`
	OldLabel     string `json:"old_label,omitempty"`
	NewLabel     string `json:"new_label,omitempty"`
	OldTarget    string `json:"old_target,omitempty"`
	NewTarget    string `json:"new_target,omitempty"`
}

type Repository struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Report struct {
	SourceAnalysis   []FileAnalysis    `json:"source_analysis,omitempty"`
	AnalysisID       string            `json:"analysis_id"`
	InputFingerprint string            `json:"input_fingerprint"`
	Method           AnalysisMethod    `json:"method"`
	MethodMetadata   map[string]string `json:"method_metadata,omitempty"`
	Repository       Repository        `json:"repository"`
	Changes          []Change          `json:"changes"`
	Warnings         []Warning         `json:"warnings"`
}

type ReportSummary struct {
	AnalysisID       string         `json:"analysis_id"`
	InputFingerprint string         `json:"input_fingerprint"`
	Method           AnalysisMethod `json:"method"`
	Repository       Repository     `json:"repository"`
	Changes          int            `json:"changes"`
	Candidates       int            `json:"candidates"`
	Unresolved       int            `json:"unresolved"`
	Warnings         int            `json:"warnings"`
}
