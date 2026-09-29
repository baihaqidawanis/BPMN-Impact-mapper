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
	MethodConfiguration AnalysisMethod = "configuration"
)

func (method AnalysisMethod) Valid() bool {
	switch method {
	case MethodHybrid, MethodBM25, MethodConfiguration:
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
	Elements map[string]*Element
	Flows    map[string]*Flow
	Incoming map[string][]string
	Outgoing map[string][]string
	Warnings []string
}

type Symbol struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Receiver  string `json:"receiver,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type Artifact struct {
	ID       string
	File     string
	Module   string
	Layer    string
	Language string
	Symbol   Symbol
	Content  string
	Tokens   []string
}

type Evidence struct {
	Type     string `json:"type"`
	Location string `json:"location"`
	Excerpt  string `json:"excerpt"`
}

type ScoreBreakdown struct {
	TraceID    float64 `json:"trace_id"`
	Symbol     float64 `json:"symbol"`
	Path       float64 `json:"path"`
	Lexical    float64 `json:"lexical"`
	Structured float64 `json:"structured"`
	BM25       float64 `json:"bm25"`
	Total      float64 `json:"total"`
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

type Change struct {
	ChangeType       ChangeType `json:"change_type"`
	OldElement       *Element   `json:"old_element"`
	NewElement       *Element   `json:"new_element"`
	Impacts          []Impact   `json:"impacts"`
	UnresolvedImpact []string   `json:"unresolved_impacts"`
	Reason           string     `json:"reason,omitempty"`
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
	AnalysisID       string         `json:"analysis_id"`
	InputFingerprint string         `json:"input_fingerprint"`
	Method           AnalysisMethod `json:"method"`
	Repository       Repository     `json:"repository"`
	Changes          []Change       `json:"changes"`
	Warnings         []Warning      `json:"warnings"`
}
