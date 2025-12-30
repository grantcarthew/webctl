package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grantcarthew/webctl/internal/cli/format"
	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

var cookiesCmd = &cobra.Command{
	Use:   "cookies",
	Short: "Extract cookies from current page (default: save to temp)",
	Long: `Extracts cookies from the current page with flexible output modes.

Default behavior (no subcommand):
  Saves cookies to /tmp/webctl-cookies/ with auto-generated filename
  Returns JSON with file path

Subcommands:
  show              Output cookies to stdout
  save <path>       Save cookies to custom path
  set <name> <value>  Set a cookie (mutation)
  delete <name>     Delete a cookie (mutation)

Universal flags (work with default/show/save modes):
  --find, -f        Search for text within cookie names and values
  --raw             Skip formatting (return raw JSON)
  --json            Output in JSON format (global flag)

Cookies-specific filter flags (observation only):
  --domain DOMAIN   Filter by cookie domain
  --name NAME       Filter by exact cookie name

Examples:

Default mode (save to temp):
  cookies                                  # All cookies to temp
  cookies --domain ".github.com"           # Only GitHub cookies to temp
  cookies --find "session"                 # Search and save matches

Show mode (stdout):
  cookies show                             # All cookies to stdout
  cookies show --domain ".github.com"      # Only GitHub cookies
  cookies show --find "session"            # Search and show matches

Save mode (custom path):
  cookies save ./cookies.json              # Save to file
  cookies save ./output/                   # Save to dir (auto-filename)
  cookies save ./auth-cookies.json --find "auth"

Mutation subcommands:
  cookies set session abc123               # Set session cookie
  cookies set auth xyz --secure --httponly # Set secure cookie
  cookies delete session                   # Delete cookie

Response formats:
  Default/Save: {"ok": true, "path": "/tmp/webctl-cookies/25-12-28-143052-cookies.json"}
  Show:         session | abc123 | .example.com | / | Session | Secure, HttpOnly

Error cases:
  - "no matches found for 'text'" - find text not in cookies
  - "daemon not running" - start daemon first with: webctl start`,
	RunE: runCookiesDefault,
}

var cookiesShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Output cookies to stdout",
	Long: `Outputs cookies to stdout for real-time inspection and piping.

Examples:
  cookies show                             # All cookies
  cookies show --domain ".github.com"      # Only GitHub cookies
  cookies show --find "session"            # Search within cookies
  cookies show --name "session_id"         # Exact name match`,
	RunE: runCookiesShow,
}

var cookiesSaveCmd = &cobra.Command{
	Use:   "save <path>",
	Short: "Save cookies to custom path",
	Long: `Saves cookies to a custom file path.

If path is a directory, auto-generates filename.
If path is a file, uses exact path.

Examples:
  cookies save ./cookies.json              # Save to file
  cookies save ./output/                   # Save to dir
  cookies save ./github-cookies.json --domain ".github.com"`,
	Args: cobra.ExactArgs(1),
	RunE: runCookiesSave,
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
	// Universal flags on root command (inherited by default/show/save subcommands)
	cookiesCmd.PersistentFlags().StringP("find", "f", "", "Search for text within cookie names and values")
	cookiesCmd.PersistentFlags().Bool("raw", false, "Skip formatting (return raw JSON)")

	// Cookies-specific filter flags (observation only)
	cookiesCmd.PersistentFlags().String("domain", "", "Filter by cookie domain")
	cookiesCmd.PersistentFlags().String("name", "", "Filter by exact cookie name")

	// Flags for set subcommand
	cookiesSetCmd.Flags().String("domain", "", "Cookie domain (defaults to current page domain)")
	cookiesSetCmd.Flags().String("path", "/", "Cookie path")
	cookiesSetCmd.Flags().Bool("secure", false, "Require HTTPS")
	cookiesSetCmd.Flags().Bool("httponly", false, "HTTP-only (no JavaScript access)")
	cookiesSetCmd.Flags().Int("max-age", 0, "Expiry in seconds from now (0 = session cookie)")
	cookiesSetCmd.Flags().String("samesite", "", "SameSite policy: Strict, Lax, or None")

	// Flags for delete subcommand
	cookiesDeleteCmd.Flags().String("domain", "", "Cookie domain (required if ambiguous)")

	// Add all subcommands
	cookiesCmd.AddCommand(cookiesShowCmd, cookiesSaveCmd, cookiesSetCmd, cookiesDeleteCmd)

	rootCmd.AddCommand(cookiesCmd)
}

