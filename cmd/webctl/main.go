package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/grantcarthew/webctl/internal/cli"
)

// formatCobraError converts verbose Cobra errors to user-friendly messages.
func formatCobraError(err error) string {
	msg := err.Error()

	// Mutual exclusivity: "if any flags in the group [head tail range] are set none of the others can be; [head tail] were all set"
	if strings.Contains(msg, "none of the others can be") {
		re := regexp.MustCompile(`\[([^\]]+)\] were all set`)
		if matches := re.FindStringSubmatch(msg); len(matches) > 1 {
			flags := strings.Split(matches[1], " ")
			for i := range flags {
				flags[i] = "--" + flags[i]
			}
			return fmt.Sprintf("%s cannot be used together", strings.Join(flags, " and "))
		}
	}

	return msg
}

func main() {
	if err := cli.Execute(); err != nil {
		// Print error if not already printed by command handler
		if !cli.IsPrintedError(err) {
			msg := formatCobraError(err)
			if cli.JSONOutput {
				resp := map[string]any{
					"ok":    false,
					"error": msg,
				}
				_ = json.NewEncoder(os.Stderr).Encode(resp)
			} else {
				fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
			}
		}
		os.Exit(1)
	}
}
