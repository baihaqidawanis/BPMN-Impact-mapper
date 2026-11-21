package llm

// Readiness errors expose a stable category, never a raw server response,
// endpoint, account identifier or repository path.
type readinessError struct {
	code, message string
}

func (e readinessError) Error() string         { return e.message }
func (e readinessError) ReadinessCode() string { return e.code }

func readinessFailure(code, message string) error {
	return readinessError{code: code, message: message}
}