// runCookiesDefault handles default behavior: save to temp directory
func runCookiesDefault(cmd *cobra.Command, args []string) error {
	// Validate that no arguments were provided (catches unknown subcommands)
	if len(args) > 0 {
		return outputError(fmt.Sprintf("unknown command %q for \"webctl cookies\"", args[0]))
	}

	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get cookies from daemon
	cookies, err := getCookiesFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Generate filename in temp directory
	outputPath, err := generateCookiesPath()
	if err != nil {
		return outputError(err.Error())
	}

	// Write cookies to file
	if err := writeCookiesToFile(outputPath, cookies); err != nil {
		return outputError(err.Error())
	}

	// Return JSON response
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":   true,
			"path": outputPath,
		})
	}

	return format.FilePath(os.Stdout, outputPath)
}

// runCookiesShow handles show subcommand: output to stdout
func runCookiesShow(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	// Get cookies from daemon
	cookies, err := getCookiesFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// JSON mode: output JSON
	if JSONOutput {
		result := map[string]any{
			"ok":      true,
			"cookies": cookies,
			"count":   len(cookies),
		}
		return outputJSON(os.Stdout, result)
	}

	// Check --raw flag
	raw, _ := cmd.Flags().GetBool("raw")
	if !raw && cmd.Parent() != nil {
		raw, _ = cmd.Parent().PersistentFlags().GetBool("raw")
	}

	if raw {
		// Raw mode: output as JSON
		result := map[string]any{
			"ok":      true,
			"cookies": cookies,
			"count":   len(cookies),
		}
		return outputJSON(os.Stdout, result)
	}

	// Text mode: use text formatter
	return format.Cookies(os.Stdout, cookies, format.NewOutputOptions(JSONOutput, NoColor))
}

// runCookiesSave handles save subcommand: save to custom path
func runCookiesSave(cmd *cobra.Command, args []string) error {
	if !execFactory.IsDaemonRunning() {
		return outputError("daemon not running. Start with: webctl start")
	}

	path := args[0]

	// Get cookies from daemon
	cookies, err := getCookiesFromDaemon(cmd)
	if err != nil {
		return outputError(err.Error())
	}

	// Handle directory vs file path
	fileInfo, err := os.Stat(path)
	if err == nil && fileInfo.IsDir() {
		// Path is a directory - auto-generate filename
		filename := generateCookiesFilename()
		path = filepath.Join(path, filename)
	}

	// Write cookies to file
	if err := writeCookiesToFile(path, cookies); err != nil {
		return outputError(err.Error())
	}

	// Return JSON response
	if JSONOutput {
		return outputJSON(os.Stdout, map[string]any{
			"ok":   true,
			"path": path,
		})
	}

	return format.FilePath(os.Stdout, path)
}

// getCookiesFromDaemon fetches cookies from daemon, applying filters
func getCookiesFromDaemon(cmd *cobra.Command) ([]ipc.Cookie, error) {
	// Try to get flags from command, falling back to parent for persistent flags
	find, _ := cmd.Flags().GetString("find")
	if find == "" && cmd.Parent() != nil {
		find, _ = cmd.Parent().PersistentFlags().GetString("find")
	}

	domain, _ := cmd.Flags().GetString("domain")
	if domain == "" && cmd.Parent() != nil {
		domain, _ = cmd.Parent().PersistentFlags().GetString("domain")
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" && cmd.Parent() != nil {
		name, _ = cmd.Parent().PersistentFlags().GetString("name")
	}

	exec, err := execFactory.NewExecutor()
	if err != nil {
		return nil, err
	}
	defer exec.Close()

	params, err := json.Marshal(ipc.CookiesParams{
		Action: "list",
	})
	if err != nil {
		return nil, err
	}

	// Execute cookies request
	resp, err := exec.Execute(ipc.Request{
		Cmd:    "cookies",
		Params: params,
	})
	if err != nil {
		return nil, err
	}

	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	// Parse cookies data
	var data ipc.CookiesData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}

	cookies := data.Cookies

	// Apply domain filter
	if domain != "" {
		cookies = filterCookiesByDomain(cookies, domain)
	}

	// Apply name filter
	if name != "" {
		cookies = filterCookiesByName(cookies, name)
	}

	// Apply --find filter if specified
	if find != "" {
		cookies = filterCookiesByText(cookies, find)
		if len(cookies) == 0 {
			return nil, fmt.Errorf("no matches found for '%s'", find)
		}
	}

	return cookies, nil
}

