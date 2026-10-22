package semantic

import (
	"errors"
	"fmt"
	"strings"
)

var ErrRuntimeMemory = errors.New("local embedding runtime has insufficient memory")
var ErrInputLimit = errors.New("semantic embedding input exceeds memory limits")
var ErrRuntimeWorker = errors.New("local embedding worker stopped unexpectedly")

func runtimeFailure(message string) error {
	lower := strings.ToLower(message)
	// Python tracebacks end with the exception that explains the failure. Keep
	// that line rather than the first 800 characters of library stack frames.
	detail := runtimeErrorDetail(message)
	for _, marker := range []string{"memoryerror", "out of memory", "cannot allocate memory", "paging file is too small", "not enough memory", "defaultcpuallocator", "winerror 1455"} {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%w: %s", ErrRuntimeMemory, detail)
		}
	}
	if strings.HasPrefix(detail, "worker exited:") {
		return fmt.Errorf("%w: %s", ErrRuntimeWorker, detail)
	}
	return fmt.Errorf("offline embedding runtime failed: %s", detail)
}

func runtimeErrorDetail(message string) string {
	lines := strings.Split(strings.TrimSpace(message), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && line != "..." {
			return bounded(line, 800)
		}
	}
	return "embedding worker exited without an error message"
}
