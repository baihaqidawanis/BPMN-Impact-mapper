package application

const reviewedState = "REVIEWED"

// RegisterRequest handles BPMN trace TASK-CREATE and records an audit action.
func RegisterRequest() string {
	return "TASK-CREATE"
}

// ValidateRequest handles the TASK-VALIDATE service activity.
func ValidateRequest() string {
	return "TASK-VALIDATE"
}
