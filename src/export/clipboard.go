package export

import (
	"bytes"
	"encoding/json"

	"github.com/atotto/clipboard"
)

// CopyText copies s to the system clipboard.
func CopyText(s string) error {
	return clipboard.WriteAll(s)
}

// CopyJSON pretty-prints the JSON string s and copies it to the system clipboard.
func CopyJSON(s string) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(s), "", "  "); err != nil {
		return clipboard.WriteAll(s)
	}
	return clipboard.WriteAll(buf.String())
}
