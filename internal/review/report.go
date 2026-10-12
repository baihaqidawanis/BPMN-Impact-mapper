package review

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func JSON(value Reconciliation) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Markdown(value Reconciliation) []byte {
	var output bytes.Buffer
	fmt.Fprintf(&output, "# Ground Truth Reconciliation\n\n- Reviewer A: `%s`\n- Reviewer B: `%s`\n- Agreed cases: %d\n- Disagreements: %d\n\n", value.ReviewerA, value.ReviewerB, len(value.Agreed.Cases), len(value.Disagreements))
	if value.ReviewPack != nil {
		fmt.Fprintf(&output, "- Review pack: `%s`\n- Input fingerprint: `%s`\n- Case digest: `%s`\n\n", value.ReviewPack.PackID, value.ReviewPack.InputFingerprint, value.ReviewPack.CaseDigest)
	}
	if len(value.Disagreements) > 0 {
		output.WriteString("## Requires adjudication\n\n")
		for _, item := range value.Disagreements {
			fmt.Fprintf(&output, "- `%s` / `%s`: %s\n", item.ChangeType, item.ElementID, item.Reason)
		}
	}
	return output.Bytes()
}
