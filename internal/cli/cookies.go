package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var cookiesCmd = &cobra.Command{
	Use:   "cookies",
	Short: "Manage browser cookies",
	Long:  "List, set, or delete cookies in the active browser session.",
	RunE:  runCookiesList,
}

var cookiesSetCmd = &cobra.Command{
	Use:   "set <name> <value>",
	Short: "Set a cookie",
	Args:  cobra.ExactArgs(2),
	RunE:  runCookiesSet,
}

var cookiesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a cookie",
	Args:  cobra.ExactArgs(1),
	RunE:  runCookiesDelete,
}

func init() {
	// Flags for set subcommand
	cookiesSetCmd.Flags().String("domain", "", "Cookie domain (defaults to current page domain)")
	cookiesSetCmd.Flags().String("path", "/", "Cookie path")
	cookiesSetCmd.Flags().Bool("secure", false, "Require HTTPS")
	cookiesSetCmd.Flags().Bool("httponly", false, "HTTP-only (no JavaScript access)")
	cookiesSetCmd.Flags().Int("max-age", 0, "Expiry in seconds from now (0 = session cookie)")
	cookiesSetCmd.Flags().String("samesite", "", "SameSite policy: Strict, Lax, or None")

	// Flags for delete subcommand
	cookiesDeleteCmd.Flags().String("domain", "", "Cookie domain (required if ambiguous)")

	cookiesCmd.AddCommand(cookiesSetCmd)
	cookiesCmd.AddCommand(cookiesDeleteCmd)
	rootCmd.AddCommand(cookiesCmd)
}

func runCookiesList(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.CookiesParams{
		Action: "list",
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "cookies",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	var data ipc.CookiesData
	if len(resp.Data) > 0 {
		if err := json.Unmarshal(resp.Data, &data); err != nil {
			return outputError(err.Error())
		}
	}

	result := map[string]any{
		"ok":      true,
		"cookies": data.Cookies,
		"count":   data.Count,
	}

	return outputJSON(os.Stdout, result)
}

func runCookiesSet(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	// Read flags
	domain, _ := cmd.Flags().GetString("domain")
	path, _ := cmd.Flags().GetString("path")
	secure, _ := cmd.Flags().GetBool("secure")
	httponly, _ := cmd.Flags().GetBool("httponly")
	maxAge, _ := cmd.Flags().GetInt("max-age")
	sameSite, _ := cmd.Flags().GetString("samesite")

	params, err := json.Marshal(ipc.CookiesParams{
		Action:   "set",
		Name:     args[0],
		Value:    args[1],
		Domain:   domain,
		Path:     path,
		Secure:   secure,
		HTTPOnly: httponly,
		MaxAge:   maxAge,
		SameSite: sameSite,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "cookies",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		return outputError(resp.Error)
	}

	return outputJSON(os.Stdout, map[string]any{"ok": true})
}

func runCookiesDelete(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return outputError(err.Error())
	}
	defer exec.Close()

	domain, _ := cmd.Flags().GetString("domain")

	params, err := json.Marshal(ipc.CookiesParams{
		Action: "delete",
		Name:   args[0],
		Domain: domain,
	})
	if err != nil {
		return outputError(err.Error())
	}

	resp, err := exec.Execute(ipc.Request{
		Cmd:    "cookies",
		Params: params,
	})
	if err != nil {
		return outputError(err.Error())
	}

	if !resp.OK {
		// Check if this is an ambiguous delete error with matches
		var data ipc.CookiesData
		if len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, &data); err == nil && len(data.Matches) > 0 {
				// Return error with matches
				result := map[string]any{
					"ok":      false,
					"error":   resp.Error,
					"matches": data.Matches,
				}
				outputJSON(os.Stdout, result)
				return outputError(resp.Error)
			}
		}
		return outputError(resp.Error)
	}

	return outputJSON(os.Stdout, map[string]any{"ok": true})
}
