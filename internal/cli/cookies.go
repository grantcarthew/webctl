package cli

import (
	"encoding/json"
	"os"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var cookiesCmd = &cobra.Command{
	Use:   "cookies",
	Short: "Manage browser cookies",
	Long: `List, set, or delete cookies in the active browser session.

Without subcommands, lists all cookies for the current page with full
CDP attributes (domain, path, expiry, secure, httpOnly, sameSite, etc.).

Subcommands:
  cookies             List all cookies (default)
  cookies set         Set a cookie with optional attributes
  cookies delete      Delete a cookie by name

List all cookies:
  cookies

  Response includes all CDP cookie attributes:
  {
    "ok": true,
    "cookies": [
      {
        "name": "session",
        "value": "abc123",
        "domain": "example.com",
        "path": "/",
        "expires": 1735084800,
        "httpOnly": true,
        "secure": true,
        "sameSite": "Lax"
      }
    ],
    "count": 1
  }

Common patterns:
  # Inspect authentication state
  navigate example.com --wait
  cookies                               # Check session cookies

  # Clear and set test cookie
  cookies delete test_flag
  cookies set test_flag enabled --max-age 3600
  reload --wait
  cookies                               # Verify cookie is set

  # Debug login issues
  navigate myapp.com/login --wait
  cookies                               # Check pre-login cookies
  type "#email" "user@example.com"
  type "#password" "secret" --key Enter
  ready
  cookies                               # Check post-login session cookie

Error cases:
  - "daemon not running" - start daemon first with: webctl start
  - "no active session" - no browser page open`,
	RunE: runCookiesList,
}

var cookiesSetCmd = &cobra.Command{
	Use:   "set <name> <value>",
	Short: "Set a cookie",
	Long: `Sets a cookie with the specified name and value.

Without flags, creates a session cookie for the current page's domain.
Use flags to control cookie attributes for persistent or secure cookies.

Flags:
  --domain      Cookie domain (defaults to current page domain)
  --path        Cookie path (defaults to "/")
  --secure      Require HTTPS connection
  --httponly    Prevent JavaScript access (document.cookie)
  --max-age     Expiry in seconds from now (0 = session cookie)
  --samesite    SameSite policy: Strict, Lax, or None

Session cookie (expires when browser closes):
  cookies set session abc123

Persistent cookie (expires in 1 hour):
  cookies set remember_me yes --max-age 3600

Persistent cookie (expires in 24 hours):
  cookies set auth_token xyz789 --max-age 86400

Secure cookie (HTTPS only, no JS access):
  cookies set session abc123 --secure --httponly

Cookie with specific domain:
  cookies set tracking id123 --domain .example.com

Cookie with SameSite policy:
  cookies set csrf_token xyz --samesite Strict
  cookies set analytics id --samesite None --secure

Full example with all attributes:
  cookies set auth_token abc123 \
    --domain example.com \
    --path /api \
    --secure \
    --httponly \
    --max-age 86400 \
    --samesite Strict

Response:
  {"ok": true}

SameSite values:
  Strict  Cookie only sent in first-party context
  Lax     Cookie sent with top-level navigations (default in browsers)
  None    Cookie sent in all contexts (requires --secure)`,
	Args: cobra.ExactArgs(2),
	RunE: runCookiesSet,
}

var cookiesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a cookie",
	Long: `Deletes a cookie by name.

If only one cookie matches the name, it is deleted immediately.
If multiple cookies match (same name, different domains), you must
specify --domain to disambiguate.

Deleting a non-existent cookie returns success (idempotent).

Flags:
  --domain      Cookie domain (required if multiple cookies match)

Delete a cookie (unambiguous):
  cookies delete session

Delete a cookie (with domain):
  cookies delete session --domain api.example.com

Response (success):
  {"ok": true}

Response (ambiguous - multiple matches):
  {
    "ok": false,
    "error": "multiple cookies named 'session' found",
    "matches": [
      {"name": "session", "domain": "example.com"},
      {"name": "session", "domain": "api.example.com"}
    ]
  }

  Then specify: cookies delete session --domain api.example.com

Common patterns:
  # Clear all auth cookies
  cookies delete session
  cookies delete auth_token
  cookies delete remember_me

  # Reset to logged-out state
  cookies delete session
  reload --wait`,
	Args: cobra.ExactArgs(1),
	RunE: runCookiesDelete,
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

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":      true,
			"cookies": data.Cookies,
			"count":   data.Count,
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use text formatter
	return format.Cookies(os.Stdout, data.Cookies, format.NewOutputOptions(JSONOutput, NoColor))
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

	// JSON mode: output JSON
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{"ok": true})
	}

	// Text mode: just output OK
	return outputSuccess(nil)
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
				// JSON mode: return error with matches
				if JSONOutput {
					result := map[string]any{
						"ok":      false,
						"error":   resp.Error,
						"matches": data.Matches,
					}
					outputJSON(os.Stdout, result)
				}
				return outputError(resp.Error)
			}
		}
		return outputError(resp.Error)
	}

	// JSON mode: output JSON
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{"ok": true})
	}

	// Text mode: just output OK
	return outputSuccess(nil)
}
