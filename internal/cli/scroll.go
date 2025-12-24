package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var scrollCmd = &cobra.Command{
	Use:   "scroll <selector> | --to x,y | --by x,y",
	Short: "Scroll to element or position",
	Long: `Scrolls to an element, absolute position, or by an offset.

Three scroll modes:
  1. Element mode: scroll an element into the center of the viewport
  2. Absolute mode: scroll to an exact position on the page
  3. Relative mode: scroll by an offset from current position

Coordinates are specified as x,y where:
  x = horizontal position (0 = left edge)
  y = vertical position (0 = top edge)

Element scroll examples:
  scroll "#footer"                    # Scroll footer into view
  scroll ".next-section"              # Scroll to next section
  scroll "article h2"                 # Scroll to first h2 in article
  scroll "[data-testid=results]"      # Scroll to test ID element

Given this HTML:
  <div id="content">
    <section id="intro">...</section>
    <section id="features">...</section>
    <section id="pricing">...</section>
    <footer id="contact">...</footer>
  </div>

Use: scroll "#pricing"  (scrolls pricing section to center of viewport)

Absolute position examples (--to):
  scroll --to 0,0                     # Scroll to top-left (top of page)
  scroll --to 0,500                   # Scroll to 500px from top
  scroll --to 0,1000                  # Scroll to 1000px from top
  scroll --to 100,200                 # Scroll to x=100, y=200

Relative offset examples (--by):
  scroll --by 0,100                   # Scroll down 100px
  scroll --by 0,-100                  # Scroll up 100px
  scroll --by 0,500                   # Scroll down 500px (half page)
  scroll --by 200,0                   # Scroll right 200px
  scroll --by -200,0                  # Scroll left 200px
  scroll --by 100,100                 # Scroll diagonally

Common patterns:
  scroll --to 0,0                     # Return to top of page
  scroll "#main-content"              # Skip to main content
  scroll --by 0,window.innerHeight    # Page down (use eval for dynamic)

Error cases:
  - "element not found" - selector doesn't match any element
  - "invalid --to coordinates" - coordinates not in x,y format
  - "provide a selector, --to x,y, or --by x,y" - no mode specified`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScroll,
}

func init() {
	scrollCmd.Flags().String("to", "", "Scroll to absolute position (x,y)")
	scrollCmd.Flags().String("by", "", "Scroll by offset (x,y)")
	rootCmd.AddCommand(scrollCmd)
}

func runScroll(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Read flags from command
	toCoords, _ := cmd.Flags().GetString("to")
	byCoords, _ := cmd.Flags().GetString("by")

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	var params ipc.ScrollParams

	// Determine scroll mode
	if toCoords != "" {
		x, y, err := parseCoords(toCoords)
		if err != nil {
			return outputError(fmt.Sprintf("invalid --to coordinates: %v", err))
		}
		params.Mode = "to"
		params.ToX = x
		params.ToY = y
	} else if byCoords != "" {
		x, y, err := parseCoords(byCoords)
		if err != nil {
			return outputError(fmt.Sprintf("invalid --by coordinates: %v", err))
		}
		params.Mode = "by"
		params.ByX = x
		params.ByY = y
	} else if len(args) == 1 {
		params.Mode = "element"
		params.Selector = args[0]
	} else {
		return outputError("provide a selector, --to x,y, or --by x,y")
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "scroll",
		Params: paramsJSON,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	result := map[string]any{
		"ok": true,
	}
	return outputJSON(os.Stdout, result)
}

// parseCoords parses a "x,y" string into integers.
func parseCoords(s string) (int, int, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected x,y format")
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid x coordinate: %v", err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid y coordinate: %v", err)
	}
	return x, y, nil
}
