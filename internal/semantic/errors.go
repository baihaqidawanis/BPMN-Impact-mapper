package semantic

import (
	"errors"
	"fmt"
	"strings"
)

var ErrRuntimeMemory = errors.New("local embedding runtime has insufficient memory")
var ErrInputLimit = errors.New("semantic embedding input exceeds memory limits")

func runtimeFailure(message string) error {
	lower := strings.ToLower(message)
	for _, marker := range []string{"memoryerror", "out of memory", "cannot allocate memory", "paging file is too small", "not enough memory", "defaultcpuallocator", "winerror 1455"} {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("%w: %s", ErrRuntimeMemory, bounded(message, 800))
		}
	}
	return fmt.Errorf("offline embedding runtime failed: %s", bounded(message, 800))
}
