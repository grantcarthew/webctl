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

With a selector: scrolls the element into the center of the viewport.
With --to x,y: scrolls to an absolute position.
With --by x,y: scrolls by the given offset.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScroll,
}

var (
	scrollTo string
	scrollBy string
)

func init() {
	scrollCmd.Flags().StringVar(&scrollTo, "to", "", "Scroll to absolute position (x,y)")
	scrollCmd.Flags().StringVar(&scrollBy, "by", "", "Scroll by offset (x,y)")
	rootCmd.AddCommand(scrollCmd)
}

func runScroll(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	var params ipc.ScrollParams

	// Determine scroll mode
	if scrollTo != "" {
		x, y, err := parseCoords(scrollTo)
		if err != nil {
			return outputError(fmt.Sprintf("invalid --to coordinates: %v", err))
		}
		params.Mode = "to"
		params.ToX = x
		params.ToY = y
	} else if scrollBy != "" {
		x, y, err := parseCoords(scrollBy)
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
