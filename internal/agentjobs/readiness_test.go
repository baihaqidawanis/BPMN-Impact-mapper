package agentjobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type readinessFixture string

func (e readinessFixture) Error() string         { return "private diagnostic" }
func (e readinessFixture) ReadinessCode() string { return string(e) }

func TestModelCheckGuidanceIsSafeAndSpecific(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{nil, "No source or inference"}, {ErrBusy, "review is running"},
		{fmt.Errorf("wrapped: %w", context.DeadlineExceeded), "timed out"},
		{readinessFixture("MODEL_NOT_INSTALLED"), "folder alone"},
		{readinessFixture("THINKING_UNSUPPORTED"), "Thinking off"},
		{readinessFixture("GENERATION_UNSUPPORTED"), "embedding-only"},
		{readinessFixture("REMOTE_MODEL"), "cloud inference is disabled"},
		{readinessFixture("MODEL_IDENTITY_CHANGED"), "new AI session"},
		{readinessFixture("MODEL_METADATA_INVALID"), "metadata is missing"},
		{errors.New("private diagnostic token=secret"), "does not establish a VPN failure"},
		{readinessFixture("UNKNOWN"), "does not establish a VPN failure"},
	} {
		message := ModelCheckMessage(tc.err)
		if !strings.Contains(message, tc.want) || strings.Contains(message, "private diagnostic") || strings.Contains(message, "secret") {
			t.Fatal(message)
		}
	}
}
