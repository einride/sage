package make

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/magefile/mage/mage"
)

func PrintMakeTargets() error {
	var b bytes.Buffer
	out := mage.Invoke(mage.Invocation{
		Stdout: &b,
		List:   true,
		Stderr: os.Stderr,
	})
	if out != 0 {
		return fmt.Errorf("mage exited with status code %d", out)
	}

	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	if len(lines) > 0 {
		// Remove "Targets: " lines
		if strings.HasPrefix(lines[0], "Targets:") {
			lines = lines[1:]
		}
		// If a default is set remove the last line informing the
		// default target
		if strings.Contains(lines[len(lines)-1], "* default") {
			lines = lines[:len(lines)-1]
		}
	}

	var targets []string
	for _, l := range lines {
		parts := strings.Fields(l)
		if len(parts) > 0 {
			// Remove spaces and default mark (*)
			targets = append(targets, strings.TrimRight(strings.TrimSpace(parts[0]), "*"))
		}
	}
	fmt.Println(strings.Join(targets, " "))
	return nil
}
