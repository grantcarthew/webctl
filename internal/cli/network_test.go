package cli

import (
	"strings"
	"testing"

	"github.com/grantcarthew/webctl/internal/ipc"
	"github.com/spf13/cobra"
)

func TestResolveMaxBodySize(t *testing.T) {
	// Mirror the real tree: --max-body-size is a persistent flag on `network`,
	// inherited by the `save` subcommand.
	newTree := func(got *int) *cobra.Command {
		parent := &cobra.Command{Use: "network", Run: func(c *cobra.Command, _ []string) {
			*got = resolveMaxBodySize(c)
		}}
		parent.PersistentFlags().Int("max-body-size", ipc.DefaultMaxBodySize, "")
		child := &cobra.Command{Use: "save", Run: func(c *cobra.Command, _ []string) {
			*got = resolveMaxBodySize(c)
		}}
		parent.AddCommand(child)
		return parent
	}

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"unset uses default", []string{}, ipc.DefaultMaxBodySize},
		{"explicit value", []string{"--max-body-size", "2048"}, 2048},
		{"explicit zero is honoured", []string{"--max-body-size", "0"}, 0},
		{"save unset uses default", []string{"save"}, ipc.DefaultMaxBodySize},
		{"save inherits explicit value", []string{"save", "--max-body-size", "4096"}, 4096},
		{"save inherits explicit zero", []string{"save", "--max-body-size", "0"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int
			root := newTree(&got)
			root.SetArgs(tt.args)
			if err := root.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveMaxBodySize = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		maxBytes  int
		want      string
		truncated bool
	}{
		{"under budget", "hello", 10, "hello", false},
		{"at budget", "hello", 5, "hello", false},
		{"over budget ascii", "hello world", 5, "hello", true},
		{"zero budget", "hello", 0, "", true},
		{"negative budget clamps to zero", "hello", -1, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, truncated := truncateBody(tt.body, tt.maxBytes)
			if got != tt.want || truncated != tt.truncated {
				t.Errorf("truncateBody(%q, %d) = (%q, %v), want (%q, %v)",
					tt.body, tt.maxBytes, got, truncated, tt.want, tt.truncated)
			}
		})
	}
}

func TestTruncateBody_RuneBoundary(t *testing.T) {
	// "é" is two bytes (0xC3 0xA9). A budget that lands mid-rune must back up to
	// the boundary, never splitting the multibyte rune.
	body := "aéb" // bytes: 'a'(1) 'é'(2) 'b'(1) = 4 bytes
	got, truncated := truncateBody(body, 2)
	if !truncated {
		t.Fatal("expected truncation")
	}
	if got != "a" {
		t.Errorf("truncateBody = %q, want %q (must not split the multibyte rune)", got, "a")
	}
	if !isValidUTF8(got) {
		t.Errorf("result %q is not valid UTF-8", got)
	}
}

func isValidUTF8(s string) bool {
	return strings.ToValidUTF8(s, "�") == s
}

func TestApplyBodyTruncation_RequestAndResponse(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{
			RequestBody: "0123456789",
			Body:        "abcdefghij",
		},
	}
	applyBodyTruncation(entries, 4)

	if entries[0].RequestBody != "0123" {
		t.Errorf("RequestBody = %q, want %q", entries[0].RequestBody, "0123")
	}
	if !entries[0].RequestBodyTruncated {
		t.Error("RequestBodyTruncated should be true")
	}
	if entries[0].Body != "abcd" {
		t.Errorf("Body = %q, want %q", entries[0].Body, "abcd")
	}
	if !entries[0].BodyTruncated {
		t.Error("BodyTruncated should be true")
	}
}

func TestApplyBodyTruncation_NoFlagWhenWithinBudget(t *testing.T) {
	entries := []ipc.NetworkEntry{{RequestBody: "tiny", Body: "small"}}
	applyBodyTruncation(entries, 1024)

	if entries[0].RequestBodyTruncated || entries[0].BodyTruncated {
		t.Error("no truncation flag should be set when bodies fit the budget")
	}
}

func TestFilterNetworkByText_MatchesRequestBody(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{URL: "https://api.example.com/login", RequestBody: `{"username":"grant"}`},
		{URL: "https://api.example.com/other", Body: `{"result":"ok"}`},
	}

	matched := filterNetworkByText(entries, "grant")
	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0].URL != "https://api.example.com/login" {
		t.Errorf("matched wrong entry: %s", matched[0].URL)
	}
}

func TestFilterNetworkByText_StillMatchesResponseBody(t *testing.T) {
	entries := []ipc.NetworkEntry{
		{URL: "https://api.example.com/login", RequestBody: `{"username":"grant"}`},
		{URL: "https://api.example.com/other", Body: `{"token":"abc123"}`},
	}

	matched := filterNetworkByText(entries, "abc123")
	if len(matched) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched))
	}
	if matched[0].URL != "https://api.example.com/other" {
		t.Errorf("matched wrong entry: %s", matched[0].URL)
	}
}
