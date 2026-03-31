package export

import "github.com/atotto/clipboard"

// CopyText copies s to the system clipboard.
func CopyText(s string) error {
	return clipboard.WriteAll(s)
}
