package export

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/criteo/klt/src/models"
)

// WriteNDJSON writes the raw JSON of each entry to an NDJSON file (one record
// per line) and returns the path of the created file.
func WriteNDJSON(entries []models.LogEntry) (string, error) {
	path := fmt.Sprintf("klt-export-%s.ndjson", time.Now().Format("20060102-150405"))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	for _, e := range entries {
		if _, err := fmt.Fprintln(f, e.RawJSON); err != nil {
			return path, err
		}
	}
	return path, nil
}

// OpenURL opens url in the default system browser.
func OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