// filterCookiesByDomain filters cookies to only include those matching the domain
func filterCookiesByDomain(cookies []ipc.Cookie, domain string) []ipc.Cookie {
	var filtered []ipc.Cookie
	domainLower := strings.ToLower(domain)

	for _, cookie := range cookies {
		cookieDomain := strings.ToLower(cookie.Domain)

		// Exact match
		if cookieDomain == domainLower {
			filtered = append(filtered, cookie)
			continue
		}

		// If filter domain doesn't start with dot, check if cookie domain matches as suffix
		// e.g., "example.com" matches ".example.com", "www.example.com", "api.example.com"
		if !strings.HasPrefix(domainLower, ".") {
			// Cookie domain ".example.com" matches filter "example.com"
			if cookieDomain == "."+domainLower {
				filtered = append(filtered, cookie)
				continue
			}
			// Cookie domain "www.example.com" matches filter "example.com"
			if strings.HasSuffix(cookieDomain, "."+domainLower) {
				filtered = append(filtered, cookie)
				continue
			}
		}
	}

	return filtered
}

// filterCookiesByName filters cookies to only include those with exact name match
func filterCookiesByName(cookies []ipc.Cookie, name string) []ipc.Cookie {
	var filtered []ipc.Cookie

	for _, cookie := range cookies {
		if cookie.Name == name {
			filtered = append(filtered, cookie)
		}
	}

	return filtered
}

// filterCookiesByText filters cookies to only include those containing the search text in name or value
func filterCookiesByText(cookies []ipc.Cookie, searchText string) []ipc.Cookie {
	var matchedCookies []ipc.Cookie
	searchLower := strings.ToLower(searchText)

	for _, cookie := range cookies {
		// Search in name
		if strings.Contains(strings.ToLower(cookie.Name), searchLower) {
			matchedCookies = append(matchedCookies, cookie)
			continue
		}
		// Search in value
		if strings.Contains(strings.ToLower(cookie.Value), searchLower) {
			matchedCookies = append(matchedCookies, cookie)
			continue
		}
	}

	return matchedCookies
}

// writeCookiesToFile writes cookies to a file in JSON format, creating directories if needed
func writeCookiesToFile(path string, cookies []ipc.Cookie) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Marshal cookies to JSON
	data := map[string]any{
		"ok":      true,
		"cookies": cookies,
		"count":   len(cookies),
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %v", err)
	}

	// Write to file
	if err := os.WriteFile(path, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write cookies: %v", err)
	}

	return nil
}

// generateCookiesPath generates a full path in /tmp/webctl-cookies/
// using the pattern: YY-MM-DD-HHMMSS-cookies.json
func generateCookiesPath() (string, error) {
	filename := generateCookiesFilename()
	return filepath.Join("/tmp/webctl-cookies", filename), nil
}

// generateCookiesFilename generates a filename using the pattern:
// YY-MM-DD-HHMMSS-cookies.json
func generateCookiesFilename() string {
	// Generate timestamp: YY-MM-DD-HHMMSS
	now := time.Now()
	timestamp := now.Format("06-01-02-150405")

	// Generate filename with fixed identifier "cookies"
	return fmt.Sprintf("%s-cookies.json", timestamp)
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
